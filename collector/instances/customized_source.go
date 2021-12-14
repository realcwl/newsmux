package collector_instances

import (
	"fmt"

	"github.com/araddon/dateparse"
	"github.com/gocolly/colly"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

type CustomizedSourceCrawler struct {
	Sink sink.CollectedDataSink
}

func (j CustomizedSourceCrawler) UpdateTitle(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Title = collector.CustomizedCrawlerExtractPlainText(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().TitleRelativeSelector, workingContext.Element, "")
	return nil
}

func (j CustomizedSourceCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Content = collector.CustomizedCrawlerExtractPlainText(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().ContentRelativeSelector, workingContext.Element, "")
	return nil
}

func (j CustomizedSourceCrawler) UpdateExternalId(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.ExternalId = collector.CustomizedCrawlerExtractPlainText(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().ExternalIdRelativeSelector, workingContext.Element, "")
	return nil
}

func (j CustomizedSourceCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	dateString := collector.CustomizedCrawlerExtractPlainText(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().TimeRelativeSelector, workingContext.Element, "")
	t, err := dateparse.ParseLocal(dateString)
	if err != nil {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
	} else {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(t)
	}
	return nil
}

// Dedup id in customized crawler is fixed logic, user don't have UI to modify it
func (j CustomizedSourceCrawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.Content)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (j CustomizedSourceCrawler) UpdateSubsource(workingContext *working_context.CrawlerWorkingContext) error {
	if workingContext.SubSource != nil {
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name
	} else {
		return fmt.Errorf("subsource is nil")
	}

	workingContext.Result.Post.SubSource.AvatarUrl = *workingContext.SubSource.AvatarUrl
	return nil
}

func (j CustomizedSourceCrawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.ImageUrls = collector.CustomizedCrawlerExtractMultiAttribute(workingContext.Task.TaskParams.
		GetCustomizedSourceCrawlerTaskParams().ImageRelativeSelector, workingContext.Element, "src")
	return nil
}

func (j CustomizedSourceCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		j.UpdateTitle,
		j.UpdateContent,
		j.UpdateExternalId,
		j.UpdateGeneratedTime,
		j.UpdateSubsource,
		j.UpdateImageUrls,
		j.UpdateDedupId,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j CustomizedSourceCrawler) GetBaseSelector(task *protocol.PanopticTask) (string, error) {
	return task.TaskParams.GetCustomizedSourceCrawlerTaskParams().BaseSelector, nil
}

func (j CustomizedSourceCrawler) GetCrawlUrl(task *protocol.PanopticTask) (string, error) {
	return task.TaskParams.GetCustomizedSourceCrawlerTaskParams().CrawlUrl, nil
}

func (j CustomizedSourceCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	metadata := task.TaskMetadata

	startUrl, err := j.GetCrawlUrl(task)
	if err != nil {
		collector.MarkAndLogCrawlError(task, err, "")
		return
	}

	baseSelector, err := j.GetBaseSelector(task)
	if err != nil {
		collector.MarkAndLogCrawlError(task, err, "")
		return
	}

	if len(task.TaskParams.SubSources) != 1 {
		collector.MarkAndLogCrawlError(task, err, "Source level customized crawler should have exact 1 subsource ")
		return
	}

	c := colly.NewCollector()
	// each crawled card(news) will go to this
	// for each page loaded, there are multiple calls into this func
	c.OnHTML(baseSelector, func(elem *colly.HTMLElement) {
		var err error

		workingContext := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, SubSource: task.TaskParams.SubSources[0], Element: elem, OriginUrl: startUrl}
		if err = j.GetMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		sink.PushResultToSinkAndRecordInTaskMetadata(j.Sink, workingContext)
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.Log.WithFields(logrus.Fields{"source": "customized"}).Error("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err, " path ", baseSelector)
	})

	c.OnScraped(func(_ *colly.Response) {
		// Set Fail/Success in task meta based on number of message succeeded
		collector.SetErrorBasedOnCounts(task, startUrl, fmt.Sprintf(" path: %s", baseSelector))
	})

	c.OnRequest(func(r *colly.Request) {
		if len(task.TaskParams.HeaderParams) == 0 {
			// to avoid http 418
			task.TaskParams.HeaderParams = collector.GetDefautlCrawlerHeader()
		}
		for _, kv := range task.TaskParams.HeaderParams {
			r.Headers.Set(kv.Key, kv.Value)
		}
	})

	c.Visit(startUrl)
}
