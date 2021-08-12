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
	defer utils.DropTempDB(name)

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
