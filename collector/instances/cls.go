package collector_instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Luismorlan/newsmux/collector"
	sink "github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ClsArticleCollector struct {
	Sink sink.CollectedDataSink
}

type ClsApiResponseItem struct {
	ID           int         `json:"id"`
	Ctime        int         `json:"ctime"`
	Title        string      `json:"title"`
	Brief        string      `json:"brief"`
	Image        string      `json:"image"`
	Level        string      `json:"level"`
	ArticleTag   interface{} `json:"article_tag"`
	ExternalLink string      `json:"external_link"`
	IsAd         int         `json:"is_ad"`
	AudioURL     string      `json:"audio_url"`
}

type ClsApiResponse struct {
	Errno int                  `json:"errno"`
	Data  []ClsApiResponseItem `json:"data"`
}

func (cls ClsArticleCollector) UpdateFileUrls(workingContext *working_context.ApiCollectorWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (cls ClsArticleCollector) GetStartUri(subsource *protocol.PanopticSubSource) string {
	return fmt.Sprintf("https://www.cls.cn/v3/depth/list/%s", subsource.ExternalId)
}

func (cls ClsArticleCollector) UpdateDedupId(post *protocol.CrawlerMessage_CrawledPost) error {
	md5, err := utils.TextToMd5Hash(post.OriginUrl)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	post.DeduplicateId = md5
	return nil
}

func (cls ClsArticleCollector) UpdateResult(wc *working_context.ApiCollectorWorkingContext) error {
	item := wc.ApiResponseItem.(ClsApiResponseItem)
	post := wc.Result.Post
	// 2021-10-20T11:39:55.092+0800
	generatedTime := time.Unix(int64(item.Ctime), 0)
	post.ContentGeneratedAt = timestamppb.New(generatedTime)

	post.OriginUrl = fmt.Sprintf("https://www.cls.cn/detail/%d", item.ID)

	post.SubSource.Name = wc.SubSource.Name
	post.SubSource.AvatarUrl = "https://newsfeed-logo.s3.us-west-1.amazonaws.com/cls.png"
	post.SubSource.ExternalId = wc.SubSource.ExternalId

	post.Content = item.Brief
	post.Title = item.Title

	err := cls.UpdateDedupId(post)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	return nil
}

func (cls ClsArticleCollector) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	client := collector.NewHttpClientFromTaskParams(task)
	if client == nil {
		return errors.New("no client")
	}
	url := cls.GetStartUri(subsource)
	resp, err := client.Get(url)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	res := &ClsApiResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	if res.Errno != 0 {
		return utils.ImmediatePrintError(errors.New(fmt.Sprintf("response not success: %+v", res)))
	}

	for _, item := range res.Data {
		// working context for each message
		workingContext := &working_context.ApiCollectorWorkingContext{
			SharedContext: working_context.SharedContext{
				Task:                 task,
				Result:               &protocol.CrawlerMessage{},
				IntentionallySkipped: false,
			},
			ApiUrl:          url,
			SubSource:       subsource,
			ApiResponseItem: item,
		}
		collector.InitializeApiCollectorResult(workingContext)
		err := cls.UpdateResult(workingContext)
		if err != nil {
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		}

		if workingContext.SharedContext.Result != nil {
			sink.PushResultToSinkAndRecordInTaskMetadata(cls.Sink, workingContext)
		}
	}
	collector.SetErrorBasedOnCounts(task, url, fmt.Sprintf("subsource: %s, body: %s", subsource.Name, string(body)))
	return nil
}

func (cls ClsArticleCollector) CollectAndPublish(task *protocol.PanopticTask) {
	collector.ParallelSubsourceApiCollect(task, cls)
}
