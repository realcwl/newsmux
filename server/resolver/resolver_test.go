package resolver

import (
	"fmt"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/Luismorlan/newsmux/utils/dotenv"
)

func TestMain(m *testing.M) {
	dotenv.LoadDotEnvsInTests()
	os.Exit(m.Run())
}

func PrepareTestForGraphQLAPIs(db *gorm.DB) *client.Client {
	client := client.New(handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &Resolver{
		DB:          db,
		SignalChans: NewSignalChannels(),
	}})))
	return client
}

func TestCreateUser(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	t.Run("Test User Creation", func(t *testing.T) {
		utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
	})

	// Test no double creation for the same id
	var resp struct {
		CreateUser struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"createUser"`
	}
	client.MustPost(fmt.Sprintf(`mutation {
		createUser(input:{name:"%s" id: "%s"}) {
			id
			name
		}
	}`, "test_user_name_new", "default_user_id"), &resp)

	require.NotEmpty(t, resp.CreateUser.Id)
	require.Equal(t, resp.CreateUser.Id, "default_user_id")
	require.Equal(t, resp.CreateUser.Name, "test_user_name")
}

func TestCreateFeed(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	t.Run("Test Feed Creation", func(t *testing.T) {
		uid := utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
		feedId, _ := utils.TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityGlobal, db, client)
		require.NotEmpty(t, feedId)
	})
}

func TestCreateSource(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	t.Run("Test Source Creation", func(t *testing.T) {
		uid := utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
		sourceId := utils.TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
		require.NotEmpty(t, sourceId)

		var source model.Source
		queryResult := db.Where("id = ?", sourceId).Preload("SubSources").First(&source)
		require.Equal(t, int64(1), queryResult.RowsAffected)
		require.Equal(t, 1, len(source.SubSources))
		require.Equal(t, DefaultSubSourceName, source.SubSources[0].Name)
	})
}

func TestUpsertSubSource(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	t.Run("Test Source Upsert", func(t *testing.T) {
		// Insert
		uid := utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
		sourceId := utils.TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
		subSourceId := utils.TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api", "test_externalid", sourceId, false, db, client)
		require.NotEmpty(t, subSourceId)

		// Update
		var subSource model.SubSource
		queryResult := db.Where("id = ?", subSourceId).First(&subSource)
		require.Equal(t, int64(1), queryResult.RowsAffected)
		subSource.Name = "NewName"
		subSource.AvatarUrl = "testing.com"
		subSource.OriginUrl = ""
		utils.TestUpdateSubSourceAndValidate(t, uid, &subSource, db, client)
	})
}

func TestQuerySubSource(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	t.Run("Test Source Query", func(t *testing.T) {
		// Insert
		uid := utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
		sourceId := utils.TestCreateSourceAndValidate(t, uid, "test_source_for_feeds_api", "test_domain", db, client)
		subSourceId := utils.TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api_1", "test_external_id_1", sourceId, false, db, client)
		require.NotEmpty(t, subSourceId)
		subSourceId2 := utils.TestCreateSubSourceAndValidate(t, uid, "test_subsource_for_feeds_api_2", "test_external_id_2", sourceId, true, db, client)
		require.NotEmpty(t, subSourceId2)

		subSources := utils.TestQuerySubSources(t, false, db, client)
		// There are two subsources, one is the "default" for the source, the other is test 1
		require.Equal(t, 2, len(subSources))
		require.Equal(t, "default", subSources[0].Name)

		require.Equal(t, "test_subsource_for_feeds_api_1", subSources[1].Name)
		require.Equal(t, "test_external_id_1", subSources[1].ExternalIdentifier)
		require.Equal(t, false, subSources[1].IsFromSharedPost)

		subSources = utils.TestQuerySubSources(t, true, db, client)
		// There are two subsources, one is the "default" for the source, the other is test 1
		require.Equal(t, 1, len(subSources))
		require.Equal(t, "test_subsource_for_feeds_api_2", subSources[0].Name)
		require.Equal(t, "test_external_id_2", subSources[0].ExternalIdentifier)
		require.Equal(t, true, subSources[0].IsFromSharedPost)
	})
}

func TestUserSubscribeFeed(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	t.Run("Test User subscribe Feed", func(t *testing.T) {
		userId := utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
		feedId1, _ := utils.TestCreateFeedAndValidate(t, userId, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityPrivate, db, client)
		utils.TestUserSubscribeFeedAndValidate(t, userId, feedId1, db, client)
		// Validate the first Feed's order
		subscription1 := &model.UserFeedSubscription{}
		db.Model(&model.UserFeedSubscription{}).
			Where("user_id = ? AND feed_id = ?", userId, feedId1).
			First(subscription1)
		require.Equal(t, subscription1.OrderInPanel, 0)

		feedId2, _ := utils.TestCreateFeedAndValidate(t, userId, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityPrivate, db, client)
		utils.TestUserSubscribeFeedAndValidate(t, userId, feedId2, db, client)
		// Validate the second Feed's order
		subscription2 := &model.UserFeedSubscription{}
		db.Model(&model.UserFeedSubscription{}).
			Where("user_id = ? AND feed_id = ?", userId, feedId2).
			First(subscription2)
		require.Equal(t, subscription2.OrderInPanel, 1)
	})
}

func TestSubscriberCount(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	userId1 := utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id_1", db, client)
	userId2 := utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id_2", db, client)
	feedId1, _ := utils.TestCreateFeedAndValidate(t, userId1, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityGlobal, db, client)
	utils.TestUserSubscribeFeedAndValidate(t, userId1, feedId1, db, client)
	utils.TestUserSubscribeFeedAndValidate(t, userId2, feedId1, db, client)
	utils.TestGetSubscriberCountAndValidate(t, feedId1, 2, db, client)
}

func TestDeleteFeed(t *testing.T) {
	db, _ := utils.CreateTempDB(t)
	client := PrepareTestForGraphQLAPIs(db)
	t.Run("Test User delete Feed", func(t *testing.T) {
		utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
		uid := utils.TestCreateUserAndValidate(t, "test_user_name", "default_user_id", db, client)
		feedId, _ := utils.TestCreateFeedAndValidate(t, uid, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityGlobal, db, client)
		utils.TestUserSubscribeFeedAndValidate(t, uid, feedId, db, client)
		utils.TestDeleteFeedAndValidate(t, uid, feedId, true, db, client)
	})

	t.Run("Test non owner delete Feed", func(t *testing.T) {
		uid1 := utils.TestCreateUserAndValidate(t, "test_user_name", "user_id_1", db, client)
		uid2 := utils.TestCreateUserAndValidate(t, "test_user_name", "user_id_2", db, client)
		feedId, _ := utils.TestCreateFeedAndValidate(t, uid1, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityGlobal, db, client)
		utils.TestUserSubscribeFeedAndValidate(t, uid1, feedId, db, client)
		utils.TestUserSubscribeFeedAndValidate(t, uid2, feedId, db, client)
		utils.TestDeleteFeedAndValidate(t, uid2, feedId, false, db, client)
	})
}

func TestQueryFeeds(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	userId := utils.TestCreateUserAndValidate(t, "test_user_for_feeds_api", "default_user_id", db, client)
	feedIdOne, updatedTimeOne := utils.TestCreateFeedAndValidate(t, userId, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityGlobal, db, client)
	feedIdTwo, updatedTimeTwo := utils.TestCreateFeedAndValidate(t, userId, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityPrivate, db, client)
	sourceId := utils.TestCreateSourceAndValidate(t, userId, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceId := utils.TestCreateSubSourceAndValidate(t, userId, "test_source_for_feeds_api", "123123213123", sourceId, false, db, client)
	utils.TestCreateSubSourceAndValidate(t, userId, "test_subsource_for_feeds_api", "test_externalid", sourceId, false, db, client)
	utils.TestUserSubscribeFeedAndValidate(t, userId, feedIdOne, db, client)
	utils.TestUserSubscribeFeedAndValidate(t, userId, feedIdTwo, db, client)

	// 0 is oldest post, 6 is newest post
	idWithReply, _ := utils.TestCreatePostAndValidate(t, "test_title_0", "test_content_0", subSourceId, feedIdOne, db, client)
	id_1, _ := utils.TestCreatePostAndValidate(t, "test_title_1", "test_content_1", subSourceId, feedIdOne, db, client)
	id_2, _ := utils.TestCreatePostAndValidate(t, "test_title_2", "test_content_2", subSourceId, feedIdOne, db, client)
	_, midCursorFirst := utils.TestCreatePostAndValidate(t, "test_title_3", "test_content_3", subSourceId, feedIdOne, db, client)
	utils.TestCreatePostAndValidate(t, "test_title_4", "test_content_4", subSourceId, feedIdOne, db, client)
	id_5, _ := utils.TestCreatePostAndValidate(t, "test_title_5", "test_content_5", subSourceId, feedIdOne, db, client)
	id_6, _ := utils.TestCreatePostAndValidate(t, "test_title_6", "test_content_6", subSourceId, feedIdOne, db, client)

	// 0 is oldest post, 6 is newest post
	utils.TestCreatePostAndValidate(t, "test_title_0", "test_content_0", subSourceId, feedIdTwo, db, client)
	utils.TestCreatePostAndValidate(t, "test_title_1", "test_content_1", subSourceId, feedIdTwo, db, client)
	utils.TestCreatePostAndValidate(t, "test_title_2", "test_content_2", subSourceId, feedIdTwo, db, client)
	_, midCursorSecond := utils.TestCreatePostAndValidate(t, "test_title_3", "test_content_3", subSourceId, feedIdTwo, db, client)
	utils.TestCreatePostAndValidate(t, "test_title_4", "test_content_4", subSourceId, feedIdTwo, db, client)
	utils.TestCreatePostAndValidate(t, "test_title_5", "test_content_5", subSourceId, feedIdTwo, db, client)
	utils.TestCreatePostAndValidate(t, "test_title_6", "test_content_6", subSourceId, feedIdTwo, db, client)

	// Create 2 posts belongs to post id 1 to test that we can query reply thread.
	reply_1, _ := utils.TestCreatePostAndValidate(t, "reply_title_1", "reply_content_1", subSourceId, "", db, client)
	reply_2, _ := utils.TestCreatePostAndValidate(t, "reply_title_2", "reply_content_2", subSourceId, "", db, client)
	utils.AddPostToReplyChain(db, idWithReply, []string{reply_1, reply_2})

	checkFeedPosts(t, userId, feedIdOne, midCursorFirst, 2, &updatedTimeOne, model.FeedRefreshDirectionNew,
		[]string{id_5, id_6}, nil, db, client)

	checkFeedPosts(t, userId, feedIdOne, midCursorSecond, 2, &updatedTimeOne, model.FeedRefreshDirectionOld,
		[]string{id_5, id_6}, nil, db, client)

	checkFeedPosts(t, userId, feedIdOne, midCursorFirst, 3, &updatedTimeOne, model.FeedRefreshDirectionOld,
		[]string{idWithReply, id_1, id_2}, map[string][]string{
			idWithReply: {
				reply_1, reply_2,
			},
		}, db, client)

	checkFeedTopPostsMultipleFeeds(t, userId, feedIdOne, feedIdTwo, midCursorFirst, midCursorSecond, updatedTimeOne, updatedTimeTwo, db, client)
	checkFeedBottomPostsMultipleFeeds(t, userId, feedIdOne, feedIdTwo, midCursorFirst, midCursorSecond, updatedTimeOne, updatedTimeTwo, db, client)
	checkFeedTopPostsWithoutSpecifyFeed(t, userId, feedIdOne, feedIdTwo, db, client)
	checkFeedTopPostsUpdateTimeChanged(t, userId, feedIdOne, midCursorFirst, "2021-08-24T21:57:15-07:00", db, client)
}

func checkFeedPosts(
	t *testing.T, userId string, feedId string, cursor int, limit int, updatedTime *string,
	direction model.FeedRefreshDirection, expectedPostsIds []string, postThread map[string][]string, db *gorm.DB, client *client.Client) {

	var resp struct {
		Feeds []struct {
			Id        string `json:"id"`
			UpdatedAt string `json:"updatedAt"`
			Posts     []struct {
				Id          string `json:"id"`
				Title       string `json:"title"`
				Content     string `json:"content"`
				Cursor      int    `json:"cursor"`
				ReplyThread []struct {
					Id      string `json:"id"`
					Title   string `json:"title"`
					Content string `json:"content"`
					Cursor  int    `json:"cursor"`
				} `json:"replyThread"`
			} `json:"posts"`
		} `json:"feeds"`
	}
	updatedTimeStr := `null`

	if updatedTime != nil {
		updatedTimeStr = fmt.Sprintf(`"%s"`, *updatedTime)
	}

	query := fmt.Sprintf(`
	query{
		feeds (input : {
		  userId : "%s"
		  feedRefreshInputs : [
			{feedId: "%s", limit: %d, cursor: %d, direction: %s, feedUpdatedTime: %s}
		  ]
		}) {
		  id
		  updatedAt
		  posts {
				id
				title
				content
				cursor
				replyThread {
					id
					title
					content
				}
		  }
		}
	}
	`, userId, feedId, limit, cursor, direction, updatedTimeStr)

	client.MustPost(query, &resp)

	require.Equal(t, 1, len(resp.Feeds))
	require.Equal(t, feedId, resp.Feeds[0].Id)
	require.Equal(t, len(expectedPostsIds), len(resp.Feeds[0].Posts))

	var postIds []string
	for _, post := range resp.Feeds[0].Posts {
		if postThread != nil {
			if thread, ok := postThread[post.Id]; ok {
				require.Equal(t, len(thread), len(post.ReplyThread))
				for i := 0; i < len(thread); i++ {
					require.Equal(t, thread[i], post.ReplyThread[i].Id)
				}
			}
		}

		postIds = append(postIds, post.Id)
	}

	require.True(t, utils.StringSlicesContainSameElements(postIds, expectedPostsIds))
}

func checkFeedTopPostsMultipleFeeds(
	t *testing.T, userId string, feedIdOne string, feedIdTwo string,
	cursorOne int, cursorTwo int, updatedTimeOne string, updatedTimeTwo string,
	db *gorm.DB, client *client.Client) {
	var resp struct {
		Feeds []struct {
			Id        string `json:"id"`
			UpdatedAt string `json:"updatedAt"`
			Posts     []struct {
				Id      string `json:"id"`
				Title   string `json:"title"`
				Content string `json:"content"`
				Cursor  int    `json:"cursor"`
			} `json:"posts"`
		} `json:"feeds"`
	}

	client.MustPost(fmt.Sprintf(`
	query{
		feeds (input : {
		  userId : "%s"
		  feedRefreshInputs : [
			{feedId: "%s", limit: %d, cursor: %d, direction: %s, feedUpdatedTime: "%s"}
			{feedId: "%s", limit: %d, cursor: %d, direction: %s, feedUpdatedTime: "%s"}
		  ]
		}) {
		  id
		  updatedAt
		  posts {
			id
			title
			content
			cursor
		  }
		}
	  }
	`, userId, feedIdOne, 2, cursorOne, model.FeedRefreshDirectionNew, updatedTimeOne,
		feedIdTwo, 2, cursorTwo, model.FeedRefreshDirectionNew, updatedTimeTwo), &resp)

	require.Equal(t, 2, len(resp.Feeds))
	require.Equal(t, feedIdOne, resp.Feeds[0].Id)
	require.Equal(t, 2, len(resp.Feeds[0].Posts))
	require.Equal(t, "test_title_6", resp.Feeds[0].Posts[0].Title)
	require.Equal(t, "test_title_5", resp.Feeds[0].Posts[1].Title)

	require.Equal(t, feedIdTwo, resp.Feeds[1].Id)
	require.Equal(t, 2, len(resp.Feeds[1].Posts))
	require.Equal(t, "test_title_6", resp.Feeds[1].Posts[0].Title)
	require.Equal(t, "test_title_5", resp.Feeds[1].Posts[1].Title)
}

func checkFeedBottomPostsMultipleFeeds(
	t *testing.T, userId string, feedIdOne string, feedIdTwo string,
	cursorOne int, cursorTwo int, updatedTimeOne string, updatedTimeTwo string,
	db *gorm.DB, client *client.Client) {
	var resp struct {
		Feeds []struct {
			Id        string `json:"id"`
			UpdatedAt string `json:"updatedAt"`
			Posts     []struct {
				Id      string `json:"id"`
				Title   string `json:"title"`
				Content string `json:"content"`
				Cursor  int    `json:"cursor"`
			} `json:"posts"`
		} `json:"feeds"`
	}

	client.MustPost(fmt.Sprintf(`
	query{
		feeds (input : {
		  userId : "%s"
		  feedRefreshInputs : [
			{feedId: "%s", limit: %d, cursor: %d, direction: %s, feedUpdatedTime: "%s"}
			{feedId: "%s", limit: %d, cursor: %d, direction: %s, feedUpdatedTime: "%s"}
		  ]
		}) {
		  id
		  updatedAt
		  posts {
			id
			title
			content
			cursor
		  }
		}
	  }
	`, userId, feedIdOne, 2, cursorOne, model.FeedRefreshDirectionOld, updatedTimeOne, feedIdTwo, 2, cursorTwo, model.FeedRefreshDirectionOld, updatedTimeTwo), &resp)

	require.Equal(t, 2, len(resp.Feeds))
	require.Equal(t, feedIdOne, resp.Feeds[0].Id)
	require.Equal(t, 2, len(resp.Feeds[0].Posts))
	require.Equal(t, "test_title_2", resp.Feeds[0].Posts[0].Title)
	require.Equal(t, "test_title_1", resp.Feeds[0].Posts[1].Title)

	require.Equal(t, feedIdTwo, resp.Feeds[1].Id)
	require.Equal(t, 2, len(resp.Feeds[1].Posts))
	require.Equal(t, "test_title_2", resp.Feeds[1].Posts[0].Title)
	require.Equal(t, "test_title_1", resp.Feeds[1].Posts[1].Title)
}

func checkFeedTopPostsWithoutSpecifyFeed(t *testing.T, userId string, feedIdOne string, feedIdTwo string, db *gorm.DB, client *client.Client) {
	var resp struct {
		Feeds []struct {
			Id        string `json:"id"`
			UpdatedAt string `json:"updatedAt"`
			Posts     []struct {
				Id      string `json:"id"`
				Title   string `json:"title"`
				Content string `json:"content"`
				Cursor  int    `json:"cursor"`
			} `json:"posts"`
		} `json:"feeds"`
	}

	client.MustPost(fmt.Sprintf(`
	query{
		feeds (input : {
		  userId : "%s"
		  feedRefreshInputs : []
		}) {
		  id
		  updatedAt
		  posts {
			id
			title
			content
			cursor
		  }
		}
	  }
	`, userId), &resp)

	require.Equal(t, 2, len(resp.Feeds))
	require.Equal(t, feedIdOne, resp.Feeds[0].Id)
	require.Equal(t, 7, len(resp.Feeds[0].Posts))
	require.Equal(t, "test_title_6", resp.Feeds[0].Posts[0].Title)
	require.Equal(t, "test_title_5", resp.Feeds[0].Posts[1].Title)
	require.Equal(t, "test_title_4", resp.Feeds[0].Posts[2].Title)
	require.Equal(t, "test_title_3", resp.Feeds[0].Posts[3].Title)
	require.Equal(t, "test_title_2", resp.Feeds[0].Posts[4].Title)
	require.Equal(t, "test_title_1", resp.Feeds[0].Posts[5].Title)
	require.Equal(t, "test_title_0", resp.Feeds[0].Posts[6].Title)

	require.Equal(t, feedIdTwo, resp.Feeds[1].Id)
	require.Equal(t, 7, len(resp.Feeds[1].Posts))
	require.Equal(t, "test_title_6", resp.Feeds[1].Posts[0].Title)
	require.Equal(t, "test_title_5", resp.Feeds[1].Posts[1].Title)
	require.Equal(t, "test_title_4", resp.Feeds[1].Posts[2].Title)
	require.Equal(t, "test_title_3", resp.Feeds[1].Posts[3].Title)
	require.Equal(t, "test_title_2", resp.Feeds[1].Posts[4].Title)
	require.Equal(t, "test_title_1", resp.Feeds[1].Posts[5].Title)
	require.Equal(t, "test_title_0", resp.Feeds[1].Posts[6].Title)
}

func checkFeedTopPostsUpdateTimeChanged(t *testing.T, userId string, feedId string, cursor int, wrongUpdatedTime string, db *gorm.DB, client *client.Client) {
	var resp struct {
		Feeds []struct {
			Id        string `json:"id"`
			UpdatedAt string `json:"updatedAt"`
			Posts     []struct {
				Id      string `json:"id"`
				Title   string `json:"title"`
				Content string `json:"content"`
				Cursor  int    `json:"cursor"`
			} `json:"posts"`
		} `json:"feeds"`
	}

	client.MustPost(fmt.Sprintf(`
		query{
			feeds (input : {
			  userId : "%s"
			  feedRefreshInputs : [
				{feedId: "%s", limit: %d, cursor: %d, direction: %s, feedUpdatedTime: "%s"}
			  ]
			}) {
			  id
			  updatedAt
			  posts {
				id
				title
				content
				cursor
			  }
			}
		  }
		`, userId, feedId, 7, cursor, model.FeedRefreshDirectionNew, wrongUpdatedTime), &resp)

	require.Equal(t, 1, len(resp.Feeds))
	require.Equal(t, feedId, resp.Feeds[0].Id)
	require.Equal(t, 7, len(resp.Feeds[0].Posts))
}

func TestUpSertFeedsAndRepublish(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	userId := utils.TestCreateUserAndValidate(t, "test_user_for_feeds_api", "default_user_id", db, client)
	sourceId := utils.TestCreateSourceAndValidate(t, userId, "test_source_for_feeds_api", "test_domain", db, client)
	subSourceIdOne := utils.TestCreateSubSourceAndValidate(t, userId, "test_source_for_feeds_api", "1111", sourceId, false, db, client)
	subSourceIdTwo := utils.TestCreateSubSourceAndValidate(t, userId, "test_source_for_feeds_api_2", "2222", sourceId, false, db, client)
	feedIdOne, _ := utils.TestCreateFeedAndValidate(t, userId, "test_feed_for_feeds_api", ``, []string{}, model.VisibilityGlobal, db, client)

	postId1, _ := utils.TestCreatePostAndValidate(t, "test_title_1", "same_content_test", subSourceIdOne, feedIdOne, db, client)
	postId2, _ := utils.TestCreatePostAndValidate(t, "test_title_2", "test_content_2", subSourceIdOne, feedIdOne, db, client)

	postId3, _ := utils.TestCreatePostAndValidate(t, "test_title_3", "same_content_test", subSourceIdTwo, feedIdOne, db, client)
	postId4, _ := utils.TestCreatePostAndValidate(t, "test_title_4", "test_content_4", subSourceIdTwo, feedIdOne, db, client)
	postId5, cursor5 := utils.TestCreatePostAndValidate(t, "test_title_5", "test_content_5", subSourceIdTwo, feedIdOne, db, client)

	t.Run("use {upsertFeed} to change subsource, should clear posts, re-publish when query {feeds}", func(t *testing.T) {
		var (
			feed         model.Feed
			subSourceOne model.SubSource
		)
		queryResult := db.Preload("SubSources").Where("id = ?", feedIdOne).First(&feed)
		require.Equal(t, int64(1), queryResult.RowsAffected)
		queryResult = db.Where("id = ?", subSourceIdOne).First(&subSourceOne)
		require.Equal(t, int64(1), queryResult.RowsAffected)

		feed.SubSources = []*model.SubSource{
			&subSourceOne,
		}
		utils.TestUpdateFeed(t, feed, db, client)
		checkFeedPosts(t, userId, feedIdOne, 0, 999, nil, model.FeedRefreshDirectionNew,
			[]string{postId1, postId2}, nil, db, client)
	})

	t.Run("use {upsertFeed} to change subsource, should clear posts, re-publish when query {feeds}", func(t *testing.T) {
		var (
			feed         model.Feed
			subSourceOne model.SubSource
			subSourceTwo model.SubSource
		)
		queryResult := db.Preload("SubSources").Where("id = ?", feedIdOne).First(&feed)
		require.Equal(t, int64(1), queryResult.RowsAffected)
		queryResult = db.Where("id = ?", subSourceIdOne).First(&subSourceOne)
		require.Equal(t, int64(1), queryResult.RowsAffected)
		queryResult = db.Where("id = ?", subSourceIdTwo).First(&subSourceTwo)
		require.Equal(t, int64(1), queryResult.RowsAffected)

		feed.SubSources = []*model.SubSource{
			&subSourceOne,
			&subSourceTwo,
		}
		utils.TestUpdateFeed(t, feed, db, client)
		checkFeedPosts(t, userId, feedIdOne, 0, 999, nil, model.FeedRefreshDirectionNew,
			[]string{postId1, postId2, postId3, postId4, postId5}, nil, db, client)
	})
	t.Run("update data expression for feed, should clear posts, re-publish when query {feeds}", func(t *testing.T) {
		var feed model.Feed
		queryResult := db.Preload("SubSources").Where("id = ?", feedIdOne).First(&feed)
		require.Equal(t, int64(1), queryResult.RowsAffected)

		feed.FilterDataExpression = datatypes.JSON(
			`{
			"id":"1",
			"expr":{
				"pred":{
				"type":"LITERAL",
				"param":{
					"text":"same_content_test"
				}
				}
			}
	 	}`)
		utils.TestUpdateFeed(t, feed, db, client)
		checkFeedPosts(t, userId, feedIdOne, 0, 999, nil, model.FeedRefreshDirectionNew,
			[]string{postId1, postId3}, nil, db, client)
	})
	t.Run("update data expression for feed, should clear posts, re-publish when query {feeds} OLD and NEW", func(t *testing.T) {
		var feed model.Feed
		queryResult := db.Preload("SubSources").Where("id = ?", feedIdOne).First(&feed)
		require.Equal(t, int64(1), queryResult.RowsAffected)
		feed.FilterDataExpression = datatypes.JSON(``)

		// publish more by querying {feeds} with NEW
		updatedAt := utils.TestUpdateFeed(t, feed, db, client)
		checkFeedPosts(t, userId, feedIdOne, 0, 1, nil, model.FeedRefreshDirectionNew,
			[]string{postId5}, nil, db, client)

		// check only 1 post is published
		var count int64
		db.Model(&model.PostFeedPublish{}).Where("feed_id = ?", feedIdOne).Count(&count)
		require.Equal(t, int64(1), count)

		// publish more by querying {feeds} with OLD
		checkFeedPosts(t, userId, feedIdOne, cursor5, 2, &updatedAt, model.FeedRefreshDirectionOld,
			[]string{postId4, postId3}, nil, db, client)

		// check only 3 post is published now after republishing
		db.Model(&model.PostFeedPublish{}).Where("feed_id = ?", feedIdOne).Count(&count)
		require.Equal(t, int64(3), count)
	})
	t.Run("update data expression for feed, should clear posts, should avoid republish retweeted posts", func(t *testing.T) {
		var feed model.Feed
		queryResult := db.Preload("SubSources").Where("id = ?", feedIdOne).First(&feed)
		require.Equal(t, int64(1), queryResult.RowsAffected)
		feed.FilterDataExpression = datatypes.JSON(``)

		subSourceWithNestedPostId := utils.TestCreateSubSourceAndValidate(t, userId, "test_source_for_feeds_api_3", "3333", sourceId, false, db, client)
		var subSourceWithNestedPost model.SubSource
		queryResult = db.Where("id = ?", subSourceWithNestedPostId).First(&subSourceWithNestedPost)
		require.Equal(t, int64(1), queryResult.RowsAffected)

		var (
			postOrigin, postCommnet model.Post
		)
		postOriginId, _ := utils.TestCreatePostAndValidate(t, "post origin", "test", subSourceWithNestedPostId, "", db, client)
		db.Where("id = ?", postOriginId).First(&postOrigin)
		postCommnetId, _ := utils.TestCreatePostAndValidate(t, "post comment", "test", subSourceWithNestedPostId, "", db, client)
		db.Where("id = ?", postCommnetId).First(&postCommnet)

		postOrigin.InSharingChain = true
		db.Save(postOrigin)

		postCommnet.SharedFromPost = &postOrigin
		postCommnet.SharedFromPostID = &postOrigin.Id
		db.Save(postCommnet)

		feed.SubSources = []*model.SubSource{
			&subSourceWithNestedPost,
		}
		utils.TestUpdateFeed(t, feed, db, client)
		checkFeedPosts(t, userId, feedIdOne, 0, 999, nil, model.FeedRefreshDirectionNew,
			[]string{postCommnetId}, nil, db, client)
	})
}

func TestUserState(t *testing.T) {
	db, _ := utils.CreateTempDB(t)

	client := PrepareTestForGraphQLAPIs(db)

	userId := utils.TestCreateUserAndValidate(t, "test_user_for_feeds_api", "default_user_id", db, client)
	feedIdOne, _ := utils.TestCreateFeedAndValidate(t, userId, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityGlobal, db, client)
	feedIdTwo, _ := utils.TestCreateFeedAndValidate(t, userId, "test_feed_for_feeds_api", `{\"a\":1}`, []string{}, model.VisibilityPrivate, db, client)
	utils.TestUserSubscribeFeedAndValidate(t, userId, feedIdOne, db, client)
	utils.TestUserSubscribeFeedAndValidate(t, userId, feedIdTwo, db, client)

	utils.TestQueryUserState(t, userId, []string{feedIdOne, feedIdTwo}, client)
}

func TestConstructCustomizedPanopticConfig(t *testing.T) {
	var input model.NewSourceInput
	input.Name = "test_source_name"
	startImmediately := false
	imageRelativeSelector := "div > .img"
	input.CustomizedCrawlerPanopticConfigForm = &model.CustomizedCrawlerPanopticConfigForm{
		StartImmediately: &startImmediately,
		CustomizedCrawlerParams: &model.CustomizedCrawlerParams{
			BaseSelector:          "base_selector",
			CrawlURL:              "url",
			ImageRelativeSelector: &imageRelativeSelector,
		},
	}
	config, err := ConstructCustomizedPanopticConfig(input, "test_source_id")
	require.NoError(t, err)
	require.Equal(t, "test_source_id", config.TaskParams.SourceId)
	require.Equal(t, "test_source_name_config", config.Name)
	require.Equal(t, false, config.TaskSchedule.StartImmediatly)
	require.Equal(t, int64(5*60*1000), config.TaskSchedule.GetRoutinely().EveryMilliseconds)
	require.Equal(t, "base_selector", config.TaskParams.GetCustomizedSourceCrawlerTaskParams().BaseSelector)
	require.Equal(t, "url", config.TaskParams.GetCustomizedSourceCrawlerTaskParams().CrawlUrl)
	require.Equal(t, "div > .img", *config.TaskParams.GetCustomizedSourceCrawlerTaskParams().ImageRelativeSelector)
	require.Equal(t, protocol.PanopticTask_COLLECTOR_USER_CUSTOMIZED_SOURCE, config.DataCollectorId)
}
