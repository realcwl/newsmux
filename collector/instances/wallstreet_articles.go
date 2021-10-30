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
)

const ()

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
	md5, err := utils.TextToMd5Hash(workingContext.OriginUrl)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
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

	err = w.UpdateDedupId(workingContext)

	return nil
}

func (w WallstreetArticleCollector) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Content = workingContext.Element.Text
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
				SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, Element: elem, OriginUrl: w.GetStartUri(subSource)}
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
			Logger.Log.WithFields(logrus.Fields{"source": "jin10"}).Error("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err, " path ", w.GetQueryPath())
		})

		c.OnScraped(func(_ *colly.Response) {
			// Set Fail/Success in task meta based on number of message succeeded
			collector.SetErrorBasedOnCounts(task, w.GetStartUri(subSource), fmt.Sprintf(" path: %s", w.GetQueryPath()))
		})

		c.Visit(w.GetStartUri(subSource))

	}
}
