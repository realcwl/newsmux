package collector_instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	. "github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/PuerkitoBio/goquery"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ZsxqDateformat = "2006-01-02T15:04:05.999Z07:00"
)

// Should Construct With Collector Builder
type ZsxqApiCollector struct {
	Sink       CollectedDataSink
	FileStore  CollectedFileStore
	ImageStore CollectedFileStore
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
	Owner struct {
		AvatarURL string `json:"avatar_url"`
		Name      string `json:"name"`
		UserID    int64  `json:"user_id"`
	} `json:"owner"`
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
	CommentsCount int64  `json:"comments_count"`
	CreateTime    string `json:"create_time"`
	Digested      bool   `json:"digested"`
	Group         struct {
		GroupID int64  `json:"group_id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
	} `json:"group"`
	LatestLikes []struct {
		CreateTime string `json:"create_time"`
		Owner      struct {
			AvatarURL string `json:"avatar_url"`
			Name      string `json:"name"`
			UserID    int64  `json:"user_id"`
		} `json:"owner"`
	} `json:"latest_likes"`
	LikesCount   int64 `json:"likes_count"`
	ReadersCount int64 `json:"readers_count"`
	ReadingCount int64 `json:"reading_count"`
	RewardsCount int64 `json:"rewards_count"`
	ShowComments []struct {
		CommentID  int64  `json:"comment_id"`
		CreateTime string `json:"create_time"`
		LikesCount int64  `json:"likes_count"`
		Owner      struct {
			AvatarURL string `json:"avatar_url"`
			Name      string `json:"name"`
			UserID    int64  `json:"user_id"`
		} `json:"owner"`
		RewardsCount int64  `json:"rewards_count"`
		Text         string `json:"text"`
	} `json:"show_comments"`
	Sticky       bool     `json:"sticky"`
	Talk         ZsxqTalk `json:"talk"`
	TopicID      int64    `json:"topic_id"`
	Type         string   `json:"type"`
	UserSpecific struct {
		Liked      bool `json:"liked"`
		Subscribed bool `json:"subscribed"`
	} `json:"user_specific"`
	Question struct {
		Owner struct {
			UserID    int64  `json:"user_id"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"owner"`
		Questionee struct {
			UserID    int64  `json:"user_id"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"questionee"`
		Text      string `json:"text"`
		Expired   bool   `json:"expired"`
		Anonymous bool   `json:"anonymous"`
	} `json:"question,omitempty"`
	Answer struct {
		Owner struct {
			UserID    int64  `json:"user_id"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"owner"`
		Text string `json:"text"`
	} `json:"answer,omitempty"`
	Answered bool `json:"answered,omitempty"`
	Silenced bool `json:"silenced,omitempty"`
}

type ZsxqApiResponse struct {
	RespData struct {
		Topics []ZsxqTopic `json:"topics"`
	} `json:"resp_data"`
	Succeeded bool `json:"succeeded"`
}

func GetZsxqFileNameMethod() CustomizeFileNameFuncType {
	return func(url, fileName string) string {
		digest, err := utils.TextToMd5Hash(url)
		if err != nil {
			return ""
		}
		return digest + "/" + fileName
	}
}

func GetZsxqFileExtMethod() CustomizeFileExtFuncType {
	return func(url, fileName string) string {
		// return empty since fileName is already having extension
		return ""
	}
}

func (collector ZsxqApiCollector) UpdateFileUrls(workingContext *ApiCollectorWorkingContext) error {
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

func (collector ZsxqApiCollector) ConstructUrl(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource, paginationInfo *PaginationInfo) string {
	return fmt.Sprintf("https://api.zsxq.com/v2/groups/%s/topics?scope=all&count=20",
		subsource.ExternalId,
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

func (collector ZsxqApiCollector) UppdateImages(wc *ApiCollectorWorkingContext) error {
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

func (collector ZsxqApiCollector) UpdateResult(wc *ApiCollectorWorkingContext) error {
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
		post.Content = CleanWeiboContent(item.Talk.Text)
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
	paginationInfo *PaginationInfo,
) error {
	header := http.Header{}
	for _, h := range task.TaskParams.HeaderParams {
		header[h.Key] = []string{h.Value}
	}
	cookies := []http.Cookie{}
	for _, c := range task.TaskParams.Cookies {
		cookies = append(cookies, http.Cookie{Name: c.Key, Value: c.Value})
	}

	fmt.Println("111111111")
	client := NewHttpClient(header, cookies)
	url := collector.ConstructUrl(task, subsource, paginationInfo)
	resp, err := client.Get(url)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	fmt.Println("222222", resp)
	body, err := io.ReadAll(resp.Body)
	res := &ZsxqApiResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	fmt.Println("33333333")
	if res.Succeeded != true {
		return errors.New(fmt.Sprintf("response not success: %v", res))
	}

	for _, topic := range res.RespData.Topics {
		// working context for each message
		workingContext := &ApiCollectorWorkingContext{
			Task:            task,
			PaginationInfo:  paginationInfo,
			ApiUrl:          url,
			Result:          &protocol.CrawlerMessage{},
			Subsource:       subsource,
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
		Logger.Log.Debug(workingContext.Result.Post.Content)
	}

	SetErrorBasedOnCounts(task, url, fmt.Sprintf("subsource: %s, body: %s", subsource.Name, string(body)))
	return nil
}

// zsxq is not paginated
func (collector ZsxqApiCollector) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	return collector.CollectOneSubsourceOnePage(task, subsource, &PaginationInfo{})
}

func (collector ZsxqApiCollector) CollectAndPublish(task *protocol.PanopticTask) {
	ParallelSubsourceApiCollect(task, collector)
}
