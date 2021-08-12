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
func getRefreshPosts(r *queryResolver, query []*model.FeedRefreshInput) ([]*model.Feed, error) {
	results := []*model.Feed{}

	//TODO: can be run in parallel
	for _, refreshInput := range query {
		// TODO why can input be null?
		if refreshInput == nil {
			continue
		}

		type FeedPostQueryResult struct {
			model.Post
			Cursor int
		}

		var feed model.Feed

		feedID := refreshInput.FeedID
		cursor := refreshInput.Cursor
		direction := refreshInput.Direction
		limit := utils.Min(feedRefreshLimit, refreshInput.Limit)

		queryResult := r.DB.Where("id = ?", feedID).First(&feed)
		if queryResult.RowsAffected != 1 {
			// TODO: add to datadog
			return nil, errors.New(fmt.Sprintf("Invalid feed id %s", feedID))
		}

		var posts []*model.Post
		if direction == model.FeedRefreshDirectionNew {
			r.DB.Debug().Model(&model.Post{}).
				Joins("LEFT JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").
				Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
				Where("feed_id = ? AND posts.cursor > ?", feedID, cursor).
				Order("cursor desc").
				Limit(limit).
				Find(&posts)
		} else {
			r.DB.Debug().Model(&model.Post{}).
				Joins("LEFT JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").
				Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
				Where("feed_id = ? AND posts.cursor < ?", feedID, cursor).
				Order("cursor desc").
				Limit(limit).
				Find(&posts)
		}
		feed.Posts = posts
		results = append(results, &feed)
	}

	return results, nil
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
