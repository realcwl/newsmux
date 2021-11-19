package collector_instances

import (
	"errors"
	"fmt"
	"regexp"

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

type GelonghuiCrawler struct {
	Sink sink.CollectedDataSink
}

func (glh GelonghuiCrawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.ImageUrls = []string{}
	// image is not in html, it is added by javascript
	// selection := workingContext.Element.DOM.Find(".live-data-item__img")
	// for i := 0; i < selection.Length(); i++ {
	// 	img := selection.Eq(i)
	// 	imageUrl := img.AttrOr("src", "")
	// 	workingContext.Result.Post.ImageUrls = append(workingContext.Result.Post.ImageUrls, imageUrl)
	// }
	return nil
}

func (glh GelonghuiCrawler) UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (glh GelonghuiCrawler) UpdateNewsType(workingContext *working_context.CrawlerWorkingContext) error {
	if workingContext.Element.DOM.HasClass("data-red") {
		workingContext.NewsType = protocol.PanopticSubSource_KEYNEWS
	} else {
		workingContext.NewsType = protocol.PanopticSubSource_FLASHNEWS
	}

	return nil
}

func (glh GelonghuiCrawler) cleanContent(workingContext *working_context.CrawlerWorkingContext) error {
	re, err := regexp.Compile(`格隆汇\d+月\d+日丨`)
	if err != nil {
		return err
	}
	workingContext.Result.Post.Content = re.ReplaceAllString(workingContext.Result.Post.Content, "")
	return nil
}

func (glh GelonghuiCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Title = workingContext.Element.DOM.Find(".desc-title").Text()
	workingContext.Result.Post.Content = workingContext.Element.DOM.Find(".desc").Text()

	glh.cleanContent(workingContext)
	return nil
}

func (glh GelonghuiCrawler) CheckAds(workingContext *working_context.CrawlerWorkingContext) error {
	if workingContext.Element.DOM.Find(".live-data-item__interpretation").Length() > 0 {
		workingContext.IntentionallySkipped = true
	}
	return nil
}

func (glh GelonghuiCrawler) UpdateTags(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Tags = []string{}
	selection := workingContext.Element.DOM.Find(".live-data-item__footer--subject")
	for i := 0; i < selection.Length(); i++ {
		tag := selection.Eq(i)
		workingContext.Result.Post.Tags = append(workingContext.Result.Post.Tags, tag.Text())
	}
	return nil
}

func (glh GelonghuiCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
	return nil
}

func (glh GelonghuiCrawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.Content)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (glh GelonghuiCrawler) UpdateSubsourceName(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.Name = collector.SubsourceTypeToName(workingContext.NewsType)
	return nil
}

func (glh GelonghuiCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		glh.UpdateContent,
		glh.UpdateImageUrls,
		glh.UpdateTags,
		glh.UpdateDedupId,
		glh.UpdateNewsType,
		glh.UpdateGeneratedTime,
		glh.UpdateSubsourceName,
		glh.CheckAds,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (glh GelonghuiCrawler) GetQueryPath() string {
	return `.live-data-item`
}

func (glh GelonghuiCrawler) GetStartUri() string {
	return "https://www.gelonghui.com/live"
}

func (glh GelonghuiCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	metadata := task.TaskMetadata

	c := colly.NewCollector()
	// each crawled card(news) will go to this
	// for each page loaded, there are multiple calls into this func
	c.OnHTML(glh.GetQueryPath(), func(elem *colly.HTMLElement) {
		var err error
		workingContext := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: task, IntentionallySkipped: false}, Element: elem, OriginUrl: glh.GetStartUri()}
		if err = glh.GetMessage(workingContext); err != nil {
			metadata.TotalMessageFailed++
			collector.LogHtmlParsingError(task, elem, err)
			return
		}
		sink.PushResultToSinkAndRecordInTaskMetadata(glh.Sink, workingContext)
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.Log.WithFields(logrus.Fields{"source": "glh"}).Error("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err, " path ", glh.GetQueryPath())
	})

	c.OnScraped(func(_ *colly.Response) {
		// Set Fail/Success in task meta based on number of message succeeded
		collector.SetErrorBasedOnCounts(task, glh.GetStartUri(), fmt.Sprintf(" path: %s", glh.GetQueryPath()))
	})

	c.OnRequest(func(r *colly.Request) {
		for _, kv := range task.TaskParams.HeaderParams {
			r.Headers.Set(kv.Key, kv.Value)
		}
	})

	c.Visit(glh.GetStartUri())
}
