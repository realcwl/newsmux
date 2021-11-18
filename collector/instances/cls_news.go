package collector_instances

import (
	"errors"
	"fmt"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gocolly/colly"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ClsNewsCrawler struct {
	Sink sink.CollectedDataSink
}

func (j ClsNewsCrawler) UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (j ClsNewsCrawler) UpdateNewsType(workingContext *working_context.CrawlerWorkingContext) error {
	selection := workingContext.Element.DOM.Find(":nth-child(2)")
	if len(selection.Nodes) == 0 {
		workingContext.NewsType = protocol.PanopticSubSource_UNSPECIFIED
		return errors.New("cls news item not found")
	}
	if selection.HasClass("c-de0422") {
		workingContext.NewsType = protocol.PanopticSubSource_KEYNEWS
		return nil
	}
	workingContext.NewsType = protocol.PanopticSubSource_FLASHNEWS
	return nil
}

func (j ClsNewsCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	selection := workingContext.Element.DOM.Find(":nth-child(2)")
	if len(selection.Nodes) == 0 {
		return errors.New("cls news DOM not found")
	}
	text := selection.Text()
	workingContext.Result.Post.Content = text
	return nil
}

func (j ClsNewsCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	// timeText := workingContext.Element.DOM.Find(".telegraph-time-box").Text()
	// 20:27
	workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
	return nil
}

func (j ClsNewsCrawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.Content)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (j ClsNewsCrawler) UpdateSubsourceName(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.Name = collector.SubsourceTypeToName(workingContext.NewsType)
	return nil
}

func (j ClsNewsCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	err := j.UpdateContent(workingContext)
	if err != nil {
		return err
	}

	err = j.UpdateDedupId(workingContext)
	if err != nil {
		return err
	}

	err = j.UpdateNewsType(workingContext)
	if err != nil {
		return err
	}

	if !collector.IsRequestedNewsType(workingContext.Task.TaskParams.SubSources, workingContext.NewsType) {
		workingContext.Result = nil
		return nil
	}

	err = j.UpdateGeneratedTime(workingContext)
	if err != nil {
		return err
	}

	err = j.UpdateSubsourceName(workingContext)
	if err != nil {
		return err
	}

	return nil
}

func (j ClsNewsCrawler) GetQueryPath() string {
	return `.telegraph-content-box`
}

func (j ClsNewsCrawler) GetStartUri() string {
	return "https://www.cls.cn/telegraph"
}

// todo: mock http response and test end to end Collect()
func (j ClsNewsCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	metadata := task.TaskMetadata

	c := colly.NewCollector()
	// each crawled card(news) will go to this
	// for each page loaded, there are multiple calls into this func
	c.OnHTML(j.GetQueryPath(), func(elem *colly.HTMLElement) {
		var err error
		workingContext := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, Element: elem, OriginUrl: j.GetStartUri()}
		if err = j.GetMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		if workingContext.Result == nil {
			return
		}
		if !workingContext.IntentionallySkipped {
			sink.PushResultToSinkAndRecordInTaskMetadata(j.Sink, workingContext)
		}
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.Log.WithFields(logrus.Fields{"source": "cls"}).Error("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err, " path ", j.GetQueryPath())
	})

	c.OnScraped(func(_ *colly.Response) {
		// Set Fail/Success in task meta based on number of message succeeded
		collector.SetErrorBasedOnCounts(task, j.GetStartUri(), fmt.Sprintf(" path: %s", j.GetQueryPath()))
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36")
	})

	c.Visit(j.GetStartUri())
}
