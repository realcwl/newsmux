package collector_instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	. "github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/file_store"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ZsxqDateformat = "2006-01-02T15:04:05.999Z07:00"
)

// Should Construct With Collector Builder
type ZsxqApiCollector struct {
	Sink       sink.CollectedDataSink
	FileStore  file_store.CollectedFileStore
	ImageStore file_store.CollectedFileStore
}

type ZsxqTalk struct {
	Images []struct {
		ImageID int64 `json:"image_id"`
		Large   struct {
			Height int64  `json:"height"`
			URL    string `json:"url"`
			Width  int64  `json:"width"`
		} `json:"large"`
		Original struct {
			Height int64  `json:"height"`
			Size   int64  `json:"size"`
			URL    string `json:"url"`
			Width  int64  `json:"width"`
		} `json:"original"`
		Thumbnail struct {
			Height int64  `json:"height"`
			URL    string `json:"url"`
			Width  int64  `json:"width"`
		} `json:"thumbnail"`
		Type string `json:"type"`
	} `json:"images"`
	Text  string `json:"text"`
	Files []struct {
		CreateTime    string `json:"create_time"`
		DownloadCount int64  `json:"download_count"`
		Duration      int64  `json:"duration"`
		FileID        int64  `json:"file_id"`
		Hash          string `json:"hash"`
		Name          string `json:"name"`
		Size          int64  `json:"size"`
	} `json:"files"`
}

type ZsxqTopic struct {
	CreateTime string `json:"create_time"`
	Group      struct {
		GroupID int64  `json:"group_id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
	} `json:"group"`
	Talk     ZsxqTalk `json:"talk"`
	TopicID  int64    `json:"topic_id"`
	Type     string   `json:"type"`
	Question struct {
		Text string `json:"text"`
	} `json:"question,omitempty"`
	Answer struct {
		Text string `json:"text"`
	} `json:"answer,omitempty"`
}

type ZsxqApiResponse struct {
	RespData struct {
		Topics []ZsxqTopic `json:"topics"`
	} `json:"resp_data"`
	Succeeded bool `json:"succeeded"`
}

type ZsxqFileDownloadApiResponse struct {
	Succeeded bool `json:"succeeded"`
	Code      int  `json:"ocde"`
	RespData  struct {
		DownloadURL string `json:"download_url"`
	} `json:"resp_data"`
}

func GetZsxqS3FileStore(t *protocol.PanopticTask, isProd bool) (*file_store.S3FileStore, error) {
	bucketName := file_store.TestS3Bucket
	if isProd {
		bucketName = file_store.ProdS3FileBucket
	}
	zsxqFileStore, err := file_store.NewS3FileStore(bucketName)
	if err != nil {
		return nil, err
	}
	zsxqFileStore.SetCustomizeFileExtFunc(GetZsxqFileExtMethod())
	zsxqFileStore.SetCustomizeFileNameFunc(GetZsxqFileNameMethod())
	zsxqFileStore.SetCustomizeUploadedUrlFunc(GetZsxqFileUrlMethod(isProd))
	zsxqFileStore.SetProcessUrlBeforeFetchFunc(GetZsxqFileDownloadUrlTransform(t))

	return zsxqFileStore, nil
}

func GetZsxqFileDownloadUrlTransform(task *protocol.PanopticTask) file_store.ProcessUrlBeforeFetchFuncType {
	return func(url string) string {
		client := NewHttpClientFromTaskParams(task)
		resp, err := client.Get(url)
		if err != nil {
			utils.ImmediatePrintError(err)
			return ""
		}
		body, err := io.ReadAll(resp.Body)

		res := &ZsxqFileDownloadApiResponse{}
		err = json.Unmarshal(body, res)
		if err != nil {
			utils.ImmediatePrintError(err)
			return ""
		}

		if !res.Succeeded {
			utils.ImmediatePrintError(errors.New(fmt.Sprintf("response from url %s not success: %+v", url, res)))
			return ""
		}

		return res.RespData.DownloadURL
	}
}

func GetZsxqFileUrlMethod(isProd bool) file_store.CustomizeUploadedUrlType {
	if isProd {
		return func(key string) string {
			return fmt.Sprintf("https://%s.s3.us-west-1.amazonaws.com/%s", file_store.ProdS3FileBucket, key)
		}
	} else {
		return func(key string) string {
			return fmt.Sprintf("https://%s.s3.us-west-1.amazonaws.com/%s", file_store.TestS3Bucket, key)
		}
	}
}

func GetZsxqFileNameMethod() file_store.CustomizeFileNameFuncType {
	return func(url, fileName string) string {
		digest, err := utils.TextToMd5Hash(url)
		if err != nil {
			return ""
		}
		return digest + "/" + fileName
	}
}

func GetZsxqFileExtMethod() file_store.CustomizeFileExtFuncType {
	return func(url, fileName string) string {
		// return empty since fileName is already having extension
		return ""
	}
}

func (collector ZsxqApiCollector) UpdateFileUrls(workingContext *working_context.ApiCollectorWorkingContext) error {
	// item *ZsxqTopic, post *protocol.CrawlerMessage_CrawledPost
	workingContext.Result.Post.FilesUrls = []string{}
	// workingContext.
	item := workingContext.ApiResponseItem.(ZsxqTopic)
	for _, file := range item.Talk.Files {
		url := fmt.Sprintf("https://api.zsxq.com/v2/files/%d/download_url", file.FileID)
		key, err := collector.FileStore.FetchAndStore(url, file.Name)
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		s3Url := collector.FileStore.GetUrlFromKey(key)
		workingContext.Result.Post.FilesUrls = append(workingContext.Result.Post.FilesUrls, s3Url)
	}
	return nil
}

func (collector ZsxqApiCollector) ConstructUrl(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource, paginationInfo *working_context.PaginationInfo) string {
	return fmt.Sprintf("https://api.zsxq.com/v2/groups/%s/topics?scope=all&count=%d",
		subsource.ExternalId,
		task.TaskParams.GetZsxqTaskParams().CountPerRequest,
	)
}

func (collector ZsxqApiCollector) UpdateDedupId(post *protocol.CrawlerMessage_CrawledPost) error {
	var sb strings.Builder
	sb.WriteString(post.Content)
	for _, file := range post.FilesUrls {
		sb.WriteString(file)
	}
	md5, err := utils.TextToMd5Hash(sb.String())
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	post.DeduplicateId = md5
	return nil
}

func (collector ZsxqApiCollector) UppdateImages(wc *working_context.ApiCollectorWorkingContext) error {
	item := wc.ApiResponseItem.(ZsxqTopic)
	wc.Result.Post.ImageUrls = []string{}
	for _, pic := range item.Talk.Images {
		key, err := collector.ImageStore.FetchAndStore(pic.Large.URL, fmt.Sprintf("%d.%s", pic.ImageID, pic.Type))
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		s3Url := collector.ImageStore.GetUrlFromKey(key)
		wc.Result.Post.ImageUrls = append(wc.Result.Post.ImageUrls, s3Url)
	}
	return nil
}

func (collector ZsxqApiCollector) UpdateResult(wc *working_context.ApiCollectorWorkingContext) error {
	item := wc.ApiResponseItem.(ZsxqTopic)
	post := wc.Result.Post
	// 2021-10-20T11:39:55.092+0800
	generatedTime, err := time.Parse(ZsxqDateformat, item.CreateTime)
	post.ContentGeneratedAt = timestamppb.New(generatedTime)

	post.OriginUrl = fmt.Sprintf("https://wx.zsxq.com/dweb2/index/group/%d", item.Group.GroupID)

	post.SubSource.Name = item.Group.Name
	post.SubSource.AvatarUrl = "https://newsfeed-logo.s3.us-west-1.amazonaws.com/zsxq.png"
	post.SubSource.ExternalId = fmt.Sprint(item.Group.GroupID)

	if item.Type == "q&a" {
		post.Content = item.Question.Text + " " + item.Answer.Text
	} else if item.Type == "talk" {
		post.Content = LineBreakerToSpace(item.Talk.Text)
		reader := strings.NewReader(post.Content)
		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			return utils.ImmediatePrintError(
				errors.New(fmt.Sprintf("fail to convert full rich-html text to node: %v", post.Content)))
		}
		// goquery Text() will not replace br with newline
		// to keep consistent with prod crawler, we need to
		// add newline
		doc.Find("br").AfterHtml("\n")
		post.Content = doc.Text()
	}

	collector.UppdateImages(wc)

	collector.UpdateFileUrls(wc)

	err = collector.UpdateDedupId(post)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	return nil
}

func (collector ZsxqApiCollector) CollectOneSubsourceOnePage(
	task *protocol.PanopticTask,
	subsource *protocol.PanopticSubSource,
	paginationInfo *working_context.PaginationInfo,
) error {
	client := NewHttpClientFromTaskParams(task)
	url := collector.ConstructUrl(task, subsource, paginationInfo)
	resp, err := client.Get(url)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	body, err := io.ReadAll(resp.Body)
	res := &ZsxqApiResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	if !res.Succeeded {
		return utils.ImmediatePrintError(errors.New(fmt.Sprintf("response not success: %+v", res)))
	}

	for _, topic := range res.RespData.Topics {
		// working context for each message
		workingContext := &working_context.ApiCollectorWorkingContext{
			SharedContext: working_context.SharedContext{
				Task:                 task,
				Result:               &protocol.CrawlerMessage{},
				IntentionallySkipped: false,
			},
			PaginationInfo:  paginationInfo,
			ApiUrl:          url,
			SubSource:       subsource,
			NewsType:        protocol.PanopticSubSource_UNSPECIFIED,
			ApiResponseItem: topic,
		}
		InitializeApiCollectorResult(workingContext)
		err := collector.UpdateResult(workingContext)
		if err != nil {
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		} else if err = collector.Sink.Push(workingContext.Result); err != nil {
			task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		}
		task.TaskMetadata.TotalMessageCollected++
		Logger.Log.WithFields(logrus.Fields{"source": "zsxq"}).Debug(workingContext.Result.Post.Content)
	}

	SetErrorBasedOnCounts(task, url, fmt.Sprintf("subsource: %s, body: %s", subsource.Name, string(body)))
	return nil
}

// zsxq is not paginated
func (collector ZsxqApiCollector) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	return collector.CollectOneSubsourceOnePage(task, subsource, &working_context.PaginationInfo{})
}

func (collector ZsxqApiCollector) CollectAndPublish(task *protocol.PanopticTask) {
	ParallelSubsourceApiCollect(task, collector)
}
