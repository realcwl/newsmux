// database_utils should be the canonical place to put shared DB utils.
// It should not include:
// 1. Any util that doesn't manipulate DB
// 2. Any util that contains business logic
package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/Luismorlan/newsmux/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	TestDBPrefix         = "testonlydb_"
	TestDBNameCharLength = 8
)

// GormTransaction is the callback function used during db.Transaction in Gorm.
type GormTransaction func(tx *gorm.DB) error

func isTempDB(dbName string) bool {
	return strings.HasPrefix(dbName, TestDBPrefix)
}

func randomTestDBName() string {
	return TestDBPrefix + RandomAlphabetString(TestDBNameCharLength)
}

// GetDBConnection get a connection to the database specified by env
func GetDBConnection() (*gorm.DB, error) {
	return GetCustomizedConnection(os.Getenv("DB_NAME"))
}

// GetDefaultDBConnection connect to database "postgres" to manage all dbs
func GetDefaultDBConnection() (*gorm.DB, error) {
	return GetCustomizedConnection(os.Getenv("DEFAULT_DB_NAME"))
}

// GetCustomizedConnection connect to any db
func GetCustomizedConnection(dbName string) (*gorm.DB, error) {
	if dbName == os.Getenv("DEFAULT_DB_NAME") {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DEFAULT_DB_USER"), os.Getenv("DEFAULT_DB_PASS"), dbName, os.Getenv("DB_PORT"))
		return getDB(dsn)
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), dbName, os.Getenv("DB_PORT"))
	return getDB(dsn)
}

// Create a temp DB for testing, note that this function should only be called
// in a testing environment with test state manager testing.T
// It is guaranteed that this table will be dropped after each test case, user
// will not need to drop the database explicitly.
//
// Note: There are 2 cases where database won't be cleaned up:
// 1. Test fail due to timeout
// 2. Exit with signal Ctrl+C
// In both cases you should log into the database and do a manual cleanup for
// databases with prefix "testonlydb_".
func CreateTempDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	db, err := GetDefaultDBConnection()
	if err != nil {
		log.Fatalln("cannot connect to DB")
	}
	dbName := randomTestDBName()
	err = db.Exec("CREATE DATABASE " + dbName).Error
	if err != nil {
		log.Fatalln("fail to create temp DB with name: ", dbName)
	}
	newDB, err := GetCustomizedConnection(dbName)
	if err != nil {
		log.Fatalln("fail to connect to newly created DB: ", dbName)
	}
	DatabaseSetupAndMigration(newDB)
	t.Cleanup(func() {
		dropTempDB(newDB, dbName)

		// Also proactively clean up the DB connections instead of deferring to GC.
		// Otherwise, we might exceed the DB max connection limit in test and
		// causing some tests to fail.
		conn, _ := db.DB()
		conn.Close()
		conn, _ = newDB.DB()
		conn.Close()
	})

	return newDB, dbName
}

// dropTempDB drops a temp db with given name. This will always be called after
// CreateTempDB. Abort program on any failure. This function can be called
// multiple times. It won't fail on deleting non-existing DB.
func dropTempDB(curDB *gorm.DB, dbName string) {
	if !isTempDB(dbName) {
		log.Fatalln("cannot delete a non-testing DB")
	}

	exists, err := IsDatabaseExist(dbName)
	if err != nil {
		log.Fatalln("cannot connect to DB")
	}

	if !exists {
		return
	}

	// We need to close the current DB connection first. Otherwise it's not
	// possible to drop it. However we don't check if sqlDB is closed successfully
	// because fail to close will still produce error when we try to drop it.
	sqlDB, err := curDB.DB()
	if err != nil {
		log.Fatalln("cannot get the current SQL DB")
	}
	if err := sqlDB.Close(); err != nil {
		log.Println("cannot close DB", err)
	}

	db, err := GetCustomizedConnection(os.Getenv("DEFAULT_DB_NAME"))

	if err != nil {
		log.Fatalln("cannot connect to DB")
	}
	db.Exec("DROP DATABASE " + dbName)
}

func getDB(connectionString string) (db *gorm.DB, err error) {
	return gorm.Open(postgres.Open(connectionString), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func BotDBSetupAndMigration(db *gorm.DB) {
	err := db.SetupJoinTable(&model.Channel{}, "SubscribedFeeds", &model.ChannelFeedSubscription{})
	if err != nil {
		panic("failed to connect database when build many2many relationship with Channels and Feeds")
	}

	err = db.SetupJoinTable(&model.Feed{}, "SubscribedChannels", &model.ChannelFeedSubscription{})
	if err != nil {
		panic("failed to connect datebase when build many2many relationship with Feeds and Channels")
	}

	err = db.SetupJoinTable(&model.User{}, "SubscribedFeeds", &model.UserFeedSubscription{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&model.Channel{})
}

func PublisherDBSetup(db *gorm.DB) {
	err := db.SetupJoinTable(&model.Feed{}, "SubscribedChannels", &model.ChannelFeedSubscription{})
	if err != nil {
		panic("failed to connect datebase")
	}
}

func DatabaseSetupAndMigration(db *gorm.DB) {
	var err error

	err = db.SetupJoinTable(&model.User{}, "SubscribedFeeds", &model.UserFeedSubscription{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.User{}, "SavedPosts", &model.UserPostSave{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.Feed{}, "Subscribers", &model.UserFeedSubscription{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.Feed{}, "Posts", &model.PostFeedPublish{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.Post{}, "SavedByUser", &model.UserPostSave{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.SetupJoinTable(&model.Post{}, "PublishedFeeds", &model.PostFeedPublish{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&model.Feed{}, &model.User{}, &model.Post{}, &model.Source{}, &model.SubSource{})
}

// IsDatabaseExist returns true on DB exist, returns false on not exist or error
func IsDatabaseExist(dbName string) (bool, error) {
	db, err := GetDefaultDBConnection()
	if err != nil {
		return false, err
	}

	var exists bool
	res := db.Raw(fmt.Sprintf("SELECT TRUE FROM pg_catalog.pg_database WHERE lower(datname) = lower('%s') limit 1;", dbName)).Scan(&exists)
	if res.Error != nil {
		return false, err
	}

	return exists, nil
}
