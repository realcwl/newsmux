package collector_instances

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Luismorlan/newsmux/collector"
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
	Jin10DateFormat = "20060102-15:04:05"
)

type Jin10Crawler struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (j Jin10Crawler) UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (j Jin10Crawler) UpdateNewsType(workingContext *working_context.CrawlerWorkingContext) error {
	selection := workingContext.Element.DOM.Find(".jin-flash-item")
	if len(selection.Nodes) == 0 {
		workingContext.NewsType = protocol.PanopticSubSource_UNSPECIFIED
		return errors.New("jin10 news item not found")
	}
	if selection.HasClass("is-important") {
		workingContext.NewsType = protocol.PanopticSubSource_KEYNEWS
		return nil
	}
	workingContext.NewsType = protocol.PanopticSubSource_FLASHNEWS
	return nil
}

// check if we should skip the message - ads for example
func (j Jin10Crawler) ShouldSkipMessage(workingContext *working_context.CrawlerWorkingContext, content string) bool {
	selection := workingContext.Element.DOM.Find(".jin-flash-item")
	// filter ads in importatn news
	if selection.HasClass("is-important") {
		lastDiv := selection.Find(".right-content > div ")
		if len(lastDiv.Children().Nodes) == 1 && lastDiv.Children().Nodes[0].Data == "b" {
			return true
		}
	}

	if workingContext.Task.TaskParams.GetJinshiTaskParams() != nil {
		for _, key := range workingContext.Task.TaskParams.GetJinshiTaskParams().SkipKeyWords {
			if strings.Contains(content, key) {
				return true
			}
		}
	}
	return false
}

func (j Jin10Crawler) UpdateContent(workingContext *working_context.CrawlerWorkingContext) error {
	var sb strings.Builder
	selection := workingContext.Element.DOM.Find(".right-content > div")
	if len(selection.Nodes) == 0 {
		return errors.New("jin10 news DOM not found")
	}
	selection.Children().Each(func(_ int, s *goquery.Selection) {
		if len(s.Nodes) > 0 && s.Nodes[0].Data == "br" {
			sb.WriteString(" ")
		}
		sb.WriteString(s.Text())
	})
	// goquery don't have a good way to get text without child elements'
	// remove children's text manually
	remove := selection.Children().Text()
	text := selection.Text()
	result := strings.Replace(text, remove, "", -1)
	sb.WriteString(result)

	content := sb.String()

	if j.ShouldSkipMessage(workingContext, content) {
		workingContext.SharedContext.IntentionallySkipped = true
		return nil
	}

	if len(content) == 0 {
		// empty content is likely to be economy stats (which we intend to skip)
		// in case it is because of other issues, we log and return
		collector.LogHtmlParsingError(workingContext.Task, workingContext.Element, errors.New("empty content (this msg is skipped)"))
		workingContext.SharedContext.IntentionallySkipped = true
		return nil
	}

	workingContext.Result.Post.Content = content
	return nil
}

func (j Jin10Crawler) UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error {
	id := workingContext.Element.DOM.AttrOr("id", "")
	timeText := workingContext.Element.DOM.Find(".item-time").Text()
	if len(id) <= 13 {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		return errors.New("jin10 news DOM id length is smaller than expected")
	}

	dateStr := id[5:13] + "-" + timeText
	generatedTime, err := collector.ParseGenerateTime(dateStr, Jin10DateFormat, ChinaTimeZone, "jin10")

	if err != nil {
		workingContext.Result.Post.ContentGeneratedAt = timestamppb.Now()
		return err
	}
	workingContext.Result.Post.ContentGeneratedAt = generatedTime
	return nil
}

func (j Jin10Crawler) UpdateExternalPostId(workingContext *working_context.CrawlerWorkingContext) error {
	id := workingContext.Element.DOM.AttrOr("id", "")
	if len(id) == 0 {
		return errors.New("can't get external post id for the news")
	}
	workingContext.ExternalPostId = id
	return nil
}

func (j Jin10Crawler) UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.ExternalPostId)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (c Jin10Crawler) UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error {
	// there is only one image can be in jin10
	selection := workingContext.Element.DOM.Find(".img-container > img")
	workingContext.Result.Post.ImageUrls = []string{}
	if len(selection.Nodes) == 0 {
		return nil
	}

	imageUrl := selection.AttrOr("data-src", "")
	if len(imageUrl) == 0 {
		return errors.New("image DOM exist but src not found")
	}
	// initialize with original image url as a fallback if any error with S3
	workingContext.Result.Post.ImageUrls = []string{imageUrl}

	key, err := c.ImageStore.FetchAndStore(imageUrl, "")
	if err != nil {
		Logger.Log.WithFields(logrus.Fields{"source": "jin10"}).
			Errorln("fail to get jin10 image, err:", err, "url", imageUrl)
		return utils.ImmediatePrintError(err)
	}
	s3Url := c.ImageStore.GetUrlFromKey(key)
	workingContext.Result.Post.ImageUrls = []string{s3Url}
	return nil
}

func (j Jin10Crawler) UpdateSubsourceName(workingContext *working_context.CrawlerWorkingContext) error {
	workingContext.Result.Post.SubSource.Name = collector.SubsourceTypeToName(workingContext.NewsType)
	return nil
}

func ShouldSplitJin10Content(title string) bool {
	NoSplitWords := []string{"金十图示", "行情", "报告"}
	return !utils.ContainsString(NoSplitWords, title)
}

func (j Jin10Crawler) MaybeSplitTitleOutOfContent(
	workingContext *working_context.CrawlerWorkingContext) {
	content := workingContext.Result.Post.Content
	re := regexp.MustCompile(`【.*】`)
	match := re.FindStringSubmatch(content)
	if len(match) != 1 {
		return
	}

	if !ShouldSplitJin10Content(content) {
		return
	}
	trimmedContent := strings.ReplaceAll(content, match[0], "")
	workingContext.Result.Post.Content = trimmedContent
	workingContext.Result.Post.Title = strings.NewReplacer("【", "", "】", "").Replace(match[0])
}

func (j Jin10Crawler) GetMessage(workingContext *working_context.CrawlerWorkingContext) error {
	collector.InitializeCrawlerResult(workingContext)

	err := j.UpdateContent(workingContext)
	if err != nil {
		return err
	}
	j.MaybeSplitTitleOutOfContent(workingContext)

	err = j.UpdateExternalPostId(workingContext)
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

	err = j.UpdateImageUrls(workingContext)
	if err != nil {
		return err
	}

	err = j.UpdateSubsourceName(workingContext)
	if err != nil {
		return err
	}

	return nil
}

func (j Jin10Crawler) GetQueryPath() string {
	return `#jin_flash_list > .jin-flash-item-container`
}

func (j Jin10Crawler) GetStartUri() string {
	return "https://www.jin10.com/index.html"
}

// todo: mock http response and test end to end Collect()
func (j Jin10Crawler) CollectAndPublish(task *protocol.PanopticTask) {
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
