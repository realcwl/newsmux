package publisher

import (
	b64 "encoding/base64"
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/server/resolver"
	"github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type TestMessageQueueReader struct {
	msgs []*MessageQueueMessage
}

func (reader *TestMessageQueueReader) DeleteMessage(msg *MessageQueueMessage) error {
	return nil
}

// Always return all messages
func (reader *TestMessageQueueReader) ReceiveMessages(maxNumberOfMessages int64) (msgs []*MessageQueueMessage, err error) {
	return reader.msgs, nil
}

// Pass in all the crawler messages that will be used for testing
// Reader will return queue message object
func NewTestMessageQueueReader(crawlerMsgs []*protocol.CrawlerMessage) *TestMessageQueueReader {
	var res TestMessageQueueReader
	var queueMsgs []*MessageQueueMessage

	for _, m := range crawlerMsgs {
		encodedBytes, _ := proto.Marshal(m)
		str := b64.StdEncoding.EncodeToString(encodedBytes)
		var msg MessageQueueMessage
		msg.Message = &str
		queueMsgs = append(queueMsgs, &msg)
	}
	res.msgs = queueMsgs
	return &res
}

func TestDecodeCrawlerMessage(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	origin := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			SourceId:           "1",
			SubSourceId:        "2",
			Title:              "hello",
			Content:            "hello world!",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamp.Timestamp{},
		},
		CrawledAt:      &timestamp.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&origin,
	})

	// Inject test dependent reader
	processor := NewPublisherMessageProcessor(reader, db)

	msgs, _ := reader.ReceiveMessages(1)
	assert.Equal(t, len(msgs), 1)

	// This is the function we tested here
	//given MessageQueueMessage, decode it into struct
	decodedObj, _ := processor.decodeCrawlerMessage(msgs[0])

	assert.True(t, cmp.Equal(*decodedObj, origin, cmpopts.IgnoreUnexported(
		protocol.CrawlerMessage{},
		protocol.CrawlerMessage_CrawledPost{},
		timestamp.Timestamp{},
	)))
}

func PrepareTestDBClient(db *gorm.DB) *client.Client {
	client := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolver.Resolver{
		DB:             db,
		SeedStateChans: nil,
	}})))
	return client
}

func TestProcessCrawlerMessage(t *testing.T) {
	db, _ := utils.CreateTempDB(t)
	client := PrepareTestDBClient(db)
	dataExpression := `{\"a\":1}`

	uid := utils.TestCreateUserAndValidate(t, "test_user_name", db, client)
	sourceId1 := utils.TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	sourceId2 := utils.TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	sourceId3 := utils.TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId1 := utils.TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api", "test_externalid", sourceId1, db, client)
	subSourceId2 := utils.TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api", "test_externalid", sourceId2, db, client)

	feedId := utils.TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", dataExpression, []string{}, []string{subSourceId1}, db, client)
	feedId2 := utils.TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api_2", dataExpression, []string{sourceId3}, []string{subSourceId1, subSourceId2}, db, client)
	utils.TestUserSubscribeFeedAndValidate(t, uid, feedId, db, client)
	utils.TestUserSubscribeFeedAndValidate(t, uid, feedId2, db, client)

	t.Run("Test Publish Post to Feed based on subsource", func(t *testing.T) {
		// msg1 is from subsource 1 which in 2 feeds
		msg1 := protocol.CrawlerMessage{
			Post: &protocol.CrawlerMessage_CrawledPost{
				SourceId:           sourceId1,
				SubSourceId:        subSourceId1,
				Title:              "hello",
				Content:            "hello world!",
				ImageUrls:          []string{"1", "4"},
				FilesUrls:          []string{"2", "3"},
				OriginUrl:          "aaa",
				ContentGeneratedAt: &timestamp.Timestamp{},
			},
			CrawledAt:      &timestamp.Timestamp{},
			CrawlerIp:      "123",
			CrawlerVersion: "vde",
			IsTest:         false,
		}
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msg1,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db)
		processor.ProcessOneCralwerMessage(msgs[0])

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msg1.Post.Title)
		fmt.Println(post.Content)
		require.Equal(t, 2, len(post.PublishedFeeds))
		require.Equal(t, feedId, post.PublishedFeeds[0].Id)
		require.Equal(t, feedId2, post.PublishedFeeds[1].Id)
	})

	t.Run("Test Publish Post to Feed based on source", func(t *testing.T) {
		// msg2 is from source 1 which in 1 feed
		msg2 := protocol.CrawlerMessage{
			Post: &protocol.CrawlerMessage_CrawledPost{
				SourceId:           sourceId3,
				SubSourceId:        "",
				Title:              "hello2",
				Content:            "hello world!",
				ImageUrls:          []string{"1", "4"},
				FilesUrls:          []string{"2", "3"},
				OriginUrl:          "aaa",
				ContentGeneratedAt: &timestamp.Timestamp{},
			},
			CrawledAt:      &timestamp.Timestamp{},
			CrawlerIp:      "123",
			CrawlerVersion: "vde",
			IsTest:         false,
		}
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msg2,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db)
		processor.ProcessOneCralwerMessage(msgs[0])

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msg2.Post.Title)
		fmt.Println(post.Content)
		require.Equal(t, 1, len(post.PublishedFeeds))
		require.Equal(t, post.PublishedFeeds[0].Id, feedId2)
	})
}
