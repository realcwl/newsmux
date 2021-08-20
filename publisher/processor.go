package publisher

import (
	b64 "encoding/base64"
	"fmt"
	"time"

	"github.com/Luismorlan/newsmux/model"
	. "github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// Process one cralwer-publisher message
// Step1. decode into protobuf generated struct
// Step2. do publishing with new post
// Step3. if publishing succeeds, delete message in queue
func (processor *CrawlerpublisherMessageProcessor) ProcessOneCralwerMessage(msg *MessageQueueMessage) error {

	decodedMsg, err := processor.decodeCrawlerMessage(msg)
	if err != nil {
		return err
	}

	// TODO: Do actual publishing for decodedMsg
	Log.Info(decodedMsg)

	feedCandidates := make(map[string]*model.Feed)

	// 1. Load all feeds into memory
	var feeds []*model.Feed
	processor.DB.Preload(clause.Associations).Find(&feeds)

	// 2. Once get a message, check if there is exact same Post (same source, same content), if not store into DB as Post
	var post model.Post

	queryResult := processor.DB.Where(
		"source_id = ? AND (IF(sub_source_id IS NULL, TRUE, sub_source_id = ?)) AND title = ? AND content = ? ",
		decodedMsg.Post.SourceId,
		decodedMsg.Post.SubSourceId,
		decodedMsg.Post.Title,
		decodedMsg.Post.Content,
	).First(&post)

	if queryResult.RowsAffected != 0 {
		fmt.Println("message has already been processed: ", post.Content)
		Log.Error("message has already been processed")
	}

	uuid := uuid.New().String()
	var (
		source    model.Source
		subSource *model.SubSource
	)

	if len(decodedMsg.Post.SourceId) > 0 {
		result := processor.DB.Preload("Feeds").Where("id = ?", decodedMsg.Post.SourceId).First(&source)
		if result.RowsAffected != 1 {

		}
		for _, feed := range source.Feeds {
			feedCandidates[feed.Id] = feed
		}
	} else {

	}

	if len(decodedMsg.Post.SubSourceId) > 0 {
		var res model.SubSource
		result := processor.DB.Preload("Feeds").Where("id = ?", decodedMsg.Post.SubSourceId).First(&res)
		if result.RowsAffected != 1 {

		}
		subSource = &res
		for _, feed := range subSource.Feeds {
			feedCandidates[feed.Id] = feed
		}
	}

	post = model.Post{
		Id:             uuid,
		Title:          decodedMsg.Post.Title,
		Content:        decodedMsg.Post.Content,
		CreatedAt:      time.Now(),
		Source:         source,
		SourceID:       decodedMsg.Post.SourceId,
		SubSource:      subSource,
		SavedByUser:    []*model.User{},
		PublishedFeeds: []*model.Feed{},
	}

	// 3. Check each feed's source/subsource and data expression
	processor.DB.Transaction(func(tx *gorm.DB) error {
		processor.DB.Create(&post)
		for id, feed := range feedCandidates {
			fmt.Println("Checking feed candidates Key: ", id, " Value: ", feed.Name)
			// Once a message is matched to a feed, write the PostFeedPublish relation to DB
			// TODO: ADD matching basedon data expression
			processor.DB.Model(&post).Association("PublishedFeeds").Append(feed)
		}
		return nil
	})

	// 5. Delete message from queue
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
