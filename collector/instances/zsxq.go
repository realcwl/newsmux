package collector_instances

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	. "github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Should Construct With Collector Builder
type ZsxqApiCollector struct {
	Sink      CollectedDataSink
	FileStore CollectedFileStore
}

func (collector ZsxqApiCollector) UpdateFileUrls(workingContext *ApiCollectorWorkingContext) error {
	return errors.New("UpdateFileUrls not implemented, should not be called")
}

func (collector ZsxqApiCollector) ConstructUrl(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource, paginationInfo *PaginationInfo) string {
	return fmt.Sprintf("https://api.zsxq.com/v2/groups/{%s}/topics?scope=all&count=20",
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

func (collector ZsxqApiCollector) UppdateFiles(zsxqBlog *MBlog, post *protocol.CrawlerMessage_CrawledPost) error {
	post.ImageUrls = []string{}
	for _, pic := range zsxqBlog.Pics {
		key, err := collector.FileStore.FetchAndStore(pic.URL)
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		s3Url := collector.FileStore.GetUrlFromKey(key)
		post.ImageUrls = append(post.ImageUrls, s3Url)
	}
	return nil
}
func (collector ZsxqApiCollector) UppdateImages(zsxqBlog *MBlog, post *protocol.CrawlerMessage_CrawledPost) error {
	post.ImageUrls = []string{}
	for _, pic := range zsxqBlog.Pics {
		key, err := collector.FileStore.FetchAndStore(pic.URL)
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		s3Url := collector.FileStore.GetUrlFromKey(key)
		post.ImageUrls = append(post.ImageUrls, s3Url)
	}
	return nil
}

func (collector ZsxqApiCollector) UpdateResult(zsxqBlog *MBlog, post *protocol.CrawlerMessage_CrawledPost) error {
	generatedTime, err := time.Parse(WeiboDateFormat, zsxqBlog.CreatedAt)
	post.ContentGeneratedAt = timestamppb.New(generatedTime)
	if zsxqBlog.User == nil {
		post.SubSource.Name = "default"
	} else {
		post.SubSource.Name = zsxqBlog.User.ScreenName
		post.SubSource.AvatarUrl = "https://weibo.com/" + fmt.Sprint(zsxqBlog.User.ID) + "/" + zsxqBlog.Bid
		post.SubSource.ExternalId = fmt.Sprint(zsxqBlog.User.ID)
	}
	collector.UppdateImages(zsxqBlog, post)
	post.OriginUrl = "https://m.weibo.cn/detail/" + zsxqBlog.ID

	post.Content = zsxqBlog.Text
	post.Content = CleanWeiboContent(post.Content)

	err = collector.UpdateDedupId(post)
	if err != nil {
		return utils.ImmediatePrintError(err)
	}

	if zsxqBlog.RetweetedStatus != nil {
		fmt.Println("starting processing shared post weibo id:", zsxqBlog.ID)
		sharedPost := &protocol.CrawlerMessage_CrawledPost{
			SubSource: &protocol.CrawledSubSource{
				SourceId: post.SubSource.SourceId,
			},
		}
		err = collector.UpdateResult(zsxqBlog.RetweetedStatus, sharedPost)
		if err != nil {
			return utils.ImmediatePrintError(err)
		}
		post.SharedFromCrawledPost = sharedPost
	}

	return nil
}

func (collector ZsxqApiCollector) CollectOneSubsourceOnePage(
	task *protocol.PanopticTask,
	subsource *protocol.PanopticSubSource,
	paginationInfo *PaginationInfo,
) error {
	var client HttpClient
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
	if res.Ok != 1 {
		return errors.New(fmt.Sprintf("response not success: %v", res))
	}

	for _, topic := range res.RespData.Topics {
		// working context for each message
		workingContext := &ApiCollectorWorkingContext{
			Task:           task,
			PaginationInfo: paginationInfo,
			ApiUrl:         url,
			Result:         &protocol.CrawlerMessage{},
			Subsource:      subsource,
		}
		InitializeApiCollectorResult(workingContext)
		err := collector.UpdateResult(&topic, workingContext.Result.Post)
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

	// Set next page identifier
	paginationInfo.NextPageId = fmt.Sprint(res.Data.CardlistInfo.Page)
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
