package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/pkg/errors"
)

func getSourceLogoUrl(sourceId string) string {
	switch sourceId {
	case collector.Jin10SourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png"
	case collector.WeiboSourceId:
		return ""
	case collector.WallstreetNewsSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/wallstrt.png"
	case collector.KuailansiSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/kuailansi.png"
	case collector.JinseSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jinse.png"
	case collector.WisburgSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/wisburg.png"
	case collector.Kr36SourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/36ke.png"
	case collector.CaixinSourceId:
		return "https://newsfeed-logo.s3.us-west-1.amazonaws.com/caixin.png"
	default:
		return ""
	}
}

func getSourceIdFromDataCollectorId(collectorId protocol.PanopticTask_DataCollectorId) string {
	switch collectorId {
	case protocol.PanopticTask_COLLECTOR_JINSHI:
		return collector.Jin10SourceId
	case protocol.PanopticTask_COLLECTOR_KUAILANSI:
		return collector.KuailansiSourceId
	case protocol.PanopticTask_COLLECTOR_WEIBO:
		return collector.WeiboSourceId
	case protocol.PanopticTask_COLLECTOR_WALLSTREET_NEWS:
		return collector.WallstreetNewsSourceId
	case protocol.PanopticTask_COLLECTOR_JINSE:
		return collector.JinseSourceId
	case protocol.PanopticTask_COLLECTOR_CAUS_ARTICLE:
		return collector.CaUsSourceId
	case protocol.PanopticTask_COLLECTOR_WEIXIN_ARTICLE:
		return collector.WeixinSourceId
	case protocol.PanopticTask_COLLECTOR_WISBURG:
		return collector.WisburgSourceId
	case protocol.PanopticTask_COLLECTOR_KR36:
		return collector.Kr36SourceId
	case protocol.PanopticTask_COLLECTOR_WALLSTREET_ARTICLE:
		return collector.WallstreetArticleSourceId
	case protocol.PanopticTask_COLLECTOR_CAUS_NEWS:
		return collector.CaUsSourceId
	case protocol.PanopticTask_COLLECTOR_CAIXIN:
		return collector.CaixinSourceId
	case protocol.PanopticTask_COLLECTOR_CLS_ARTICLE:
		return collector.ClsSourceId
	default:
		return ""
	}
}

// Validate a message before it get published to sink.
// Validator applied only to the shared context, where it contains the task to
// be returned back to Panoptic, as well as the crawled messages.
// A non valid shared context must not be pushed to sink.
func ValidateSharedContext(sharedContext *working_context.SharedContext) error {
	validators := []func(*working_context.SharedContext) error{
		crawledMessageValidation,
		panopticTaskValidation,
		crossTaskMessageValidation,
	}

	for _, v := range validators {
		if err := v(sharedContext); err != nil {
			return errors.Wrap(err, sharedContext.String())
		}
	}

	return nil
}

// Validate CrawledMessage is set correctly before pushing to sink. This type of
// validation only looks at CrawledMessage without any context about the
// original task itself. It's a stateless validation.
func crawledMessageValidation(sharedContext *working_context.SharedContext) error {
	messageValidators := []func(*protocol.CrawlerMessage) error{
		validateMessageSubSourceIsSetCorrectly,
		validateMessagePostIsSetCorrectly,
		validateMessageMetadataIsSetCorrectly,
	}
	for _, v := range messageValidators {
		if err := v(sharedContext.Result); err != nil {
			return utils.ImmediatePrintError(err)
		}
	}
	return nil
}

// Validate PanopticTask is set correctly before pushing to sink. This type of
// validation only looks at PanopticTask without any context about the
// original task itself. It's a stateless validation.
func panopticTaskValidation(sharedContext *working_context.SharedContext) error {
	taskValidators := []func(*protocol.PanopticTask) error{
		validateTaskMetadataIsSetCorrectly,
	}
	for _, v := range taskValidators {
		if err := v(sharedContext.Task); err != nil {
			return err
		}
	}
	return nil
}

// Validate CrawledMessage indeed matches the task specification.
func crossTaskMessageValidation(sharedContext *working_context.SharedContext) error {
	task := sharedContext.Task
	msg := sharedContext.Result

	if msg.Post.SubSource.SourceId != task.TaskParams.SourceId {
		return errors.New("crawled message mismatch task's source id")
	}

	defaultUrl := getSourceLogoUrl(task.TaskParams.SourceId)
	if defaultUrl != "" && msg.Post.SubSource.AvatarUrl != defaultUrl {
		return errors.New("crawled message's avatar doesn't match the source's default avatar url: " + defaultUrl)
	}

	if msg.Post.SubSource.SourceId != getSourceIdFromDataCollectorId(task.DataCollectorId) {
		return fmt.Errorf("crawled message's source id doesn't match the data collector id, msg: %s != task: %s",
			msg.Post.SubSource.SourceId,
			getSourceIdFromDataCollectorId(task.DataCollectorId),
		)
	}

	return nil
}

// SubSource on Post is set correctly. A subsource is valid iff:
// - Has AvatarUrl
// - Has SourceId
// - Has SubSourceId
func validateMessageSubSourceIsSetCorrectly(msg *protocol.CrawlerMessage) error {
	if msg.Post.SubSource.AvatarUrl == "" {
		return errors.New("crawled post subsource must have avatar url")
	}

	if msg.Post.SubSource.Name == "" {
		return errors.New("crawled post subsource must have name")
	}

	if msg.Post.SubSource.SourceId == "" {
		return errors.New("crawled post subsource must be associated with a source id")
	}

	return nil
}

func isPostWithEmptyContent(post *protocol.CrawlerMessage_CrawledPost) bool {
	return post.Content == "" && len(post.ImageUrls) == 0 && len(post.FilesUrls) == 0
}

// A Post is valid iff:
// - It has content
// - It is associated with a generated time to render correct timestamp
// - It has a deduplicateId
func validateMessagePostIsSetCorrectly(msg *protocol.CrawlerMessage) error {
	if msg.Post == nil {
		return errors.New("crawled post must have be set")
	}

	if (msg.Post.SharedFromCrawledPost == nil && isPostWithEmptyContent(msg.Post)) ||
		(isPostWithEmptyContent(msg.Post) && (msg.Post.SharedFromCrawledPost == nil || isPostWithEmptyContent(msg.Post.SharedFromCrawledPost))) {
		if !strings.Contains(msg.Post.OriginUrl, "weixin") {
			// Weixin is an exception given the fact that weixin posts can be an image and link to original article
			return errors.New("crawled post must have Content / Image / File")
		}
	}

	if msg.Post.ContentGeneratedAt == nil {
		return errors.New("crawled post must be associated with a generated time")
	}

	if msg.Post.DeduplicateId == "" {
		return errors.New("crawled post must have a deduplicateId")
	}

	if msg.Post.ContentGeneratedAt.AsTime().After(time.Now()) {
		return errors.New("crawled post must not be created in the future")
	}

	return nil
}

// A message's metadata is valid iff:
// - It is associated with a crawled time
func validateMessageMetadataIsSetCorrectly(msg *protocol.CrawlerMessage) error {
	if msg.CrawledAt == nil {
		return errors.New("crawled message must be associated with a crawled time")
	}

	return nil
}

// A PanopticTask's metadata is set correctly iff:
// - It must have the config name that generated it
// - It must be associated with an end state, and the state is correct
func validateTaskMetadataIsSetCorrectly(task *protocol.PanopticTask) error {
	if task.TaskMetadata.ConfigName == "" {
		return errors.New("PanopticTask must have config name populated")
	}

	if task.TaskMetadata.ResultState == protocol.TaskMetadata_STATE_UNSPECIFIED {
		return errors.New("PanopticTask must be associated with a result state")
	}

	if task.TaskMetadata.TotalMessageFailed > 0 &&
		task.TaskMetadata.ResultState == protocol.TaskMetadata_STATE_SUCCESS {
		return errors.New("PanopticTask must be at failure state if it has non zero failure message")
	}

	return nil
}
