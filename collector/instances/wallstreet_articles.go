package collector_instances

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Luismorlan/newsmux/collector"
	sink "github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gocolly/colly"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	WallstreetArticleDateFormat = "2006-01-02T15:04:05.999-07:00"
)

type WallstreetArticleCollector struct {
	Sink sink.CollectedDataSink
}

func (w WallstreetArticleCollector) UpdateFileUrls(workingContext *working_context.ApiCollectorWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (w WallstreetArticleCollector) GetStartUri(subsource *protocol.PanopticSubSource) string {
	return fmt.Sprintf("https://wallstreetcn.com/news/%s", subsource.ExternalId)
}

func (w WallstreetArticleCollector) GetQueryPath() string {
	return `.article-entry`
}

func (w WallstreetArticleCollector) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.Content)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (w WallstreetArticleCollector) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	dateStr := workingContext.Element.DOM.Find(`.meta > time `).AttrOr("datetime", "")

	generatedTime, err := collector.ParseGenerateTime(dateStr, WallstreetArticleDateFormat, ChinaTimeZone, "wallstreet_article")

	if err != nil {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		return err
	}
	workingContext.Result.Post.ContentGeneratedAt = generatedTime
	return nil
}

func (w WallstreetArticleCollector) UpdateOriginUrl(workingContext *working_context.CrawlerWorkingContext) error {
	link := workingContext.Element.DOM.Find(`.container > a`).AttrOr("href", "")
	workingContext.Result.Post.OriginUrl = link
	return nil
}

func (w WallstreetArticleCollector) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	err := w.UpdateContent(workingContext)
	if err != nil {
		return err
	}

	err = w.UpdateImageUrls(workingContext)
	if err != nil {
		return err
	}

	err = w.UpdateTitle(workingContext)
	if err != nil {
		return err
	}

	err = w.UpdateOriginUrl(workingContext)
	if err != nil {
		return err
	}

	err = w.UpdateDedupId(workingContext)
	if err != nil {
		return err
	}

	err = w.UpdateGeneratedTime(workingContext)
	if err != nil {
		return err
	}

	return nil
}

func (w WallstreetArticleCollector) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Content = workingContext.Element.DOM.Find(`.container > .content`).Text()
	return nil
}

func (w WallstreetArticleCollector) UpdateTitle(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Title = workingContext.Element.DOM.Find(`.container > a > span`).Text()
	return nil
}

func (w WallstreetArticleCollector) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	imageUrl := strings.Split(workingContext.Element.DOM.Find(`img`).AttrOr(`src`, ``), "?")[0]
	workingContext.Result.Post.ImageUrls = []string{imageUrl}
	return nil
}

func (w WallstreetArticleCollector) CollectAndPublish(task *protocol.PanopticTask) {
	metadata := task.TaskMetadata
	metadata.ResultState = protocol.TaskMetadata_STATE_SUCCESS

	for _, subSource := range task.TaskParams.SubSources {
		c := colly.NewCollector()
		// each crawled card(news) will go to this
		// for each page loaded, there are multiple calls into this func

		c.OnHTML(w.GetQueryPath(), func(elem *colly.HTMLElement) {
			var err error
			workingContext := &working_context.CrawlerWorkingContext{
				SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, Element: elem, OriginUrl: w.GetStartUri(subSource), SubSource: subSource}
			collector.InitializeCrawlerResult(workingContext)
			if err = w.GetMessage(workingContext); err != nil {
				metadata.TotalMessageFailed++
				collector.LogHtmlParsingError(task, elem, err)
				return
			}
			if workingContext.Result == nil {
				return
			}
			if !workingContext.IntentionallySkipped {
				sink.PushResultToSinkAndRecordInTaskMetadata(w.Sink, workingContext)
			}
		})

		// Set error handler
		c.OnError(func(r *colly.Response, err error) {
			// todo: error should be put into metadata
			task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
			Logger.Log.WithFields(logrus.Fields{"source": "wallstreet_article"}).Error("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err, " path ", w.GetQueryPath())
		})

		c.OnScraped(func(_ *colly.Response) {
			// Set Fail/Success in task meta based on number of message succeeded
			collector.SetErrorBasedOnCounts(task, w.GetStartUri(subSource), fmt.Sprintf(" path: %s", w.GetQueryPath()))
		})

		c.Visit(w.GetStartUri(subSource))
	}
}
