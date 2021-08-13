package resolver

import (
	"errors"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
	"gorm.io/gorm"
)

// constructSeedStateFromUser constructs SeedState with model.User with
// pre-populated SubscribedFeeds.
func constructSeedStateFromUser(user *model.User) *model.SeedState {
	res := &model.SeedState{
		UserSeedState: &model.UserSeedState{
			ID:        user.Id,
			Name:      user.Name,
			AvatarURL: user.AvatarUrl,
		},
		FeedSeedState: feedToSeedState(user.SubscribedFeeds),
	}

	return res
}

// feedToSeedState converts from Feed to FeedSeedState.
func feedToSeedState(feeds []*model.Feed) []*model.FeedSeedState {
	res := []*model.FeedSeedState{}

	for _, feed := range feeds {
		res = append(res, &model.FeedSeedState{
			ID:   feed.Id,
			Name: feed.Name,
		})
	}

	return res
}

func updateUserSeedState(tx *gorm.DB, input *model.SeedStateInput) error {
	var user model.User
	res := tx.Model(&model.User{}).Where("id=?", input.UserSeedState.ID).First(&user)
	if res.RowsAffected != 1 {
		return errors.New("user not found")
	}

	user.AvatarUrl = input.UserSeedState.AvatarURL
	user.Name = input.UserSeedState.Name

	if err := tx.Save(&user).Error; err != nil {
		return err
	}

	return nil
}

func updateFeedSeedState(tx *gorm.DB, input *model.SeedStateInput) error {
	for _, feedSeedStateInput := range input.FeedSeedState {
		// Handler error in a soft way. If the feed doesn't exist, continue.
		var tmp model.Feed
		res := tx.Model(&model.Feed{}).Where("id = ?", feedSeedStateInput.ID).First(&tmp)
		if res.RowsAffected != 1 {
			continue
		}
		res = tx.Model(&model.Feed{}).Where("id = ?", feedSeedStateInput.ID).
			Updates(model.Feed{Name: feedSeedStateInput.Name})
		if res.Error != nil {
			// Return error will rollback
			return res.Error
		}
	}

	return nil
}

// updateUserFeedSubscription will do 2 things:
// 1. remove/add unnecessary user feed subscription.
// 2. reorder Feed subscriptions.
func updateUserFeedSubscription(tx *gorm.DB, input *model.SeedStateInput) error {
	feedIdToPos := make(map[string]int)
	for idx, feedSeedStateInput := range input.FeedSeedState {
		feedIdToPos[feedSeedStateInput.ID] = idx
	}

	var userToFeeds []model.UserFeedSubscription
	if err := tx.Model(&model.UserFeedSubscription{}).
		Where("user_id = ?", input.UserSeedState.ID).
		Find(&userToFeeds).Error; err != nil {
		return err
	}

	for _, userToFeed := range userToFeeds {
		pos, ok := feedIdToPos[userToFeed.FeedID]
		if !ok {
			continue
		}

		// Otherwise we should just update the position. We use map to update the
		// field order_in_panel due to zero-like value will be ignored during
		// structural update. See https://gorm.io/docs/update.html for details.
		if err := tx.Model(&model.UserFeedSubscription{}).
			Where("user_id = ? AND feed_id = ?", userToFeed.UserID, userToFeed.FeedID).
			Updates(map[string]interface{}{
				"order_in_panel": pos,
			}).Error; err != nil {
			return err
		}
	}

	// return nil will commit the whole transaction
	return nil
}

// create a syncUp transaction callback that performs the core business logic
func syncUpTransaction(input *model.SeedStateInput) utils.GormTransaction {
	return func(tx *gorm.DB) error {
		if err := updateUserSeedState(tx, input); err != nil {
			// return error will rollback
			return err
		}

		if err := updateFeedSeedState(tx, input); err != nil {
			return err
		}

		if err := updateUserFeedSubscription(tx, input); err != nil {
			return err
		}

		// return nil will commit the whole transaction
		return nil
	}
}

// getting the latest SeedState from the DB
func getSeedStateById(db *gorm.DB, userId string) (*model.SeedState, error) {
	var user model.User
	res := db.Model(&model.User{}).Where("id=?", userId).First(&user)
	if res.RowsAffected != 1 {
		return nil, errors.New("user not found or duplicate user")
	}

	var feeds []model.Feed
	db.Model(&model.UserFeedSubscription{}).
		Select("feeds.id", "feeds.name").
		Joins("INNER JOIN feeds ON feeds.id = user_feed_subscriptions.feed_id").
		Order("order_in_panel").
		Find(&feeds)

	for idx := range feeds {
		user.SubscribedFeeds = append(user.SubscribedFeeds, &feeds[idx])
	}

	ss := constructSeedStateFromUser(&user)

	return ss, nil
}
