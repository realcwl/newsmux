package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"
	"time"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *mutationResolver) CreateUser(ctx context.Context, input model.NewUserInput) (*model.User, error) {
	uuid := uuid.New().String()

	t := model.User{
		Id:              uuid,
		Name:            input.Name,
		CreatedAt:       time.Now(),
		SubscribedFeeds: []*model.Feed{},
	}
	r.DB.Create(&t)
	return &t, nil
}

func (r *mutationResolver) CreateFeed(ctx context.Context, input model.NewFeedInput) (*model.Feed, error) {
	var user model.User

	userID := input.UserID
	uuid := uuid.New().String()

	queryResult := r.DB.Where("id = ?", userID).First(&user)
	if queryResult.RowsAffected != 1 {
		return nil, errors.New("invalid user id")
	}

	feed := model.Feed{
		Id:                   uuid,
		Name:                 input.Name,
		CreatedAt:            time.Now(),
		Creator:              user,
		FilterDataExpression: datatypes.JSON(input.FilterDataExpression),
		Subscribers:          []*model.User{},
		Posts:                []*model.Post{},
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		r.DB.Create(&feed)
		for _, subSourceId := range input.SubSourceIds {
			var subSource model.SubSource
			result := r.DB.Where("id = ?", subSourceId).First(&subSource)
			if result.RowsAffected != 1 {
				return errors.New("SubSource not found")
			}

			if e := r.DB.Model(&feed).Association("SubSources").Append(&subSource); e != nil {
				return e
			}
		}
		// return nil will commit the whole transaction
		return nil
	})
	if err != nil {
		return nil, err
	}

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
		return nil, errors.New("no valid user found")
	}
	if result.Error != nil {
		return nil, result.Error
	}

	result = r.DB.First(&feed, "id = ?", feedId)
	if result.RowsAffected != 1 {
		return nil, errors.New("no valid feed found")
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

func (r *queryResolver) Feeds(ctx context.Context, input *model.FeedsForUserInput) ([]*model.Feed, error) {
	// Each feed needs to specify cursor and direction
	// Direction = TOP:    load feed new posts with cursor larger than A (default -1), from newest one, no more than LIMIT
	// Direction = BOTTOM: load feed old posts with cursor smaller than B (default -1), from newest one, no more than LIMIT
	//
	// If not specified, use TOP as direction, -1 as cursor to give newest Posts
	// How is cursor defined:
	//      it is an auto-increament index Posts
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
				Cursor:    defaultCursor,
				Direction: model.FeedRefreshDirectionNew,
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
