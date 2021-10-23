package collector_instances

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/collector"
	sink "github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	JINSE_URI = "https://api.jinse.com/noah/v2/lives?limit=3"

	JINSE_JINXUAN = "精选"
)

type JinseApiCrawler struct {
	Sink sink.CollectedDataSink
}

type JinseResponse struct {
	News     int `json:"news"`
	Count    int `json:"count"`
	Total    int `json:"total"`
	TopID    int `json:"top_id"`
	BottomID int `json:"bottom_id"`
	List     []struct {
		Date  string      `json:"date"`
		Lives []JinsePost `json:"lives"`
	} `json:"list"`
	DefaultShareImg string `json:"default_share_img"`
	PrefixLink      string `json:"prefix_link"`
}

type JinsePost struct {
	ID             int    `json:"id"`
	Content        string `json:"content"`
	ContentPrefix  string `json:"content_prefix"`
	LinkName       string `json:"link_name"`
	Link           string `json:"link"`
	Grade          int    `json:"grade"`
	Sort           string `json:"sort"`
	Category       int    `json:"category"`
	HighlightColor string `json:"highlight_color"`
	Images         []struct {
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Thumbnail string `json:"thumbnail"`
		URL       string `json:"url"`
	} `json:"images"`
	CreatedAt       int           `json:"created_at"`
	Attribute       string        `json:"attribute"`
	UpCounts        int           `json:"up_counts"`
	DownCounts      int           `json:"down_counts"`
	ZanStatus       string        `json:"zan_status"`
	Readings        []interface{} `json:"readings"`
	ExtraType       int           `json:"extra_type"`
	Extra           interface{}   `json:"extra"`
	Prev            interface{}   `json:"prev"`
	Next            interface{}   `json:"next"`
	WordBlocks      []interface{} `json:"word_blocks"`
	IsShowComment   int           `json:"is_show_comment"`
	IsForbidComment int           `json:"is_forbid_comment"`
	CommentCount    int           `json:"comment_count"`
	AnalystUser     interface{}   `json:"analyst_user"`
	ShowSourceName  string        `json:"show_source_name"`
	VoteID          int           `json:"vote_id"`
	Vote            interface{}   `json:"vote"`
}

// For Jinse, it's an important update iff:
// - It has highlight color
// - It has 精选 as attributes
// - It's grade is greater than 4
func (k JinseApiCrawler) GetNewsTypeForPost(post *JinsePost) protocol.PanopticSubSource_SubSourceType {
	if post.HighlightColor != "" ||
		post.Attribute == JINSE_JINXUAN ||
		post.Grade > 4 {
		return protocol.PanopticSubSource_KEYNEWS
	}

	return protocol.PanopticSubSource_FLASHNEWS
}

func (k JinseApiCrawler) GetCrawledSubSourceNameFromPost(post *JinsePost) (string, error) {
	t := k.GetNewsTypeForPost(post)
	return collector.SubsourceTypeToName(t), nil
}

func (k JinseApiCrawler) ParseGenerateTime(post *JinsePost) *timestamppb.Timestamp {
	t := time.Unix(int64(post.CreatedAt), 0)
	return timestamppb.New(t)
}

func (k JinseApiCrawler) ValidatePost(post *JinsePost) error {
	if post.Content == "" {
		return errors.New("content must not be empty")
	}
	return nil
}

func (j JinseApiCrawler) TrimmedContent(post *JinsePost) string {
	return strings.Trim(post.Content, fmt.Sprintf("【%s】", post.ContentPrefix))
}

func (k JinseApiCrawler) ProcessSinglePost(post *JinsePost,
	workingContext *working_context.ApiCollectorWorkingContext) error {
	if err := k.ValidatePost(post); err != nil {
		return err
	}

	subSourceType := k.GetNewsTypeForPost(post)

	if !collector.IsRequestedNewsType(workingContext.Task.TaskParams.SubSources, subSourceType) {
		// Return nil if the post is not of requested type. Note that this is
		// intentionally not considered as failure, and thus will not increase
		// failure count.
		return nil
	}

	collector.InitializeApiCollectorResult(workingContext)

	ts := k.ParseGenerateTime(post)

	name, err := k.GetCrawledSubSourceNameFromPost(post)
	if err != nil {
		return errors.Wrap(err, "cannot find post subsource")
	}

	err = k.GetDedupId(post, workingContext)
	if err != nil {
		return errors.Wrap(err, "cannot get dedup id from post.")
	}

	workingContext.Result.Post.ContentGeneratedAt = ts
	workingContext.Result.Post.Content = k.TrimmedContent(post)
	workingContext.Result.Post.Title = post.ContentPrefix
	workingContext.Result.Post.SubSource.Name = name
	workingContext.Result.Post.SubSource.AvatarUrl = collector.GetSourceLogoUrl(
		workingContext.Task.TaskParams.SourceId)

	return nil
}

func (k JinseApiCrawler) GetDedupId(post *JinsePost, workingContext *working_context.ApiCollectorWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Task.TaskParams.SourceId + strconv.Itoa(post.ID))
	if err != nil {
		return err
	}
	workingContext.Result.Post.DeduplicateId = md5
	return nil
}

func (k JinseApiCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	res := &JinseResponse{}
	err := collector.HttpGetAndParseJsonResponse(JINSE_URI, res)
	if err != nil {
		Logger.Log.WithFields(logrus.Fields{"source": "jinse"}).Errorln("fail to get Jinse response:", err)
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		return
	}

	for _, list := range res.List {
		for _, post := range list.Lives {
			workingContext := &working_context.ApiCollectorWorkingContext{
				SharedContext: working_context.SharedContext{Task: task},
				ApiUrl:        JINSE_URI,
			}

			err := k.ProcessSinglePost(&post, workingContext)
			if err != nil {
				Logger.Log.WithFields(logrus.Fields{"source": "jinse"}).Errorln("fail to process a single Jinse Post:", err,
					"\npost content:\n", collector.PrettyPrint(post))
				workingContext.Task.TaskMetadata.TotalMessageFailed++
				continue
			}

			// Returning nil in ProcessSinglePost doesn't necessarily mean success, it
			// could just be that we're skiping that post (e.g. subsource type doesn't
			// match)
			if workingContext.Result != nil {
				sink.PushResultToSinkAndRecordInTaskMetadata(k.Sink, workingContext)
			}
		}
	}

	collector.SetErrorBasedOnCounts(task, JINSE_URI)
}
