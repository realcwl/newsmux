package collector_instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/file_store"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	WeiboDateFormat = time.RubyDate
)

// Generated with tool: https://mholt.github.io/json-to-go/
type WeiboDetailApiResponse struct {
	Ok   int `json:"ok"`
	Data struct {
		Ok              int    `json:"ok"`
		LongTextContent string `json:"longTextContent"`
	} `json:"data"`
}

type MBlogUser struct {
	ID              int    `json:"id"`
	ScreenName      string `json:"screen_name"`
	ProfileImageURL string `json:"profile_image_url"`
	ProfileURL      string `json:"profile_url"`
	AvatarHd        string `json:"avatar_hd"`
}

type MBlog struct {
	CreatedAt       string     `json:"created_at"`
	ID              string     `json:"id"`
	Text            string     `json:"text"`
	User            *MBlogUser `json:"user"`
	RetweetedStatus *MBlog     `json:"retweeted_status"`
	IsLongText      bool       `json:"isLongText"`
	RawText         string     `json:"raw_text"`
	Pics            []struct {
		Pid   string `json:"pid"`
		URL   string `json:"url"`
		Size  string `json:"size"`
		Large struct {
			Size string `json:"size"`
			URL  string `json:"url"`
		} `json:"large"`
	} `json:"pics"`
	Bid string `json:"bid"`
}

type WeiboApiResponse struct {
	Ok   int `json:"ok"`
	Data struct {
		CardlistInfo struct {
			Total int `json:"total"`
			Page  int `json:"page"`
		} `json:"cardlistInfo"`
		Cards []struct {
			Mblog MBlog `json:"mBlog"`
		} `json:"cards"`
	} `json:"data"`
}

// Should Construct With Collector Builder
type WeiboApiCollector struct {
	Sink       sink.CollectedDataSink
	ImageStore file_store.CollectedFileStore
}

func (w WeiboApiCollector) UpdateFileUrls(workingContext *working_context.ApiCollectorWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (w WeiboApiCollector) ConstructUrl(
	task *protocol.PanopticTask, subsource *protocol.PanopticSubSource,
	paginationInfo *working_context.PaginationInfo) string {
	return fmt.Sprintf("https://m.weibo.cn/api/container/getIndex?uid=%s&type=uid&page=%s&containerid=107603%s",
		subsource.ExternalId,
		paginationInfo.NextPageId,
		subsource.ExternalId,
	)
}

func (w WeiboApiCollector) UpdateDedupId(post *protocol.CrawlerMessage_CrawledPost) error {
	md5, err := utils.TextToMd5Hash(post.Content)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	post.DeduplicateId = md5
	return nil
}

func (w WeiboApiCollector) GetFullText(url string) (string, error) {
	var client collector.HttpClient
	resp, err := client.Get(url)
	if err != nil {
		return "", utils.ImmediatePrintError(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", utils.ImmediatePrintError(err)
	}
	if strings.Contains(string(body), "打开微博客户端，查看全文") {
		return "", utils.ImmediatePrintError(errors.New("need to open weibo client app"))
	}

	res := &WeiboDetailApiResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return "", utils.ImmediatePrintError(err)
	}
	if res.Ok != 1 {
		return "", utils.ImmediatePrintError(fmt.Errorf("response not success: %v", res))
	}

	return collector.HtmlToText(res.Data.LongTextContent)
}

func (collector WeiboApiCollector) UppdateImages(mBlog *MBlog, post *protocol.CrawlerMessage_CrawledPost) error {
	post.ImageUrls = []string{}
	for _, pic := range mBlog.Pics {
		key, err := collector.ImageStore.FetchAndStore(pic.URL, "")
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		s3Url := collector.ImageStore.GetUrlFromKey(key)
		post.ImageUrls = append(post.ImageUrls, s3Url)
	}
	return nil
}

func (w WeiboApiCollector) UpdateResultFromMblog(mBlog *MBlog, post *protocol.CrawlerMessage_CrawledPost) error {
	generatedTime, err := time.Parse(WeiboDateFormat, mBlog.CreatedAt)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	post.ContentGeneratedAt = timestamppb.New(generatedTime)
	if mBlog.User == nil {
		post.SubSource.Name = "default"
	} else {
		post.SubSource.Name = mBlog.User.ScreenName
		post.SubSource.AvatarUrl = "https://weibo.com/" + fmt.Sprint(mBlog.User.ID) + "/" + mBlog.Bid
		post.SubSource.ExternalId = fmt.Sprint(mBlog.User.ID)
	}
	w.UppdateImages(mBlog, post)
	// overwrite task level url by post url
	post.OriginUrl = "https://m.weibo.cn/detail/" + mBlog.ID
	if strings.Contains(mBlog.Text, ">全文<") {
		allTextUrl := "https://m.weibo.cn/statuses/extend?id=" + mBlog.ID
		text, err := w.GetFullText(allTextUrl)
		if err != nil {
			// if can't get full text, use short one as fall-back
			post.Content = mBlog.Text
			// fallback instead of count as error
			utils.ImmediatePrintError(err)
		} else {
			post.Content = text
		}
	} else {
		post.Content = mBlog.Text
	}
	post.Content = collector.LineBreakerToSpace(post.Content)

	err = w.UpdateDedupId(post)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	if mBlog.RetweetedStatus != nil {
		fmt.Println("starting processing shared post weibo id:", mBlog.ID)
		sharedPost := &protocol.CrawlerMessage_CrawledPost{
			SubSource: &protocol.CrawledSubSource{
				SourceId: post.SubSource.SourceId,
			},
		}
		err = w.UpdateResultFromMblog(mBlog.RetweetedStatus, sharedPost)
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		post.SharedFromCrawledPost = sharedPost
	}

	return nil
}

func (w WeiboApiCollector) CollectOneSubsourceOnePage(
	task *protocol.PanopticTask,
	subsource *protocol.PanopticSubSource,
	paginationInfo *working_context.PaginationInfo,
) error {
	var client collector.HttpClient
	url := w.ConstructUrl(task, subsource, paginationInfo)
	resp, err := client.Get(url)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	res := &WeiboApiResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}
	if res.Ok != 1 {
		return fmt.Errorf("response not success: %v", res)
	}

	for _, card := range res.Data.Cards {
		// working context for each message
		workingContext := &working_context.ApiCollectorWorkingContext{
			SharedContext:  working_context.SharedContext{Task: task, Result: &protocol.CrawlerMessage{}, IntentionallySkipped: false},
			PaginationInfo: paginationInfo,
			ApiUrl:         url,
			SubSource:      subsource,
		}
		collector.InitializeApiCollectorResult(workingContext)
		mBlog := card.Mblog
		err := w.UpdateResultFromMblog(&mBlog, workingContext.Result.Post)
		if err != nil {
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		} else if err = w.Sink.Push(workingContext.Result); err != nil {
			task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
			task.TaskMetadata.TotalMessageFailed++
			return utils.ImmediatePrintError(err)
		}
		task.TaskMetadata.TotalMessageCollected++
		Logger.Log.WithFields(logrus.Fields{"source": "weibo"}).Debug(workingContext.Result.Post.Content)
	}

	// Set next page identifier
	paginationInfo.NextPageId = fmt.Sprint(res.Data.CardlistInfo.Page)
	collector.SetErrorBasedOnCounts(task, url, fmt.Sprintf("subsource: %s, body: %s", subsource.Name, string(body)))
	return nil
}

// Support configable multi-page API call
func (w WeiboApiCollector) CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error {
	// NextPageId is set from API
	paginationInfo := working_context.PaginationInfo{
		CurrentPageCount: 1,
		NextPageId:       "1",
	}

	for {
		err := w.CollectOneSubsourceOnePage(task, subsource, &paginationInfo)
		if err != nil {
			return err
		}
		paginationInfo.CurrentPageCount++
		if task.GetTaskParams() == nil || paginationInfo.CurrentPageCount > int(task.TaskParams.GetWeiboTaskParams().MaxPages) {
			break
		}
	}

	return nil
}

func (w WeiboApiCollector) CollectAndPublish(task *protocol.PanopticTask) {
	collector.ParallelSubsourceApiCollect(task, w)
}
