package collector_instances

import (
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
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	JINSE_URI = "https://api.jinse.com/noah/v2/lives?limit=3"
	// CHINA_TIMEZONE        = "Asia/Shanghai"
	// KUAILANSI_TIME_FORMAT = "2006-01-02 15:04:05"

	// IP_BAN_MESSAGE = "IP访问受限制"
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

// For kuailansi, if Level == 0, it's a important update.
func (k JinseApiCrawler) GetNewsTypeForPost(post *KuailansiPost) (protocol.PanopticSubSource_SubSourceType, error) {
	level, err := strconv.Atoi(post.Level)
	if err != nil {
		return protocol.PanopticSubSource_UNSPECIFIED, errors.Wrap(err, "cannot parse post.Level")
	}

	if level >= 1 {
		return protocol.PanopticSubSource_FLASHNEWS, nil
	}

	return protocol.PanopticSubSource_KEYNEWS, nil
}

func (k JinseApiCrawler) GetCrawledSubSourceNameFromPost(post *KuailansiPost) (string, error) {
	t, err := k.GetNewsTypeForPost(post)
	if err != nil {
		return "", errors.Wrap(err, "fail to get subsource type from post"+collector.PrettyPrint(post))
	}
	return collector.SubsourceTypeToName(t), nil
}

func (k JinseApiCrawler) ParseGenerateTime(post *KuailansiPost) (*timestamppb.Timestamp, error) {
	location, err := time.LoadLocation(CHINA_TIMEZONE)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse time zome: "+CHINA_TIMEZONE)
	}
	t, err := time.ParseInLocation(KUAILANSI_TIME_FORMAT, post.Time, location)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse Kuailansi post time: "+post.Time)
	}
	return timestamppb.New(t), nil
}

func (k JinseApiCrawler) ValidatePost(post *KuailansiPost) error {
	if strings.Contains(post.Content, IP_BAN_MESSAGE) {
		return errors.New("IP is banned")
	}
	return nil
}

func (k JinseApiCrawler) ProcessSinglePost(post *KuailansiPost,
	workingContext *working_context.ApiCollectorWorkingContext) error {
	if err := k.ValidatePost(post); err != nil {
		return err
	}

	subSourceType, err := k.GetNewsTypeForPost(post)
	if err != nil {
		return err
	}

	if !collector.IsRequestedNewsType(workingContext.Task.TaskParams.SubSources, subSourceType) {
		// Return nil if the post is not of requested type. Note that this is
		// intentionally not considered as failure.
		return nil
	}

	collector.InitializeApiCollectorResult(workingContext)

	ts, err := k.ParseGenerateTime(post)
	if err != nil {
		return err
	}

	name, err := k.GetCrawledSubSourceNameFromPost(post)
	if err != nil {
		return errors.Wrap(err, "cannot find post subsource")
	}

	err = k.GetDedupId(workingContext)
	if err != nil {
		return errors.Wrap(err, "cannot get dedup id from post.")
	}

	workingContext.Result.Post.ContentGeneratedAt = ts
	workingContext.Result.Post.Content = post.Content
	workingContext.Result.Post.SubSource.Name = name
	workingContext.Result.Post.SubSource.AvatarUrl = collector.GetSourceLogoUrl(
		workingContext.Task.TaskParams.SourceId)

	return nil
}

func (k JinseApiCrawler) GetDedupId(workingContext *working_context.ApiCollectorWorkingContext) error {
	md5, err := utils.TextToMd5Hash(workingContext.Result.Post.SubSource.Id + workingContext.Result.Post.Content)
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
		Logger.Log.Errorln("fail to get Jinse response:", err)
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
				Logger.Log.Errorln("fail to process a single Kuailansi Post:", err,
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

	collector.SetErrorBasedOnCounts(task, KUAILANSI_URI)
}
