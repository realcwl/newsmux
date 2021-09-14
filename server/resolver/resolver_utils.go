package resolver

import (
	"fmt"
	"math"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	feedRefreshLimit           = 30
	defaultFeedsQueryCursor    = math.MaxInt32
	defaultFeedsQueryDirection = model.FeedRefreshDirectionOld
	maxRepublishDBBatches      = 10
)

// Given a list of FeedRefreshInput, get posts for the requested feeds
// Do it by iterating through feeds
func getRefreshPosts(r *queryResolver, queries []*model.FeedRefreshInput) ([]*model.Feed, error) {
	results := []*model.Feed{}

	//TODO: can be run in parallel
	for idx, _ := range queries {
		query := queries[idx]
		if query == nil {
			// This is not expected since gqlgen guarantees it is not nil
			continue
		}
		// Prepare feed basic info
		var feed model.Feed
		queryResult := r.DB.Preload("SubSources").Where("id = ?", query.FeedID).First(&feed)
		if queryResult.RowsAffected != 1 {
			return []*model.Feed{}, fmt.Errorf("invalid feed id %s", query.FeedID)
		}
		if err := sanitizeFeedRefreshInput(query, &feed); err != nil {
			return []*model.Feed{}, errors.Wrap(err, fmt.Sprint("feed query invalid ", query))
		}
		if err := getFeedPostsOrRePublish(r.DB, &feed, query); err != nil {
			return []*model.Feed{}, errors.Wrap(err, fmt.Sprint("failure when get posts for feed id ", feed.Id))
		}
		results = append(results, &feed)
	}

	return results, nil
}

func getFeedPostsOrRePublish(db *gorm.DB, feed *model.Feed, query *model.FeedRefreshInput) error {
	var posts []*model.Post
	// try to read published posts
	Log.Info("read published post for feed: ", feed.Id, " query: ", query)
	if query.Direction == model.FeedRefreshDirectionNew {
		db.Model(&model.Post{}).
			Preload(clause.Associations).
			Preload("SharedFromPost.SubSource").
			Joins("LEFT JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").
			Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
			Where("feed_id = ? AND posts.cursor > ?", feed.Id, query.Cursor).
			Order("content_generated_at desc").
			Limit(query.Limit).
			Find(&posts)
		feed.Posts = posts
	} else {
		db.Model(&model.Post{}).
			Preload(clause.Associations).
			Preload("SharedFromPost.SubSource").
			Joins("LEFT JOIN post_feed_publishes ON post_feed_publishes.post_id = posts.id").
			Joins("LEFT JOIN feeds ON post_feed_publishes.feed_id = feeds.id").
			Where("feed_id = ? AND posts.cursor < ?", feed.Id, query.Cursor).
			Order("content_generated_at desc").
			Limit(query.Limit).
			Find(&posts)
		feed.Posts = posts

		if len(posts) < query.Limit {
			// query OLD but can't satisfy the limit, republish in this case
			lastCursor := query.Cursor
			if len(posts) > 0 {
				lastCursor = int(posts[len(posts)-1].Cursor)
			}
			Log.Info("run ondemand publish posts to feed: ", feed.Id, " triggered by NEW in {feeds} API from curosr ", lastCursor)
			rePublishPostsFromCursor(db, feed, query.Limit-len(posts), lastCursor)
		}
	}
	return nil
}

// Redo posts publish to feeds
// From a particular cursor down
// If cursor is -1, republish from NEWest
func rePublishPostsFromCursor(db *gorm.DB, feed *model.Feed, limit int, fromCursor int) {
	var (
		postsToPublish []*model.Post
		batches        = 0
	)

	limit = utils.Min(feedRefreshLimit, limit)

	var subsourceIds []string
	for _, subsource := range feed.SubSources {
		subsourceIds = append(subsourceIds, subsource.Id)
	}

	for len(postsToPublish) < limit && batches <= maxRepublishDBBatches {
		var postsCandidates []*model.Post
		// 1. Read subsources' most recent posts
		// 2. skip if post is shared by another one, this used to handle case as retweet
		// 	  this will also work, if in future we will support user generate comments on other user posts
		//    the shared post creation and publish is in one transaction, so the sharing can only happen
		//    after the shared one is published.
		//    however for re-publish,
		db.Model(&model.Post{}).
			Preload(clause.Associations).
			Preload("SharedFromPost.SubSource").
			Joins("LEFT JOIN sub_sources ON posts.sub_source_id = sub_sources.id").
			Where("sub_sources.id IN ? AND posts.cursor < ? AND (NOT posts.in_sharing_chain)", subsourceIds, fromCursor).
			Order("content_generated_at desc").
			Limit(limit).
			Find(&postsCandidates)

		// 2. Try match postsCandidate with Feed
		for idx := range postsCandidates {
			post := postsCandidates[idx]
			fromCursor = utils.Min(fromCursor, int(post.Cursor))
			matched, error := utils.DataExpressionMatchPostChain(string(feed.FilterDataExpression), post)
			if error != nil {
				continue
			}
			if matched {
				postsToPublish = append(postsToPublish, post)
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

// get all feeds a user subscribed
func getUserSubscriptions(r *queryResolver, userID string) ([]*model.Feed, error) {
	var user model.User
	queryResult := r.DB.Where("id = ?", userID).Preload("SubscribedFeeds").First(&user)
	if queryResult.RowsAffected != 1 {
		return nil, errors.New("User not found")
	}
	return user.SubscribedFeeds, nil
}

func sanitizeFeedRefreshInput(query *model.FeedRefreshInput, feed *model.Feed) error {
	if query.Cursor < 0 {
		return errors.New("query.Cursor should be >= 0")
	}

	if query.Limit <= 0 {
		return errors.New("query.Limit should be > 0")
	}

	// Check if requested cursors are out of sync from last feed update
	// If out of sync, default to query latest posts
	// Use unix() to avoid accuracy loss due to gqlgen serialization impacting matching
	if query.FeedUpdatedTime == nil || query.FeedUpdatedTime.Unix() != feed.UpdatedAt.Unix() {
		Log.Info(
			"requested with outdated feed updated time, feed_id=", feed.Id,
			" query updated time=", query.FeedUpdatedTime,
			" feed updated at=", feed.UpdatedAt)
		query.Cursor = defaultFeedsQueryCursor
		query.Direction = defaultFeedsQueryDirection
	}

	// Cap query limit
	if query.Limit > feedRefreshLimit {
		query.Limit = feedRefreshLimit
	}

	return nil
}

func isClearPostsNeededForFeedsUpsert(feed *model.Feed, input *model.UpsertFeedInput) (bool, error) {
	var subsourceIds []string
	for _, subsource := range feed.SubSources {
		subsourceIds = append(subsourceIds, subsource.Id)
	}
	dataExpressionMatched, err := utils.AreJSONsEqual(feed.FilterDataExpression.String(), input.FilterDataExpression)
	if err != nil {
		return false, err
	}

	if !dataExpressionMatched || !utils.StringSlicesContainSameElements(subsourceIds, input.SubSourceIds) {
		return true, nil
	}

	return false, nil
}

func UpsertSubsourceImpl(db *gorm.DB, input model.UpsertSubSourceInput) (*model.SubSource, error) {
	var subSource model.SubSource
	queryResult := db.Preload("Feeds").
		Where("name = ? AND source_id = ?", input.Name, input.SourceID).
		First(&subSource)
	if queryResult.RowsAffected == 0 {
		// Create new SubSource
		subSource = model.SubSource{
			Id:                 uuid.New().String(),
			Name:               input.Name,
			ExternalIdentifier: input.ExternalIdentifier,
			SourceID:           input.SourceID,
			AvatarUrl:          input.AvatarURL,
			OriginUrl:          input.OriginURL,
			IsFromSharedPost:   input.IsFromSharedPost,
		}
		db.Create(&subSource)
		return &subSource, nil
	}
	// Update existing SubSource
	subSource.ExternalIdentifier = input.ExternalIdentifier
	subSource.AvatarUrl = input.AvatarURL
	subSource.OriginUrl = input.OriginURL
	// udpate won't udpate IsFromShare,
	// to prevent an already needed subsource got shared, and become IsFromSharedPost = true
	db.Save(&subSource)

	return &subSource, nil
}
