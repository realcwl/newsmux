package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *mutationResolver) CreateUser(ctx context.Context, input model.NewUserInput) (*model.User, error) {
	var user model.User
	res := r.DB.Model(&model.User{}).Where("id = ?", input.ID).First(&user)
	if res.RowsAffected == 0 {
		// if the user doesn't exist, create the user.
		t := model.User{
			Id:   input.ID,
			Name: input.Name,
			// TODO(chenweilunster): For now we set only default user avatar, later
			// we'll allow user to customize their avatar in the frontend.
			AvatarUrl:       "https://robohash.org/54a9068a8750731226a284514c01b0bb?set=set4&bgset=&size=400x400",
			CreatedAt:       time.Now(),
			SubscribedFeeds: []*model.Feed{},
		}
		r.DB.Create(&t)
		return &t, nil
	}

	// otherwise
	return &user, nil
}

func (r *mutationResolver) UpsertFeed(ctx context.Context, input model.UpsertFeedInput) (*model.Feed, error) {
	// Upsert a feed
	// return feed with updated posts
	var (
		user           model.User
		feed           model.Feed
		needClearPosts = true
	)

	// get creator user
	userID := input.UserID
	queryResult := r.DB.Where("id = ?", userID).First(&user)
	if queryResult.RowsAffected != 1 {
		return nil, errors.New("invalid user id")
	}

	if input.FeedID != nil {
		// If it is update:
		// 1. read from DB
		queryResult := r.DB.Where("id = ?", *input.FeedID).Preload("SubSources").Preload("Posts").First(&feed)
		if queryResult.RowsAffected != 1 {
			return nil, errors.New("invalid feed id")
		}

		// 2. check if dropping posts is needed
		var err error
		needClearPosts, err = isClearPostsNeededForFeedsUpsert(&feed, &input)
		if err != nil {
			return nil, err
		}

		// Update feed object
		feed.Name = input.Name
		feed.Creator = user
		feed.FilterDataExpression = datatypes.JSON(input.FilterDataExpression)
	} else {
		// If it is insert, create feed object
		feed = model.Feed{
			Id:                   uuid.New().String(),
			Name:                 input.Name,
			Creator:              user,
			FilterDataExpression: datatypes.JSON(input.FilterDataExpression),
		}
	}

	// One caveat on gorm: if we don't specify a createdAt
	// gorm will automatically update its created time after Save is called
	// even though DB is not udpated (this is a hell of debugging)

	// Upsert DB
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// Update all columns, except primary keys and subscribers, to new value on conflict
		queryResult = r.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: false,
			DoUpdates: clause.AssignmentColumns([]string{"name", "updated_at", "creator_id", "filter_data_expression"}),
		}).Create(&feed)

		if queryResult.RowsAffected != 1 {
			return errors.New("can't upsert")
		}

		// Update subsources
		var subSources []model.SubSource
		r.DB.Where("id IN ?", input.SubSourceIds).Find(&subSources)
		if e := r.DB.Model(&feed).Association("SubSources").Replace(subSources); e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var updatedFeed model.Feed
	r.DB.Preload(clause.Associations).First(&updatedFeed, "id = ?", feed.Id)

	// If no data expression or subsources changed, skip, otherwise clear the feed's posts
	if !needClearPosts {
		// get posts
		Log.Info("update feed metadata without clear published posts")
		return &updatedFeed, nil
	}

	// Clear the feed's posts
	Log.Info("changed feed clear all posts published")
	r.DB.Where("feed_id = ?", updatedFeed.Id).Delete(&model.PostFeedPublish{})
	updatedFeed.Posts = []*model.Post{}

	return &updatedFeed, nil
}

func (r *mutationResolver) DeleteFeed(ctx context.Context, input model.DeleteFeedInput) (*model.Feed, error) {
	userId := input.UserID
	feedId := input.FeedID

	var feed model.Feed

	result := r.DB.First(&feed, "id = ?", feedId)
	if result.RowsAffected != 1 {
		return nil, errors.New("no valid feed found")
	}
	if result.Error != nil {
		return nil, result.Error
	}

	// Check ownership
	if feed.CreatorID != userId {
		return nil, errors.New("cannot delete a non-owned feed")
	}

	// Delete automatically cascade to join tables according to the schema.
	if err := r.DB.Delete(&feed).Error; err != nil {
		return nil, err
	}

	// Feed deletion updates seed state.
	ss, err := getSeedStateById(r.DB, input.UserID)
	if err != nil {
		return nil, err
	}
	fmt.Println(r.SeedStateChans)
	go func() { r.SeedStateChans.PushSeedStateToUser(ss, input.UserID) }()

	return &feed, nil
}

func (r *mutationResolver) CreatePost(ctx context.Context, input model.NewPostInput) (*model.Post, error) {
	var (
		subSource      model.SubSource
		sharedFromPost *model.Post
	)

	result := r.DB.Where("id = ?", input.SubSourceID).First(&subSource)
	if result.RowsAffected != 1 {
		return nil, errors.New("SubSource not found")
	}

	if input.SharedFromPostID != nil {
		var res model.Post
		result := r.DB.Where("id = ?", input.SharedFromPostID).First(&res)
		if result.RowsAffected != 1 {
			return nil, errors.New("SharedFromPost not found")
		}
		sharedFromPost = &res
	}

	post := model.Post{
		Id:             uuid.New().String(),
		Title:          input.Title,
		Content:        input.Content,
		CreatedAt:      time.Now(),
		SubSource:      subSource,
		SubSourceID:    input.SubSourceID,
		SharedFromPost: sharedFromPost,
		SavedByUser:    []*model.User{},
		PublishedFeeds: []*model.Feed{},
	}
	r.DB.Create(&post)

	for _, feedId := range input.FeedsIDPublishTo {
		err := r.DB.Transaction(func(tx *gorm.DB) error {
			var feed model.Feed
			result := r.DB.Where("id = ?", feedId).First(&feed)
			if result.RowsAffected != 1 {
				return errors.New("Feed not found")
			}

			if e := r.DB.Model(&post).Association("PublishedFeeds").Append(&feed); e != nil {
				return e
			}
			// return nil will commit the whole transaction
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return &post, nil
}

func (r *mutationResolver) Subscribe(ctx context.Context, input model.SubscribeInput) (*model.User, error) {
	userId := input.UserID
	feedId := input.FeedID

	var user model.User
	var feed model.Feed

	result := r.DB.First(&user, "id = ?", userId)
	if result.RowsAffected != 1 {
		return nil, errors.New(fmt.Sprintf("no valid user found %s", userId))
	}
	if result.Error != nil {
		return nil, result.Error
	}

	result = r.DB.First(&feed, "id = ?", feedId)
	if result.RowsAffected != 1 {
		return nil, errors.New(fmt.Sprintf("no valid feed found %s", feedId))
	}
	if result.Error != nil {
		return nil, result.Error
	}

	// The join table is ready after this associate, do not need to do for feed model
	// Doing that will change the UpdateTime, which is not expected and breaks when feed setting is updated
	if err := r.DB.Model(&user).Association("SubscribedFeeds").Append(&feed); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *mutationResolver) CreateSource(ctx context.Context, input model.NewSourceInput) (*model.Source, error) {
	var user model.User
	r.DB.Where("id = ?", input.UserID).First(&user)

	source := model.Source{
		Id:        uuid.New().String(),
		Name:      input.Name,
		Domain:    input.Domain,
		CreatedAt: time.Now(),
		Creator:   user,
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		r.DB.Create(&source)
		// Create default sub source, this subsource have no creator, no external id
		r.CreateSubSource(ctx, model.NewSubSourceInput{
			UserID:             user.Id,
			Name:               DefaultSubSourceName,
			ExternalIdentifier: "",
			SourceID:           source.Id,
		})
		return nil
	})

	return &source, err
}

func (r *mutationResolver) CreateSubSource(ctx context.Context, input model.NewSubSourceInput) (*model.SubSource, error) {
	uuid := uuid.New().String()

	var user model.User
	r.DB.Where("id = ?", input.UserID).First(&user)

	t := model.SubSource{
		Id:                 uuid,
		Name:               input.Name,
		ExternalIdentifier: input.ExternalIdentifier,
		CreatedAt:          time.Now(),
		SourceID:           input.SourceID,
		Creator:            user,
	}
	r.DB.Create(&t)

	return &t, nil
}

func (r *mutationResolver) SyncUp(ctx context.Context, input *model.SeedStateInput) (*model.SeedState, error) {
	if err := r.DB.Transaction(syncUpTransaction(input)); err != nil {
		return nil, err
	}

	ss, err := getSeedStateById(r.DB, input.UserSeedState.ID)
	if err != nil {
		return nil, err
	}

	// Asynchronously push to user's all other channels.
	go func() { r.SeedStateChans.PushSeedStateToUser(ss, input.UserSeedState.ID) }()

	return ss, err
}

func (r *queryResolver) AllFeeds(ctx context.Context) ([]*model.Feed, error) {
	var feeds []*model.Feed
	result := r.DB.Preload(clause.Associations).Find(&feeds)
	return feeds, result.Error
}

func (r *queryResolver) Sources(ctx context.Context) ([]*model.Source, error) {
	var sources []*model.Source
	result := r.DB.Preload(clause.Associations).Find(&sources)
	return sources, result.Error
}

func (r *queryResolver) SubSources(ctx context.Context) ([]*model.SubSource, error) {
	var subSources []*model.SubSource
	result := r.DB.Preload(clause.Associations).Find(&subSources)
	return subSources, result.Error
}

func (r *queryResolver) Posts(ctx context.Context) ([]*model.Post, error) {
	var posts []*model.Post
	result := r.DB.Preload(clause.Associations).Find(&posts)
	return posts, result.Error
}

func (r *queryResolver) Users(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	result := r.DB.Preload(clause.Associations).Find(&users)
	return users, result.Error
}

func (r *queryResolver) Feeds(ctx context.Context, input *model.FeedsGetPostsInput) ([]*model.Feed, error) {
	feedRefreshInputs := input.FeedRefreshInputs
	if len(feedRefreshInputs) == 0 {
		feeds, err := getUserSubscriptions(r, input.UserID)
		if err != nil {
			return nil, err
		}
		for _, feed := range feeds {
			feedRefreshInputs = append(feedRefreshInputs, &model.FeedRefreshInput{
				FeedID:    feed.Id,
				Limit:     feedRefreshLimit,
				Cursor:    defaultFeedsQueryCursor,
				Direction: defaultFeedsQueryDirection,
			})
		}
	}

	return getRefreshPosts(r, feedRefreshInputs)
}

func (r *subscriptionResolver) SyncDown(ctx context.Context, userID string) (<-chan *model.SeedState, error) {
	ss, err := getSeedStateById(r.DB, userID)
	if err != nil {
		return nil, err
	}

	ch, chId := r.SeedStateChans.AddNewConnection(ctx, userID)
	r.SeedStateChans.PushSeedStateToSingleChannelForUser(ss, chId, userID)

	return ch, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
