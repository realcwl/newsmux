package publisher

import (
	b64 "encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/server/resolver"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jinzhu/copier"
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
	db, _ := CreateTempDB(t)

	origin := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
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
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)

	jsonStr := DataExpressionJsonForTest
	var root model.DataExpressionRoot
	json.Unmarshal([]byte(jsonStr), &root)
	bytes, _ := json.Marshal(root)
	dataExpression := strings.ReplaceAll(string(bytes), `"`, `\"`)

	uid := TestCreateUserAndValidate(t, "test_user_name", "test_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	sourceId2 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId1 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api", "test_externalid", sourceId1, db, client)
	subSourceId2 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api", "test_externalid", sourceId2, db, client)

	feedId, _ := TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", dataExpression, []string{subSourceId1}, db, client)
	feedId2, _ := TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api_2", dataExpression, []string{subSourceId1, subSourceId2}, db, client)
	TestUserSubscribeFeedAndValidate(t, uid, feedId, db, client)
	TestUserSubscribeFeedAndValidate(t, uid, feedId2, db, client)

	msgToTwoFeeds := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			SubSourceId:        subSourceId1,
			Title:              "msgToTwoFeeds",
			Content:            "老王做空以太坊",
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

	msgToOneFeed := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			SubSourceId:        subSourceId2,
			Title:              "msgToOneFeed",
			Content:            "老王做空以太坊",
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

	var msgDataExpressionUnMatched protocol.CrawlerMessage
	copier.Copy(&msgDataExpressionUnMatched, &msgToOneFeed)
	msgDataExpressionUnMatched.Post.Title = "msgDataExpressionUnMatched"
	msgDataExpressionUnMatched.Post.Content = "马斯克做空以太坊"

	t.Run("Test Publish Post to Feed based on subsource", func(t *testing.T) {
		// msgToTwoFeeds is from subsource 1 which in 2 feeds
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToTwoFeeds,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db)
		err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgToTwoFeeds.Post.Title)
		require.Equal(t, 2, len(post.PublishedFeeds))
		require.Equal(t, feedId, post.PublishedFeeds[0].Id)
		require.Equal(t, feedId2, post.PublishedFeeds[1].Id)
	})

	t.Run("Test Publish Post to Feed based on source", func(t *testing.T) {
		// msgToOneFeed is from source 1 which in 1 feed
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToOneFeed,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db)
		err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgToOneFeed.Post.Title)
		require.Equal(t, 1, len(post.PublishedFeeds))
		require.Equal(t, post.PublishedFeeds[0].Id, feedId2)
	})

	t.Run("Test Post deduplication", func(t *testing.T) {
		// msgToOneFeed is from source 1 which in 1 feed
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToOneFeed,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing Again, should have error indicating publish failure
		processor := NewPublisherMessageProcessor(reader, db)
		err := processor.ProcessOneCralwerMessage(msgs[0])
		require.NotNil(t, err)
	})

	t.Run("Test Publish Post to Feed based on source Data Expression not matched", func(t *testing.T) {
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgDataExpressionUnMatched,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db)
		err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgDataExpressionUnMatched.Post.Title)
		require.Equal(t, 0, len(post.PublishedFeeds))
	})
}
