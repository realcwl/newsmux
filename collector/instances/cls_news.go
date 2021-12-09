package collector_instances

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/file_store"
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
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (cls ClsNewsCrawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.ImageUrls = []string{}
	selection := workingContext.Element.DOM.Find(".telegraph-images-box > img")

	// initialize with original image url as a fallback if any error with S3
	for i := 0; i < selection.Length(); i++ {
		img := selection.Eq(i)
		imageUrl := img.AttrOr("src", "")
		parts := strings.Split(imageUrl, "?")
		imageUrl = parts[0]
		if len(imageUrl) == 0 {
			Logger.Log.WithFields(logrus.Fields{"source": "cls_news"}).
				Errorln("image DOM exist but src not found at index ", i, " of selection")
			continue
		}
		workingContext.Result.Post.ImageUrls = append(workingContext.Result.Post.ImageUrls, imageUrl)
	}

	// replace each original image url with S3 url
	for idx, url := range workingContext.Result.Post.ImageUrls {
		key, err := cls.ImageStore.FetchAndStore(url, "")
		if err != nil {
			Logger.Log.WithFields(logrus.Fields{"source": "cls_news"}).
				Errorln("fail to get cls_news image, err:", err, "url", url)
			return utils.ImmediatePrintError(err)
		}
		s3Url := cls.ImageStore.GetUrlFromKey(key)
		workingContext.Result.Post.ImageUrls[idx] = s3Url
	}
	return nil
}

func (j ClsNewsCrawler) UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (j ClsNewsCrawler) UpdateNewsType(workingContext *working_context.CrawlerWorkingContext) error {
	s := workingContext.Element.DOM.Find(".telegraph-content-box")
	selection := s.Find(":nth-child(2)")
	if len(selection.Nodes) > 0 && selection.HasClass("c-de0422") {
		workingContext.NewsType = protocol.PanopticSubSource_KEYNEWS
	} else {
		workingContext.NewsType = protocol.PanopticSubSource_FLASHNEWS
	}

	if !collector.IsRequestedNewsType(workingContext.Task.TaskParams.SubSources, workingContext.NewsType) {
		workingContext.IntentionallySkipped = true
	}

	return nil
}

func (j ClsNewsCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Content = workingContext.Element.DOM.Find(".telegraph-content-box span:not(.telegraph-time-box)").Text()
	title_selection := workingContext.Element.DOM.Find(".telegraph-content-box span:not(.telegraph-time-box) > strong")
	if title_selection.Length() > 0 {
		replacer := strings.NewReplacer("【", "", "】", "")
		workingContext.Result.Post.Title = replacer.Replace(title_selection.Text())
		workingContext.Result.Post.Content = strings.ReplaceAll(workingContext.Result.Post.Content, title_selection.Text(), "")
	}
	return nil
}

func (j ClsNewsCrawler) UpdateTags(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.Tags = []string{}
	selection := workingContext.Element.DOM.Find(".label-item")
	for i := 0; i < selection.Length(); i++ {
		tag := selection.Eq(i)
		workingContext.Result.Post.Tags = append(workingContext.Result.Post.Tags, tag.Text())
	}
	return nil
}

func (j ClsNewsCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
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

func (j ClsNewsCrawler) UpdateVipSkip(workingContext *working_context.CrawlerWorkingContext) error {
	selection := workingContext.Element.DOM.Find(".telegraph-vip-box")
	if selection.Length() > 0 {
		workingContext.IntentionallySkipped = true
	}
	return nil
}

func (j ClsNewsCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	updaters := []func(workingContext *working_context.CrawlerWorkingContext) error{
		j.UpdateContent,
		j.UpdateImageUrls,
		j.UpdateTags,
		j.UpdateDedupId,
		j.UpdateNewsType,
		j.UpdateGeneratedTime,
		j.UpdateSubsourceName,
		j.UpdateVipSkip,
	}
	for _, updater := range updaters {
		err := updater(workingContext)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j ClsNewsCrawler) GetQueryPath() string {
	return `.telegraph-list`
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
		sink.PushResultToSinkAndRecordInTaskMetadata(j.Sink, workingContext)
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
		for _, kv := range task.TaskParams.HeaderParams {
			r.Headers.Set(kv.Key, kv.Value)
		}
	})

	c.Visit(j.GetStartUri())
}
