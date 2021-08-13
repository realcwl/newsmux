package resolver

import (
	"errors"
	"sort"

	"github.com/Luismorlan/newsmux/model"
	"gorm.io/gorm"
)

// GormTransaction is the callback function used during db.Transaction in Gorm.
type GormTransaction func(tx *gorm.DB) error

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
			// This mean user has deleted this subscription, we should also remove the
			// dependency in userFeedSubscription table.
			if err := tx.Delete(&userToFeed).Error; err != nil {
				return err
			}
			continue
		}

		// Otherwise we should just update the position. We use map to update the
		// field order_in_panel due to zero-like value will be ignored during
		// structural update. See https://gorm.io/docs/update.html for details.
		if err := tx.Debug().Model(&model.UserFeedSubscription{}).
			Where("user_id = ? AND feed_id = ?", userToFeed.UserID, userToFeed.FeedID).
			Updates(map[string]interface{}{
				"order_in_panel": pos,
			}).Error; err != nil {
			return err
		}
		delete(feedIdToPos, userToFeed.FeedID)
	}

	// If there are more to be processed, add them into subscription table.
	for feedId, pos := range feedIdToPos {
		if err := tx.Model(&model.UserFeedSubscription{}).
			Create(&model.UserFeedSubscription{
				UserID:       input.UserSeedState.ID,
				FeedID:       feedId,
				OrderInPanel: pos,
			}).Error; err != nil {
			return err
		}
	}

	// return nil will commit the whole transaction
	return nil
}

// create a syncUp transaction callback that performs the core business logic
func syncUpTransaction(input *model.SeedStateInput) GormTransaction {
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
	res := db.Model(&model.User{}).Where("id=?", userId).
		Preload("SubscribedFeeds").First(&user)
	if res.RowsAffected != 1 {
		return nil, errors.New("user not found")
	}

	// sort seed state by corresponding order.
	feedIdToPos := make(map[string]int)
	var userToFeeds []model.UserFeedSubscription
	if err := db.Model(&model.UserFeedSubscription{}).
		Where("user_id = ?", userId).
		Find(&userToFeeds).Error; err != nil {
		return nil, err
	}

	for _, userToFeed := range userToFeeds {
		feedIdToPos[userToFeed.FeedID] = userToFeed.OrderInPanel
	}

	ss := constructSeedStateFromUser(&user)
	sort.SliceStable(ss.FeedSeedState, func(i, j int) bool {
		return feedIdToPos[ss.FeedSeedState[i].ID] < feedIdToPos[ss.FeedSeedState[j].ID]
	})

	return ss, nil
}
