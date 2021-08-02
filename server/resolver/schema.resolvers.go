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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *mutationResolver) CreateFeed(ctx context.Context, input model.NewFeedInput) (*model.Feed, error) {
	// TODO(Jamie): temporarily use uuid as id
	uuid := uuid.New().String()

	// TODO(Jamie): move to finalized schema and API interface
	t := model.Feed{
		Id:          "f_" + uuid,
		Title:       input.Title,
		CreatedAt:   time.Now(),
		CreatorID:   nil,
		Subscribers: []*model.User{},
	}

	r.DB.Create(&t)

	if input.CreatorID != nil {
		var user model.User
		r.DB.Where("id = ?", input.CreatorID).First(&user)
		r.DB.Model(&t).Association("Creator").Append(&user)
	}

	return &t, nil
}

func (r *mutationResolver) CreateUser(ctx context.Context, input model.NewUserInput) (*model.User, error) {
	// TODO(Jamie): temporarily use uuid as id
	uuid := uuid.New().String()

	// TODO(Jamie): move to finalized schema and API interface
	t := model.User{
		Id:              "u_" + uuid,
		Name:            input.Name,
		CreatedAt:       time.Now(),
		Age:             input.Age,
		SubscribedFeeds: []*model.Feed{},
	}
	r.DB.Create(&t)
	return &t, nil
}

func (r *mutationResolver) Subscribe(ctx context.Context, input model.SubscribeInput) (*model.User, error) {
	userId := input.UserID
	feedId := input.FeedID

	var user model.User
	var feed model.Feed

	result := r.DB.First(&user, "id = ?", userId)
	if result.RowsAffected != 1 {
		return nil, errors.New("No valid user found")
	}
	if result.Error != nil {
		return nil, result.Error
	}

	result = r.DB.First(&feed, "id = ?", feedId)
	if result.RowsAffected != 1 {
		return nil, errors.New("No valid feed found")
	}
	if result.Error != nil {
		return nil, result.Error
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Association("SubscribedFeeds").Append(&feed); err != nil {
			return err
		}
		if err := tx.Model(&feed).Association("Subscribers").Append(&user); err != nil {
			return err
		}
		// return nil will commit the whole transaction
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *queryResolver) Feeds(ctx context.Context) ([]*model.Feed, error) {
	var feeds []*model.Feed
	result := r.DB.Preload(clause.Associations).Find(&feeds)
	return feeds, result.Error
}

func (r *queryResolver) Users(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	result := r.DB.Preload(clause.Associations).Find(&users)
	return users, result.Error
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
