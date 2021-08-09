package resolver

import (
	"context"
	"fmt"

	"github.com/Luismorlan/newsmux/model"
)

// Update user subscribed feeds given the user provided feed inputs.
func updateSubscribedFeeds(feedSeedStateInputs []*model.FeedSeedStateInput, feeds []*model.Feed) []*model.Feed {
	// filter all user feed seed state
	filteredSeedStates := []*model.Feed{}
	allIds := map[string]bool{}
	for _, feedSeedStateInput := range feedSeedStateInputs {
		allIds[feedSeedStateInput.ID] = true
	}

	for _, ss := range feeds {
		if allIds[ss.Id] {
			filteredSeedStates = append(filteredSeedStates, ss)
		}
	}

	return filteredSeedStates
}

func (r *mutationResolver) SyncUp(ctx context.Context, input *model.SeedStateInput) (*model.SeedState, error) {
	userId := input.UserSeedState.ID
	var user model.User
	res := r.DB.Model(&model.User{}).Where("id=?", userId).Preload("SubscribedFeeds").First(&user)
	if res.RowsAffected != 1 {
		fmt.Println("user not found")
		//return nil, errors.New("user not found")
	}

	// Populate user related seed states
	user.AvartarUrl = input.UserSeedState.AvatarURL
	user.Name = input.UserSeedState.Name

	// Populate feed related seed states
	user.SubscribedFeeds = updateSubscribedFeeds(input.FeedSeedState, user.SubscribedFeeds)
	r.DB.Save(&user)

	return constructSeedStateFromUser(&user), nil
}

func (r *subscriptionResolver) SyncDown(ctx context.Context, userID string) (<-chan *model.SeedState, error) {
	ch := r.SeedStateChans.AddNewConnection(ctx, userID)
	r.SeedStateChans.PushSeedStateToUser(&model.SeedState{
		UserSeedState: &model.UserSeedState{
			Name: "Dummy name",
		},
	}, userID)

	return ch, nil
}
