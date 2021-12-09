package collector_instances

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/clients"
	"github.com/Luismorlan/newsmux/collector/file_store"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	CaUsArticleDateFormat = "2006-01-02 15:04:05"
)

type CaUsArticleCrawler struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (j CaUsArticleCrawler) UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (j CaUsArticleCrawler) UpdateNewsType(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateNewsType not implemented, should not be called")
}

func (j CaUsArticleCrawler) UpdateArticleDom(workingContext *working_context.CrawlerWorkingContext) error {
	path := workingContext.Element.DOM.Find(".content_left > a").AttrOr("href", "")
	url := "https://caus.com" + path
	workingContext.Result.Post.OriginUrl = url

	client := clients.NewHttpClientFromTaskParams(workingContext.Task)
	resp, err := client.Get(url)
	if err != nil {
		collector.LogHtmlParsingError(workingContext.Task, workingContext.Element, err)
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		collector.LogHtmlParsingError(workingContext.Task, workingContext.Element, err)
		return err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		collector.LogHtmlParsingError(workingContext.Task, workingContext.Element, err)
		return err
	}
	workingContext.Element.DOM = doc.Selection.Find(".details-center > div")
	return nil
}

func (j CaUsArticleCrawler) UpdateTitle(workingContext *working_context.CrawlerWorkingContext) error {
	title := workingContext.Element.DOM.Find(".text_title").Text()
	workingContext.Result.Post.Title = title
	return nil
}

func (j CaUsArticleCrawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	txt := workingContext.Element.DOM.Find(".details-center .img-wrapper p").Text()
	workingContext.Result.Post.Content = txt
	return nil
}

func (j CaUsArticleCrawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	dateStr := workingContext.Element.DOM.Find(".tag .time").Text()
	generatedTime, err := collector.ParseGenerateTime(dateStr, CaUsArticleDateFormat, ChinaTimeZone, "caus article")
	if err != nil {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		return err
	}
	workingContext.Result.Post.ContentGeneratedAt = generatedTime
	return nil
}

func (j CaUsArticleCrawler) UpdateExternalPostId(workingContext *working_context.CrawlerWorkingContext) error {
	path := workingContext.Element.DOM.Find(".content_left > a").AttrOr("href", "")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return utils.ImmediatePrintError(errors.New("invalid url path for external id"))
	}
	workingContext.ExternalPostId = parts[1]
	return nil
}

func (j CaUsArticleCrawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.OriginUrl)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (c CaUsArticleCrawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	imgElem := workingContext.Element.DOM.Find(".content_right > a > div > div > img")
	if len(imgElem.Nodes) == 0 {
		return nil
	}
	imageUrl := imgElem.AttrOr("src", "")
	if len(imageUrl) == 0 {
		return errors.New("caus article image DOM exist but src not found")
	}
	// initialize with original image url as a fallback if any error with S3
	workingContext.Result.Post.ImageUrls = []string{imageUrl}

	key, err := c.ImageStore.FetchAndStore(imageUrl, "")
	if err != nil {
		Logger.Log.WithFields(logrus.Fields{"source": "caus_article"}).
			Errorln("fail to get caus_article image, err:", err, "url:", imageUrl)
		return utils.ImmediatePrintError(err)
	}
	s3Url := c.ImageStore.GetUrlFromKey(key)
	workingContext.Result.Post.ImageUrls = []string{s3Url}
	return nil
}

func (j CaUsArticleCrawler) UpdateSubsourceName(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.Name = "商业"
	return nil
}

func (j CaUsArticleCrawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	// Image is out side of the DOM
	err := j.UpdateImageUrls(workingContext)
	if err != nil {
		return err
	}

	err = j.UpdateExternalPostId(workingContext)
	if err != nil {
		return err
	}

	// working context DOM is changed to the article content after this call
	err = j.UpdateArticleDom(workingContext)
	if err != nil {
		return err
	}

	err = j.UpdateTitle(workingContext)
	if err != nil {
		return err
	}

	err = j.UpdateContent(workingContext)
	if err != nil {
		return err
	}

	err = j.UpdateDedupId(workingContext)
	if err != nil {
		return err
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

func (j CaUsArticleCrawler) GetQueryPath() string {
	return `.contentbox`
}

func (j CaUsArticleCrawler) GetStartUri() string {
	return "https://caus.com/home?id=0"
}

// todo: mock http response and test end to end Collect()
func (j CaUsArticleCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	metadata := task.TaskMetadata
	metadata.ResultState = protocol.TaskMetadata_STATE_SUCCESS

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
		sink.PushResultToSinkAndRecordInTaskMetadata(j.Sink, workingContext)
	})

	// Set error handler
	c.OnError(func(r *colly.Response, err error) {
		// todo: error should be put into metadata
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		Logger.Log.WithFields(logrus.Fields{"source": "jin10"}).Error("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err, " path ", j.GetQueryPath())
	})

	c.OnScraped(func(_ *colly.Response) {
		// Set Fail/Success in task meta based on number of message succeeded
		collector.SetErrorBasedOnCounts(task, j.GetStartUri(), fmt.Sprintf(" path: %s", j.GetQueryPath()))
	})

	c.Visit(j.GetStartUri())
}
