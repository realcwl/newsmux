package resolver

import (
	"errors"
	"fmt"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/publisher"
	"github.com/Luismorlan/newsmux/utils"
	"gorm.io/gorm"
)

const (
	feedRefreshLimit = 30
	defaultCursor    = -1
)

// Redo posts publish to feeds
// This is done by load all posts in subsources in a feed
// And go over each post check if match with feed's data expression
//
func rePublishPostsForFeed(db *gorm.DB, feed *model.Feed, input model.UpsertFeedInput, limit int, maxDBLookupBatches int) {
	var (
		postsToPublish []model.Post
		batches        = 0
		oldestCursor   = 2147483647
	)

	for {
		var postsCandidates []model.Post
		// 1. Read subsources' most recent posts
		db.Debug().Model(&model.Post{}).
			Joins("LEFT JOIN sub_sources ON posts.sub_source_id = sub_sources.id").
			Where("sub_sources.id IN ? AND posts.cursor < ?", input.SubSourceIds, oldestCursor).
			Order("cursor desc").
			Limit(limit).
			Find(&postsCandidates)

		fmt.Println(postsCandidates)
		fmt.Println(len(postsCandidates))

		// 2. Try match postsCandidate with Feed
		for _, post := range postsCandidates {
			oldestCursor = int(post.Cursor)
			matched, error := publisher.DataExpressionMatchPost(input.FilterDataExpression, post)
			if error != nil {
				continue
			}
			if matched {
				postsToPublish = append(postsToPublish, post)
			}
		}

		if len(postsToPublish) >= limit || batches > maxDBLookupBatches {
			break
		}
		batches = batches + 1
	}

	db.Model(&feed).Association("Posts").Delete()
	db.Model(&feed).Association("Posts").Replace(postsToPublish)
}

// Given a list of FeedRefreshInput, get posts for the requested feeds
// Do it by iterating through feeds
func getRefreshPosts(r *queryResolver, queries []*model.FeedRefreshInput) ([]*model.Feed, error) {
	results := []*model.Feed{}

	//TODO: can be run in parallel
	for _, query := range queries {
		if query == nil {
			// This is not expected since gqlgen guarantees it is not nil
			continue
		}

		// Prepare feed basic info
		var (
			feed      model.Feed
			feedID    = query.FeedID
			cursor    = query.Cursor
			direction = query.Direction
		)
		queryResult := r.DB.Where("id = ?", feedID).First(&feed)
		if queryResult.RowsAffected != 1 {
			return []*model.Feed{}, fmt.Errorf("invalid feed id %s", feedID)
		}

		// Check if requested cursors are out of sync from last feed update
		// If out of sync, default to query latest posts
		// Use unix() to avoid accuracy loss due to gqlgen serialization impacting matching
		if query.FeedUpdatedTime == nil || query.FeedUpdatedTime.Unix() != feed.UpdatedAt.Unix() {
			cursor = -1
			direction = model.FeedRefreshDirectionNew
		}

		// Fill in posts
		limit := utils.Min(feedRefreshLimit, query.Limit)
		if err := getOneFeedRefreshPosts(r.DB, &feed, cursor, direction, limit); err != nil {
			return []*model.Feed{}, fmt.Errorf("failure when get posts for feed id %s", feedID)
		}
		results = append(results, &feed)
	}

	return results, nil
}

func getOneFeedRefreshPosts(db *gorm.DB, feed *model.Feed, cursor int, direction model.FeedRefreshDirection, limit int) error {
	var posts []*model.Post
	if direction == model.FeedRefreshDirectionNew {
		db.Model(&model.Post{}).
			Joins("LEFT JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").
			Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
			Where("feed_id = ? AND posts.cursor > ?", feed.Id, cursor).
			Order("cursor desc").
			Limit(limit).
			Find(&posts)
	} else {
		db.Model(&model.Post{}).
			Joins("LEFT JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").
			Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
			Where("feed_id = ? AND posts.cursor < ?", feed.Id, cursor).
			Order("cursor desc").
			Limit(limit).
			Find(&posts)
	}
	feed.Posts = posts
	return nil
}

// get all feeds a user subscribed
func getUserSubscriptions(r *queryResolver, userID string) ([]*model.Feed, error) {
	var user model.User
	queryResult := r.DB.Where("id = ?", userID).Preload("SubscribedFeeds").First(&user)
	if queryResult.RowsAffected != 1 {
		return nil, errors.New("User not found")
	}
	return user.SubscribedFeeds, nil
}
