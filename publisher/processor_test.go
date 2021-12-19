package publisher

import (
	b64 "encoding/base64"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Luismorlan/newsmux/deduplicator"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/server/resolver"
	. "github.com/Luismorlan/newsmux/utils"
	"github.com/Luismorlan/newsmux/utils/dotenv"
)

func TestMain(m *testing.M) {
	dotenv.LoadDotEnvsInTests()
	os.Exit(m.Run())
}

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
			SubSource: &protocol.CrawledSubSource{
				Id: "2",
			},
			Title:              "hello",
			Content:            "hello world!",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&origin,
	})

	// Inject test dependent reader
	processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})

	msgs, _ := reader.ReceiveMessages(1)
	assert.Equal(t, len(msgs), 1)

	// This is the function we tested here
	//given MessageQueueMessage, decode it into struct
	decodedObj, _ := processor.decodeCrawlerMessage(msgs[0])

	assert.True(t, cmp.Equal(*decodedObj, origin, cmpopts.IgnoreUnexported(
		protocol.CrawlerMessage{},
		protocol.CrawlerMessage_CrawledPost{},
		protocol.CrawledSubSource{},
		timestamppb.Timestamp{},
	)))
}

func PrepareTestDBClient(db *gorm.DB) *client.Client {
	client := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolver.Resolver{
		DB:          db,
		SignalChans: nil,
	}})))
	return client
}

func TestProcessCrawlerMessage(t *testing.T) {
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)

	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	sourceId2 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId1 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api", "test_externalid", sourceId1, false, db, client)
	subSourceId2 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api_2", "test_externalid", sourceId2, false, db, client)

	feedId, _ := TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	feedId2, _ := TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api_2", DataExpressionJsonForTest, []string{subSourceId1, subSourceId2}, model.VisibilityPrivate, db, client)
	TestUserSubscribeFeedAndValidate(t, uid, feedId, db, client)
	TestUserSubscribeFeedAndValidate(t, uid, feedId2, db, client)

	testTimeStamp := timestamppb.Now()

	msgToTwoFeeds := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "1",
			SubSource: &protocol.CrawledSubSource{
				Name:     "test_subsource_for_feeds_api",
				SourceId: sourceId1,
			},
			Title:              "msgToTwoFeeds",
			Content:            "老王做空以太坊",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"Tesla", "中概股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: testTimeStamp,
		},
		CrawledAt:      testTimeStamp,
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	msgToOneFeed := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "2",
			SubSource: &protocol.CrawledSubSource{
				Name:     "test_subsource_for_feeds_api_2",
				SourceId: sourceId2,
			},
			Title:              "msgToOneFeed",
			Content:            "老王做空以太坊_2",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: testTimeStamp,
		},
		CrawledAt:      testTimeStamp,
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	var msgDataExpressionUnMatched protocol.CrawlerMessage
	copier.Copy(&msgDataExpressionUnMatched, &msgToOneFeed)
	msgDataExpressionUnMatched.Post.Title = "msgDataExpressionUnMatched"
	msgDataExpressionUnMatched.Post.Content = "马斯克做空以太坊"
	msgDataExpressionUnMatched.Post.DeduplicateId = "3"

	t.Run("Test Publish Post to Feed based on subsource", func(t *testing.T) {
		// msgToTwoFeeds is from subsource 1 which in 2 feeds
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToTwoFeeds,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgToTwoFeeds.Post.Title)
		require.Equal(t, 2, len(post.PublishedFeeds))
		require.Equal(t, feedId, post.PublishedFeeds[0].Id)
		require.Equal(t, feedId2, post.PublishedFeeds[1].Id)
		require.Equal(t, testTimeStamp.Seconds, post.ContentGeneratedAt.Unix())
		require.Equal(t, testTimeStamp.Seconds, post.CrawledAt.Unix())
		require.Equal(t, "Tesla,中概股", post.Tag)
	})

	t.Run("Test Publish Post to Feed based on source", func(t *testing.T) {
		// msgToOneFeed is from source 1 which in 1 feed
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToOneFeed,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgToOneFeed.Post.Title)
		require.Equal(t, 1, len(post.PublishedFeeds))
		require.Equal(t, post.PublishedFeeds[0].Id, feedId2)
		require.Equal(t, 2, len(post.ImageUrls))
		require.Equal(t, "1", post.ImageUrls[0])
		require.Equal(t, 2, len(post.FileUrls))
		require.Equal(t, "aaa", post.OriginUrl)
		require.Equal(t, "电动车,港股", post.Tag)
	})

	t.Run("Test Post deduplication", func(t *testing.T) {
		// send message again
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToOneFeed,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing Again, there should be no new post
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.NoError(t, err)

		var count int64
		processor.DB.Model(&model.Post{}).Where("title = ?", msgToOneFeed.Post.Title).Count(&count)
		require.Equal(t, int64(1), count)
	})

	t.Run("Test Publish Post to Feed based on source Data Expression not matched", func(t *testing.T) {
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgDataExpressionUnMatched,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload("PublishedFeeds").First(&post, "title = ?", msgDataExpressionUnMatched.Post.Title)
		require.Equal(t, 0, len(post.PublishedFeeds))
	})
}

func TestProcessCrawlerRetweetMessage(t *testing.T) {
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)
	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId1 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_1", "test_externalid", sourceId1, false, db, client)
	feedId, _ := TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)

	msgToOneFeed := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "1",
			SubSource: &protocol.CrawledSubSource{
				// New subsource to be created and mark as isFromSharedPost
				Name:       "test_subsource_1",
				SourceId:   sourceId1,
				ExternalId: "a",
				AvatarUrl:  "a",
				OriginUrl:  "a",
			},
			Title:              "老王干得好", // This doesn't match data exp
			Content:            "老王干得好",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
			SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
				DeduplicateId: "2",
				SubSource: &protocol.CrawledSubSource{
					// New subsource to be created and mark as isFromSharedPost
					Name:       "test_subsource_2",
					SourceId:   sourceId1,
					ExternalId: "a",
					AvatarUrl:  "a",
					OriginUrl:  "a",
				},
				Title:              "老王做空以太坊", // This matches data exp
				Content:            "老王做空以太坊详情",
				ImageUrls:          []string{"1", "4"},
				FilesUrls:          []string{"2", "3"},
				Tags:               []string{"Tesla", "中概股"},
				OriginUrl:          "bbb",
				ContentGeneratedAt: &timestamppb.Timestamp{},
			},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}

	t.Run("Test publish post with retweet sharing", func(t *testing.T) {
		// msgToTwoFeeds is from subsource 1 which in 2 feeds
		reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
			&msgToOneFeed,
		})
		msgs, _ := reader.ReceiveMessages(1)

		// Processing
		processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
		_, err := processor.ProcessOneCralwerMessage(msgs[0])
		require.Nil(t, err)

		// Checking process result
		var post model.Post
		processor.DB.Preload(clause.Associations).First(&post, "title = ?", msgToOneFeed.Post.Title)

		require.Equal(t, msgToOneFeed.Post.Title, post.Title)
		require.Equal(t, msgToOneFeed.Post.Content, post.Content)
		require.Equal(t, 1, len(post.PublishedFeeds))
		require.Equal(t, feedId, post.PublishedFeeds[0].Id)
		require.Equal(t, "电动车,港股", post.Tag)

		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.Title, post.SharedFromPost.Title)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.Content, post.SharedFromPost.Content)
		require.Equal(t, true, post.SharedFromPost.InSharingChain)
		require.Equal(t, 0, len(post.SharedFromPost.PublishedFeeds))
		require.Equal(t, "Tesla,中概股", post.SharedFromPost.Tag)

		// Check isFromSharedPost mark is set correctly
		var subScourceOrigin model.SubSource
		processor.DB.Preload(clause.Associations).Where("id=?", post.SubSourceID).First(&subScourceOrigin)
		require.False(t, subScourceOrigin.IsFromSharedPost)

		// Check new subsource is created
		var subScourceShared model.SubSource
		processor.DB.Preload(clause.Associations).Where("id=?", post.SharedFromPost.SubSourceID).First(&subScourceShared)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.SubSource.Name, subScourceShared.Name)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.SubSource.ExternalId, subScourceShared.ExternalIdentifier)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.SubSource.OriginUrl, subScourceShared.OriginUrl)
		require.Equal(t, msgToOneFeed.Post.SharedFromCrawledPost.SubSource.AvatarUrl, subScourceShared.AvatarUrl)
		require.True(t, subScourceShared.IsFromSharedPost)
	})
}

func TestRetweetMessageProcessSubsourceCreation(t *testing.T) {
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)
	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)

	msgOne := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "1",
			SubSource: &protocol.CrawledSubSource{
				// New subsource to be created and mark as isFromSharedPost
				Name:       "test_subsource_1",
				SourceId:   sourceId1,
				ExternalId: "a",
				AvatarUrl:  "a",
				OriginUrl:  "a",
			},
			Title:              "老王干得好", // This doesn't match data exp
			Content:            "老王干得好",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
			SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
				DeduplicateId: "2",
				SubSource: &protocol.CrawledSubSource{
					// New subsource to be created and mark as isFromSharedPost
					Name:       "test_subsource_2",
					SourceId:   sourceId1,
					ExternalId: "a",
					AvatarUrl:  "a",
					OriginUrl:  "a",
				},
				Title:              "老王做空以太坊", // This matches data exp
				Content:            "老王做空以太坊详情",
				ImageUrls:          []string{"1", "4"},
				FilesUrls:          []string{"2", "3"},
				Tags:               []string{"Tesla", "中概股"},
				OriginUrl:          "bbb",
				ContentGeneratedAt: &timestamppb.Timestamp{},
			},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}
	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&msgOne,
	})
	msgs, _ := reader.ReceiveMessages(1)
	processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
	_, err := processor.ProcessOneCralwerMessage(msgs[0])
	require.Nil(t, err)
	var subScourceOne model.SubSource
	var subScourceTwo model.SubSource
	processor.DB.Preload(clause.Associations).Where("name=?", "test_subsource_1").First(&subScourceOne)
	processor.DB.Preload(clause.Associations).Where("name=?", "test_subsource_2").First(&subScourceTwo)
	require.False(t, subScourceOne.IsFromSharedPost)
	require.True(t, subScourceTwo.IsFromSharedPost)

	msgTwo := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "3",
			SubSource: &protocol.CrawledSubSource{
				// Changing order of the two subsources
				Name:       "test_subsource_2",
				SourceId:   sourceId1,
				ExternalId: "a",
				AvatarUrl:  "a",
				OriginUrl:  "a",
			},
			Title:              "老王干得好_new_msg", //avoid dedup error
			Content:            "老王干得好_new_msg",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
			SharedFromCrawledPost: &protocol.CrawlerMessage_CrawledPost{
				DeduplicateId: "4",
				SubSource: &protocol.CrawledSubSource{
					// Changing order of the two subsources
					Name:       "test_subsource_1",
					SourceId:   sourceId1,
					ExternalId: "a",
					AvatarUrl:  "a",
					OriginUrl:  "a",
				},
				Title:              "老王做空以太坊_new_msg", //avoid dedup error
				Content:            "老王做空以太坊详情_new_msg",
				ImageUrls:          []string{"1", "4"},
				FilesUrls:          []string{"2", "3"},
				Tags:               []string{"Tesla", "中概股"},
				OriginUrl:          "bbb",
				ContentGeneratedAt: &timestamppb.Timestamp{},
			},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}
	reader = NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&msgTwo,
	})
	msgs, _ = reader.ReceiveMessages(1)
	processor = NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
	_, err = processor.ProcessOneCralwerMessage(msgs[0])
	require.Nil(t, err)
	processor.DB.Preload(clause.Associations).Where("name=?", "test_subsource_1").First(&subScourceOne)
	processor.DB.Preload(clause.Associations).Where("name=?", "test_subsource_2").First(&subScourceTwo)
	require.False(t, subScourceOne.IsFromSharedPost)
	require.False(t, subScourceTwo.IsFromSharedPost)
}

func TestMessagePublishToManyFeeds(t *testing.T) {
	db, _ := CreateTempDB(t)
	client := PrepareTestDBClient(db)
	uid := TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	sourceId1 := TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId1 := TestCreateSubSourceAndValidate(t, uid, "test_subsource_1", "test_externalid", sourceId1, false, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)
	TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", DataExpressionJsonForTest, []string{subSourceId1}, model.VisibilityPrivate, db, client)

	msgOne := protocol.CrawlerMessage{
		Post: &protocol.CrawlerMessage_CrawledPost{
			DeduplicateId: "1",
			SubSource: &protocol.CrawledSubSource{
				// New subsource to be created and mark as isFromSharedPost
				Name:       "test_subsource_1",
				SourceId:   sourceId1,
				ExternalId: "a",
				AvatarUrl:  "a",
				OriginUrl:  "a",
			},
			Title:              "老王做空以太坊", // This matches data exp
			Content:            "老王做空以太坊",
			ImageUrls:          []string{"1", "4"},
			FilesUrls:          []string{"2", "3"},
			Tags:               []string{"电动车", "港股"},
			OriginUrl:          "aaa",
			ContentGeneratedAt: &timestamppb.Timestamp{},
		},
		CrawledAt:      &timestamppb.Timestamp{},
		CrawlerIp:      "123",
		CrawlerVersion: "vde",
		IsTest:         false,
	}
	reader := NewTestMessageQueueReader([]*protocol.CrawlerMessage{
		&msgOne,
	})
	msgs, _ := reader.ReceiveMessages(1)
	processor := NewPublisherMessageProcessor(reader, db, deduplicator.FakeDeduplicatorClient{})
	_, err := processor.ProcessOneCralwerMessage(msgs[0])
	require.NoError(t, err)
	var post model.Post
	processor.DB.Preload(clause.Associations).Where("content=?", "老王做空以太坊").First(&post)
	require.Equal(t, 10, len(post.PublishedFeeds))
	require.NotEqual(t, post.PublishedFeeds[1].Id, post.PublishedFeeds[0].Id)
}
