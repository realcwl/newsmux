package collector_instances

import (
	"errors"
	"fmt"
	"strings"

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

type CustomizedSubSourceCrawler struct {
	Sink sink.CollectedDataSink
}

func (crawler CustomizedSubSourceCrawler) UpdateTitle(workingContext *working_context.CrawlerWorkingContext) error {
	titleSelector := *workingContext.SubSource.CustomizedCrawlerParamsForSubSource.TitleRelativeSelector
	if len(titleSelector) == 0 {
		return nil
	}
	workingContext.Result.Post.Title = workingContext.Element.DOM.Find(titleSelector).Text()
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	contentSelector := *workingContext.SubSource.CustomizedCrawlerParamsForSubSource.ContentRelativeSelector
	if len(contentSelector) == 0 {
		return nil
	}
	workingContext.Result.Post.Content = workingContext.Element.DOM.Find(contentSelector).Text()
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateExternalId(workingContext *working_context.CrawlerWorkingContext) error {
	contentSelector := *workingContext.SubSource.CustomizedCrawlerParamsForSubSource.ExternalIdRelativeSelector
	if len(contentSelector) == 0 {
		return nil
	}
	workingContext.Result.Post.Content = workingContext.Element.DOM.Find(contentSelector).Text()
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	contentSelector := *workingContext.SubSource.CustomizedCrawlerParamsForSubSource.ExternalIdRelativeSelector
	if len(contentSelector) == 0 {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		return nil
	}

	dateString := workingContext.Element.DOM.Find(contentSelector).Text()
	t, err := dateparse.ParseLocal(dateString)
	if err != nil {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
	} else {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.New(t)
	}
	return nil
}

// Dedup id in customized crawler is fixed logic, user don't have UI to modify it
func (crawler CustomizedSubSourceCrawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.Content)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

// For subsource customized crawler, we don't use subsource jquery selector to get subsource name, we use the one specified in task params instead
func (crawler CustomizedSubSourceCrawler) UpdateSubsource(workingContext *working_context.CrawlerWorkingContext) error {
	if workingContext.SubSource != nil {
		workingContext.Result.Post.SubSource.Name = workingContext.SubSource.Name
	} else {
		return fmt.Errorf("subsource is nil")
	}

	workingContext.Result.Post.SubSource.AvatarUrl = *workingContext.SubSource.AvatarUrl
	return nil
}

func (crawler CustomizedSubSourceCrawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.ImageUrls = []string{}
	imageSelector := *workingContext.SubSource.CustomizedCrawlerParamsForSubSource.ImageRelativeSelector
	selection := workingContext.Element.DOM.Find(imageSelector)
	for i := 0; i < selection.Length(); i++ {
		img := selection.Eq(i)
		imageUrl := img.AttrOr("src", "")
		parts := strings.Split(imageUrl, "?")
		imageUrl = parts[0]
		if len(imageUrl) == 0 {
			return errors.New("image DOM exist but src not found")
		}
		workingContext.Result.Post.ImageUrls = append(workingContext.Result.Post.ImageUrls, imageUrl)
	}
	return nil
}

func (crawler CustomizedSubSourceCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		crawler.UpdateTitle,
		crawler.UpdateContent,
		crawler.UpdateExternalId,
		crawler.UpdateGeneratedTime,
		crawler.UpdateSubsource,
		crawler.UpdateImageUrls,
		crawler.UpdateDedupId,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (crawler CustomizedSubSourceCrawler) GetBaseSelector(subsource *protocol.PanopticSubSource) (string, error) {
	return subsource.CustomizedCrawlerParamsForSubSource.BaseSelector, nil
}

func (crawler CustomizedSubSourceCrawler) GetCrawlUrl(subsource *protocol.PanopticSubSource) (string, error) {
	return subsource.CustomizedCrawlerParamsForSubSource.CrawlUrl, nil
}

func (crawler CustomizedSubSourceCrawler) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	metadata := task.TaskMetadata

	startUrl, err := crawler.GetCrawlUrl(subsource)
	if err != nil {
		collector.MarkAndLogCrawlError(task, err, "")
		return err
	}

	baseSelector, err := crawler.GetBaseSelector(subsource)
	if err != nil {
		collector.MarkAndLogCrawlError(task, err, "")
		return err
	}

	c := colly.NewCollector()
	// each crawled card(news) will go to this
	// for each page loaded, there are multiple calls into this func
	c.OnHTML(baseSelector, func(elem *colly.HTMLElement) {
		var err error

		workingContext := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, Element: elem, OriginUrl: startUrl,
			SubSource: subsource}
		if err = crawler.GetMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		sink.PushResultToSinkAndRecordInTaskMetadata(crawler.Sink, workingContext)
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
		for _, kv := range task.TaskParams.HeaderParams {
			r.Headers.Set(kv.Key, kv.Value)
		}
	})

	c.Visit(startUrl)

	return nil
}

func (crawler CustomizedSubSourceCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	collector.ParallelSubsourceApiCollect(task, crawler)
	collector.SetErrorBasedOnCounts(task, "customized subsource crawler")
}
