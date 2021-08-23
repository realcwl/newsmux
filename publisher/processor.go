package publisher

import (
	b64 "encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/Luismorlan/newsmux/model"
	. "github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type CrawlerpublisherMessageProcessor struct {
	Reader MessageQueueReader
	DB     *gorm.DB
}

// Create new processor with reader dependency injection
func NewPublisherMessageProcessor(reader MessageQueueReader, db *gorm.DB) *CrawlerpublisherMessageProcessor {
	return &CrawlerpublisherMessageProcessor{
		Reader: reader,
		DB:     db,
	}
}

// Use Reader to read N messages and process them in parallel
// Time out or queue name etc are defined in reader
// Reader focus on how to get message from queue
// Processor focus on how to process the message
// This function doesn't return anything, only log errors
func (processor *CrawlerpublisherMessageProcessor) ReadAndProcessMessages(maxNumberOfMessages int64) {
	// Pull queued messages from queue
	msgs, err := processor.Reader.ReceiveMessages(maxNumberOfMessages)

	if err != nil {
		Log.Error("fail read crawler messages from queue : ", err)
		return
	}

	// Process
	// TODO: process in parallel
	for _, msg := range msgs {
		if err := processor.ProcessOneCralwerMessage(msg); err != nil {
			Log.Error("fail process one crawler message : ", err)
			continue
		}
	}
}

func (processor *CrawlerpublisherMessageProcessor) findDuplicatedPost(decodedMsg *CrawlerMessage) (bool, *model.Post) {
	var post model.Post
	queryResult := processor.DB.Debug().Where(
		"source_id = ? AND (COALESCE(sub_source_id, '') = COALESCE(?, '')) AND title = ? AND content = ? ",
		decodedMsg.Post.SourceId,
		decodedMsg.Post.SubSourceId,
		decodedMsg.Post.Title,
		decodedMsg.Post.Content,
	).First(&post)

	return queryResult.RowsAffected != 0, &post
}

func (processor *CrawlerpublisherMessageProcessor) prepareFeedCandidates(
	source *model.Source,
	subSource *model.SubSource,
) map[string]*model.Feed {

	feedCandidates := make(map[string]*model.Feed)

	if source != nil {
		for _, feed := range source.Feeds {
			feedCandidates[feed.Id] = feed
		}
	}

	if subSource != nil {
		for _, feed := range subSource.Feeds {
			feedCandidates[feed.Id] = feed
		}
	}
	return feedCandidates
}

func (processor *CrawlerpublisherMessageProcessor) prepareSource(id string) (*model.Source, error) {
	var res model.Source
	if len(id) > 0 {
		result := processor.DB.Preload("Feeds").Where("id = ?", id).First(&res)
		if result.RowsAffected != 1 {
			return nil, errors.New(fmt.Sprintf("source not found: %s", id))
		}
		return &res, nil
	} else {
		return nil, errors.New("source id can not be empty")
	}
}

func (processor *CrawlerpublisherMessageProcessor) prepareSubSource(id string) (*model.SubSource, error) {
	var res model.SubSource
	if len(id) > 0 {
		result := processor.DB.Preload("Feeds").Where("id = ?", id).First(&res)
		if result.RowsAffected != 1 {
			return nil, errors.New(fmt.Sprintf("sub source not found: %s", id))
		}
		return &res, nil
	}
	return nil, nil
}

// Process one cralwer-publisher message in following major steps:
// Step1. decode into protobuf generated struct
// Step2. deduplication
// Step3. do publishing with new post
// Step4. if publishing succeeds, delete message in queue
func (processor *CrawlerpublisherMessageProcessor) ProcessOneCralwerMessage(msg *MessageQueueMessage) error {
	Log.Info("process queued message")

	decodedMsg, err := processor.decodeCrawlerMessage(msg)
	if err != nil {
		return err
	}

	// Once get a message, check if there is exact same Post (same sources, same content), if not store into DB as Post
	if duplicated, existingPost := processor.findDuplicatedPost(decodedMsg); duplicated == true {
		return errors.New(fmt.Sprintf("message has already been processed, existing post_id: %s", existingPost.Id))
	}

	// Prepare Post relations to Sources and Subsources
	source, err := processor.prepareSource(decodedMsg.Post.SourceId)
	if err != nil {
		return err
	}

	// subsource can be nil
	subSource, err := processor.prepareSubSource(decodedMsg.Post.SubSourceId)
	if err != nil {
		return err
	}

	// Load feeds into memory based on source and subsource of the post
	feedCandidates := processor.prepareFeedCandidates(source, subSource)

	// Create new post based on message
	post := model.Post{
		Id:             uuid.New().String(),
		Title:          decodedMsg.Post.Title,
		Content:        decodedMsg.Post.Content,
		CreatedAt:      time.Now(),
		Source:         *source,
		SourceID:       decodedMsg.Post.SourceId,
		SubSource:      subSource,
		SavedByUser:    []*model.User{},
		PublishedFeeds: []*model.Feed{},
	}

	// Check each feed's source/subsource and data expression
	err = processor.DB.Transaction(func(tx *gorm.DB) error {
		processor.DB.Create(&post)
		for _, feed := range feedCandidates {
			// Once a message is matched to a feed, write the PostFeedPublish relation to DB
			matched, err := DataExpressionJsonMatch(feed.FilterDataExpression.String(), post.Content)
			if err != nil {
				return err
			}
			if matched {
				processor.DB.Model(&post).Association("PublishedFeeds").Append(feed)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Delete message from queue
	return processor.Reader.DeleteMessage(msg)
}

// Parse message into meaningful structure CrawlerMessage
// This function assumes message passed in can be parsed, otherwise it will throw error
func (processor *CrawlerpublisherMessageProcessor) decodeCrawlerMessage(msg *MessageQueueMessage) (*CrawlerMessage, error) {
	str, err := msg.Read()
	if err != nil {
		return nil, err
	}

	sDec, err := b64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}

	decodedMsg := &CrawlerMessage{}
	if err := proto.Unmarshal(sDec, decodedMsg); err != nil {
		return nil, err
	}

	return decodedMsg, nil
}
