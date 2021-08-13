package resolver

import (
	"testing"

	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestConstructSeedStateFromUser(t *testing.T) {
	ss := constructSeedStateFromUser(&model.User{
		Id:         "user_id",
		Name:       "user_name",
		AvartarUrl: "user_avartar_url",
		SubscribedFeeds: []*model.Feed{
			{Id: "feed_id_1", Name: "feed_name_1"},
			{Id: "feed_id_2", Name: "feed_name_2"},
		},
	})

	assert.Equal(t, ss, &model.SeedState{
		UserSeedState: &model.UserSeedState{
			ID:         "user_id",
			Name:       "user_name",
			AvartarURL: "user_avartar_url",
		},
		// Order dependent comparison.
		FeedSeedState: []*model.FeedSeedState{
			{ID: "feed_id_1", Name: "feed_name_1"},
			{ID: "feed_id_2", Name: "feed_name_2"},
		},
	})
}

func TestUpdateUserSeedState(t *testing.T) {
	db, name := utils.CreateTempDB()
	defer utils.DropTempDB(db, name)

	assert.Nil(t, db.Create(&model.User{
		Id:              "id",
		Name:            "name",
		AvartarUrl:      "avartar_url",
		SubscribedFeeds: []*model.Feed{},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserSeedState(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID:        "id",
				Name:      "new_name",
				AvatarURL: "new_avartar_url",
			},
		}); err != nil {
			// return error will rollback
			return err
		}

		return nil
	})

	var user model.User
	assert.Nil(t, db.Debug().Model(&model.User{}).Select("id", "name", "avartar_url").Where("id=?", "id").First(&user).Error)
	assert.Equal(t, &model.User{
		Id:         "id",
		Name:       "new_name",
		AvartarUrl: "new_avartar_url",
	}, &user)
}

func TestUpdateUserSeedState_UserNotFound(t *testing.T) {
	db, name := utils.CreateTempDB()
	defer utils.DropTempDB(db, name)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserSeedState(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID:        "id",
				Name:      "new_name",
				AvatarURL: "new_avartar_url",
			},
		}); err != nil {
			// return error will rollback
			return err
		}

		return nil
	})
	assert.NotNil(t, err)
}

func TestUpdateFeedState(t *testing.T) {
	db, name := utils.CreateTempDB()
	defer utils.DropTempDB(db, name)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Feed{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateFeedSeedState(tx, &model.SeedStateInput{
			FeedSeedState: []*model.FeedSeedStateInput{
				{ID: "id_1", Name: "new_name_1"},
				{ID: "id_2", Name: "new_name_2"},
			},
		}); err != nil {
			// return error will rollback
			return err
		}
		return nil
	})

	var feeds []model.Feed
	db.Select("id", "name").
		Find(&feeds, []string{"id_1", "id_2"}).
		Order("id")

	assert.Equal(t, 2, len(feeds))
	assert.Equal(t, []model.Feed{
		{Id: "id_1", Name: "new_name_1"},
		{Id: "id_2", Name: "new_name_2"}},
		feeds)
}

func TestUpdateUserFeedSubscription_ChangeOrder(t *testing.T) {
	db, name := utils.CreateTempDB()
	defer utils.DropTempDB(db, name)

	assert.Nil(t, db.Create(&model.User{
		Id:              "id",
		Name:            "name",
		AvartarUrl:      "avartar_url",
		SubscribedFeeds: []*model.Feed{},
	}).Error)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Feed{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	assert.Nil(t, db.Create(&[]model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_1", OrderInPanel: 0},
		{UserID: "id", FeedID: "id_2", OrderInPanel: 1},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserFeedSubscription(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID: "id",
			},
			FeedSeedState: []*model.FeedSeedStateInput{
				{ID: "id_2", Name: "name_2"},
				{ID: "id_1", Name: "name_1"},
			},
		}); err != nil {
			// return error will rollback
			return err
		}
		return nil
	})

	var userToFeeds []model.UserFeedSubscription
	assert.Nil(t, db.Model(&model.UserFeedSubscription{}).
		Select("user_id, feed_id", "order_in_panel").
		Where("user_id = ?", "id").
		Order("order_in_panel").
		Find(&userToFeeds).Error)
	assert.Equal(t, []model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_2", OrderInPanel: 0},
		{UserID: "id", FeedID: "id_1", OrderInPanel: 1},
	}, userToFeeds)
}

func TestUpdateUserFeedSubscription_DropSubscription(t *testing.T) {
	db, name := utils.CreateTempDB()
	defer utils.DropTempDB(db, name)

	assert.Nil(t, db.Create(&model.User{
		Id:              "id",
		Name:            "name",
		AvartarUrl:      "avartar_url",
		SubscribedFeeds: []*model.Feed{},
	}).Error)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Feed{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	assert.Nil(t, db.Create(&[]model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_1", OrderInPanel: 0},
		{UserID: "id", FeedID: "id_2", OrderInPanel: 1},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserFeedSubscription(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID: "id",
			},
			// Drop subscription to id_1
			FeedSeedState: []*model.FeedSeedStateInput{
				{ID: "id_2", Name: "name_2"},
			},
		}); err != nil {
			return err
		}
		return nil
	})

	var userToFeeds []model.UserFeedSubscription
	assert.Nil(t, db.Model(&model.UserFeedSubscription{}).
		Select("user_id, feed_id", "order_in_panel").
		Where("user_id = ?", "id").
		Order("order_in_panel").
		Find(&userToFeeds).Error)
	assert.Equal(t, []model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_2", OrderInPanel: 0},
	}, userToFeeds)
}

func TestUpdateUserFeedSubscription_AddSubscription(t *testing.T) {
	db, name := utils.CreateTempDB()
	defer utils.DropTempDB(db, name)

	assert.Nil(t, db.Create(&model.User{
		Id:              "id",
		Name:            "name",
		AvartarUrl:      "avartar_url",
		SubscribedFeeds: []*model.Feed{},
	}).Error)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Feed{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	assert.Nil(t, db.Create(&[]model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_1", OrderInPanel: 0},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserFeedSubscription(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID: "id",
			},
			// add subscription to id_2
			FeedSeedState: []*model.FeedSeedStateInput{
				{ID: "id_1", Name: "name_1"},
				{ID: "id_2", Name: "name_2"},
			},
		}); err != nil {
			return err
		}
		return nil
	})

	var userToFeeds []model.UserFeedSubscription
	assert.Nil(t, db.Model(&model.UserFeedSubscription{}).
		Select("user_id, feed_id", "order_in_panel").
		Where("user_id = ?", "id").
		Order("order_in_panel").
		Find(&userToFeeds).Error)
	assert.Equal(t, []model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_1", OrderInPanel: 0},
		{UserID: "id", FeedID: "id_2", OrderInPanel: 1},
	}, userToFeeds)
}

func TestUpdateUserFeedSubscription_AddAndDropSubscription(t *testing.T) {
	db, name := utils.CreateTempDB()
	defer utils.DropTempDB(db, name)

	assert.Nil(t, db.Create(&model.User{
		Id:              "id",
		Name:            "name",
		AvartarUrl:      "avartar_url",
		SubscribedFeeds: []*model.Feed{},
	}).Error)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Feed{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	assert.Nil(t, db.Create(&[]model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_1", OrderInPanel: 0},
	}).Error)

	db.Transaction(func(tx *gorm.DB) error {
		if err := updateUserFeedSubscription(tx, &model.SeedStateInput{
			UserSeedState: &model.UserSeedStateInput{
				ID: "id",
			},
			// add subscription to id_2, drop id_1
			FeedSeedState: []*model.FeedSeedStateInput{
				{ID: "id_2", Name: "name_2"},
			},
		}); err != nil {
			return err
		}
		return nil
	})

	var userToFeeds []model.UserFeedSubscription
	assert.Nil(t, db.Model(&model.UserFeedSubscription{}).
		Select("user_id, feed_id", "order_in_panel").
		Where("user_id = ?", "id").
		Order("order_in_panel").
		Find(&userToFeeds).Error)
	assert.Equal(t, []model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_2", OrderInPanel: 0},
	}, userToFeeds)
}

func TestGetSeedStateById(t *testing.T) {
	db, name := utils.CreateTempDB()
	defer utils.DropTempDB(db, name)

	assert.Nil(t, db.Create(&model.User{
		Id:              "id",
		Name:            "name",
		AvartarUrl:      "avartar_url",
		SubscribedFeeds: []*model.Feed{},
	}).Error)

	assert.Nil(t, db.Select("id", "name").Create(&[]model.Feed{
		{
			Id:   "id_1",
			Name: "name_1",
		},
		{
			Id:   "id_2",
			Name: "name_2",
		},
	}).Error)

	assert.Nil(t, db.Create(&[]model.UserFeedSubscription{
		{UserID: "id", FeedID: "id_1", OrderInPanel: 1},
		{UserID: "id", FeedID: "id_2", OrderInPanel: 0},
	}).Error)

	ss, err := getSeedStateById(db, "id")

	assert.Nil(t, err)
	assert.Equal(t, &model.SeedState{
		UserSeedState: &model.UserSeedState{
			ID:         "id",
			Name:       "name",
			AvartarURL: "avartar_url",
		},
		FeedSeedState: []*model.FeedSeedState{
			{ID: "id_2", Name: "name_2"},
			{ID: "id_1", Name: "name_1"},
		},
	}, ss)
}
