package resolver

import (
	"errors"
	"fmt"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
	"gorm.io/gorm"
)

const (
	feedRefreshLimit      = 30
	defaultCursor         = -1
	maxRepublishDBBatches = 5
)

// Redo posts publish to feeds
// From a particular cursor down
// If cursor is -1, republish whole feeds
func rePublishPostsFromCursor(db *gorm.DB, feed *model.Feed, limit int, fromCursor int) {
	var (
		postsToPublish []*model.Post
		batches        = 0
	)

	limit = utils.Min(feedRefreshLimit, limit)

	if fromCursor == -1 {
		fromCursor = 2147483647
	}

	var subsourceIds []string
	for _, subsource := range feed.SubSources {
		subsourceIds = append(subsourceIds, subsource.Id)
	}

	for {
		if len(postsToPublish) >= limit || batches > maxRepublishDBBatches {
			break
		}
		var postsCandidates []model.Post
		// 1. Read subsources' most recent posts
		db.Model(&model.Post{}).
			Joins("LEFT JOIN sub_sources ON posts.sub_source_id = sub_sources.id").
			Where("sub_sources.id IN ? AND posts.cursor < ?", subsourceIds, fromCursor).
			Order("cursor desc").
			Limit(limit).
			Find(&postsCandidates)

		// 2. Try match postsCandidate with Feed
		for ind := range postsCandidates {
			post := postsCandidates[ind]
			fromCursor = int(post.Cursor)
			matched, error := utils.DataExpressionMatchPost(string(feed.FilterDataExpression), post)
			if error != nil {
				continue
			}
			if matched {
				postsToPublish = append(postsToPublish, &post)
				// to publish exact same number of posts queried
				if len(postsToPublish) >= limit {
					break
				}
			}
		}
		batches = batches + 1
	}

	// This call will also update feed object with posts, no need to append
	db.Model(feed).UpdateColumns(model.Feed{UpdatedAt: feed.UpdatedAt}).Association("Posts").Append(postsToPublish)
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
		queryResult := r.DB.Preload("SubSources").Where("id = ?", feedID).First(&feed)
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
		limit := query.Limit
		if limit < 0 || limit > feedRefreshLimit {
			limit = feedRefreshLimit
		}

		if err := getFeedPostsOrRePublish(r.DB, &feed, cursor, direction, limit); err != nil {
			return []*model.Feed{}, fmt.Errorf("failure when get posts for feed id %s", feedID)
		}
		results = append(results, &feed)
	}

	return results, nil
}

func getFeedPostsOrRePublish(db *gorm.DB, feed *model.Feed, cursor int, direction model.FeedRefreshDirection, limit int) error {
	var posts []*model.Post
	// try to read published posts
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

	// cases where we need republish first
	if direction == model.FeedRefreshDirectionNew {
		// query NEW but publish table is empty
		var count int64
		db.Model(&model.PostFeedPublish{}).
			Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
			Where("feed_id = ?", feed.Id).
			Count(&count)
		if count == 0 {
			rePublishPostsFromCursor(db, feed, limit, -1)
		} else {
			fmt.Println("NOT REPUBLISHING", count)
		}
	} else {
		// query OLD but can't satisfy the limit
		lastCursor := cursor
		if len(posts) < limit {
			if len(posts) > 0 {
				lastCursor = int(posts[len(posts)-1].Cursor)
			}
			// republish to fulfill the query limit
			rePublishPostsFromCursor(db, feed, limit-len(posts), lastCursor)
		}
	}
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
