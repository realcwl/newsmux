package resolver

import (
	"errors"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
)

const (
	feedRefreshLimit = 30
	defaultCursor    = -1
)

func getRefreshPosts(r *queryResolver, query []*model.FeedRefreshInput) ([]*model.FeedOutput, error) {
	result := []*model.FeedOutput{}

	for _, refreshInput := range query {
		// TODO why can input be null?
		if refreshInput == nil {
			continue
		}

		var (
			feedID    = refreshInput.FeedID
			cursor    = refreshInput.Cursor
			direction = refreshInput.Direction
			limit     = utils.Min(feedRefreshLimit, refreshInput.Limit)
		)

		type FeedPostQueryResult struct {
			model.Post
			Cursor int
		}

		var postsWithCursors []*FeedPostQueryResult

		if direction == model.FeedRefreshDirectionTop {
			r.DB.Debug().Model(&model.Post{}).Joins("JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").Order("cursor desc").Limit(limit).Where("feed_id = ? AND cursor > ?", feedID, cursor).Find(&postsWithCursors)
		} else {
			r.DB.Debug().Model(&model.Post{}).Joins("JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").Order("cursor desc").Limit(limit).Where("feed_id = ? AND cursor < ?", feedID, cursor).Find(&postsWithCursors)
		}

		feedResult := model.FeedOutput{
			FeedID: feedID,
			Posts:  []*model.PostInFeedOutput{},
		}

		for _, r := range postsWithCursors {
			feedResult.Posts = append(feedResult.Posts, &model.PostInFeedOutput{
				Post:   &r.Post,
				Cursor: r.Cursor,
			})
		}

		result = append(result, &feedResult)
	}

	return result, nil
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
