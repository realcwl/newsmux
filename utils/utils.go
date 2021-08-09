package utils

import (
	"github.com/Luismorlan/newsmux/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ContainsString returns true iff the provided string slice hay contains string
// needle.
func ContainsString(hay []string, needle string) bool {
	for _, str := range hay {
		if str == needle {
			return true
		}
	}
	return false
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Get DB instance for development
func GetDBDev() (db *gorm.DB, err error) {
	// TODO(jamie): move to .env
	dsn := "host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=dev_jamie port=5432 sslmode=disable"
	return getDB(dsn)
}

// Get DB instance for unit test
func GetDBLocalTest() (db *gorm.DB, err error) {
	// TODO(jamie): move to .env and think about how to easily clean up unit test db
	dsn := "host=localhost user=postgres password=postgres dbname=unit_test_db port=5432 sslmode=disable"
	return getDB(dsn)
}

// Get DB instance for production
func GetDBProduction() (db *gorm.DB, err error) {
	// TODO(jamie): move to .env
	dsn := "host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=dev_jamie port=5432 sslmode=disable"
	return getDB(dsn)
}

func getDB(connectionString string) (db *gorm.DB, err error) {
	return gorm.Open(postgres.Open(connectionString), &gorm.Config{})
}

func DatabaseSetupAndMigration(db *gorm.DB) {
	var err error

	err = db.SetupJoinTable(&model.User{}, "SubscribedFeeds", &model.UserFeedSubscription{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.Feed{}, "Subscribers", &model.UserFeedSubscription{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.Post{}, "SavedByUser", &model.UserPostSave{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.User{}, "SavedPosts", &model.UserPostSave{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.Post{}, "PublishedFeeds", &model.PostFeedPublish{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.Feed{}, "Posts", &model.PostFeedPublish{})
	if err != nil {
		panic("failed to connect database")
	}

	db.Debug().AutoMigrate(&model.Feed{}, &model.User{}, &model.Post{}, &model.Source{}, &model.SubSource{})
}
