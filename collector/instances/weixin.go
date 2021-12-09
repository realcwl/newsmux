package collector_instances

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/Luismorlan/newsmux/collector"
	clients "github.com/Luismorlan/newsmux/collector/clients"
	"github.com/Luismorlan/newsmux/collector/file_store"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

func GetWeixinS3ImageStore(t *protocol.PanopticTask, isProd bool) (*file_store.S3FileStore, error) {
	bucketName := file_store.TestS3Bucket
	if isProd {
		bucketName = file_store.ProdS3FileBucket
	}
	zsxqFileStore, err := file_store.NewS3FileStore(bucketName)
	if err != nil {
		return nil, err
	}
	zsxqFileStore.SetCustomizeFileExtFunc(GetWeixinImgExtMethod())
	return zsxqFileStore, nil
}

func GetWeixinImgExtMethod() file_store.CustomizeFileExtFuncType {
	return func(url string, fileName string) string {
		var re = regexp.MustCompile(`wx_fmt\%3D(.*)(\%22)*`)
		found := re.FindStringSubmatch(url)
		if len(found) == 0 {
			Logger.Log.WithFields(logrus.Fields{"source": "weixin"}).
				Errorln("Can't find image extend name from src , image url = ", url)
			return ""
		}
		str := found[0]
		str = strings.Replace(str, "wx_fmt%3D", "", -1)
		str = strings.Replace(str, "%22", "", -1)
		return "." + str
	}
}

const (
	WeixinArticleDateFormat = time.RFC1123
)

type WeixinArticleRssCollector struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (w WeixinArticleRssCollector) UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (w WeixinArticleRssCollector) UpdateExternalPostId(workingContext *working_context.CrawlerWorkingContext) error {
	id := workingContext.Element.DOM.AttrOr("id", "")
	if len(id) == 0 {
		return errors.New("can't get external post id for the news")
	}
	workingContext.ExternalPostId = id
	return nil
}

func (w WeixinArticleRssCollector) UpdateDedupId(workingContext *working_context.RssCollectorWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Task.TaskParams.SourceId + workingContext.Result.Post.OriginUrl)
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (w WeixinArticleRssCollector) UpdateAvatarUrl(post *protocol.CrawlerMessage_CrawledPost, res *gofeed.Feed) error {
	avatarUrl := res.Image.URL
	if len(avatarUrl) == 0 {
		return nil
	}
	// initialize with original image url as a fallback if any error with S3
	post.SubSource.AvatarUrl = avatarUrl

	key, err := w.ImageStore.FetchAndStore(avatarUrl, "")
	if err != nil {
		Logger.Log.WithFields(logrus.Fields{"source": "weixin"}).
			Errorln("fail to get weixin user avatar image, err:", err, "url", avatarUrl)
		return utils.ImmediatePrintError(err)
	}
	s3Url := w.ImageStore.GetUrlFromKey(key)
	post.SubSource.AvatarUrl = s3Url
	return nil
}

func (w WeixinArticleRssCollector) ConstructUrl(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) string {
	return fmt.Sprintf("https://cdn.werss.weapp.design/api/v1/feeds/%s.xml",
		subsource.ExternalId,
	)
}

func (w WeixinArticleRssCollector) UpdateResultFromArticle(
	article *gofeed.Item,
	res *gofeed.Feed,
	workingContext *working_context.RssCollectorWorkingContext,
) error {
	post := workingContext.Result.Post
	// date
	generatedTime, err := time.Parse(WeixinArticleDateFormat, article.Published)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	post.ContentGeneratedAt = timestamppb.New(generatedTime)
	// avatar url
	post.SubSource.Name = workingContext.SubSource.Name
	// post.SubSource.AvatarUrl = res.Image.URL
	w.UpdateAvatarUrl(post, res)
	post.SubSource.ExternalId = workingContext.SubSource.ExternalId
	post.OriginUrl = article.Link
	post.Title = article.Title

	post.Content = "点击右上角时间进入正文"

	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	err = w.UpdateDedupId(workingContext)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	fmt.Println("YZ test", collector.PrettyPrint(post))

	return nil
}

func (w WeixinArticleRssCollector) CollectOneSubsourceOnePage(
	task *protocol.PanopticTask,
	subsource *protocol.PanopticSubSource,
) error {
	client := clients.NewHttpClientFromTaskParams(task)
	url := w.ConstructUrl(task, subsource)
	resp, err := client.Get(url)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)

	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	for _, article := range feed.Items {
		// working context for each message
		workingContext := &working_context.RssCollectorWorkingContext{
			SharedContext:   working_context.SharedContext{Task: task, Result: &protocol.CrawlerMessage{}, IntentionallySkipped: false},
			RssUrl:          url,
			SubSource:       subsource,
			RssResponseItem: article,
		}
		collector.InitializeRssCollectorResult(workingContext)
		err := w.UpdateResultFromArticle(article, feed, workingContext)
		if err != nil {
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		}

		if workingContext.SharedContext.Result != nil {
			sink.PushResultToSinkAndRecordInTaskMetadata(w.Sink, workingContext)
		}
	}

	return nil
}

// Support configable multi-page API call
func (w WeixinArticleRssCollector) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	return w.CollectOneSubsourceOnePage(task, subsource)
}

func (w WeixinArticleRssCollector) CollectAndPublish(task *protocol.PanopticTask) {
	collector.ParallelSubsourceApiCollect(task, w)
	collector.SetErrorBasedOnCounts(task, "weixin")
}
