package resolver

import (
	"errors"
	"fmt"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
)

const (
	feedRefreshLimit = 30
	defaultCursor    = -1
)

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
		if err := getOneFeedRefreshPosts(r, &feed, cursor, direction, limit); err != nil {
			return []*model.Feed{}, fmt.Errorf("failure when get posts for feed id %s", feedID)
		}
		results = append(results, &feed)
	}

	return results, nil
}

func getOneFeedRefreshPosts(r *queryResolver, feed *model.Feed, cursor int, direction model.FeedRefreshDirection, limit int) error {
	var posts []*model.Post
	if direction == model.FeedRefreshDirectionNew {
		r.DB.Model(&model.Post{}).
			Joins("LEFT JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").
			Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
			Where("feed_id = ? AND posts.cursor > ?", feed.Id, cursor).
			Order("cursor desc").
			Limit(limit).
			Find(&posts)
	} else {
		r.DB.Model(&model.Post{}).
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
