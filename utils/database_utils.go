// database_utils should be the canonical place to put shared DB utils.
// It should not include:
// 1. Any util that doesn't manipulate DB
// 2. Any util that contains business logic
package utils

import (
	"fmt"
	"log"
	"strings"

	"github.com/Luismorlan/newsmux/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

// getDefaultDBConnection returns a connection to the default database postgres.
func getDefaultDBConnection() (*gorm.DB, error) {
	return getCustomizedConnection("postgres")
}

// getCustomizedConnection connect to customized database
func getCustomizedConnection(dbName string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=%s port=5432 sslmode=disable", dbName)
	return getDB(dsn)
}

// Create a temp DB for testing, not that this function should only be called
// in a testing environment, and should almost always Destroy the temp DB
// immediatly after usage (via DropTempDB). Abort program on any failure.
// e.g. Unless you know what you're doing, in all cases you should write:
//
// 		db, name := utils.CreateTempDB()
// 		defer utils.DropTempDB(name)
//
// to make sure the DB is cleaned up.
func CreateTempDB() (*gorm.DB, string) {
	db, err := getDefaultDBConnection()

	if err != nil {
		log.Fatalln("cannot connect to DB")
	}

	dbName := randomTestDBName()

	err = db.Exec("CREATE DATABASE " + dbName).Error
	if err != nil {
		log.Fatalln("fail to create temp DB with name: ", dbName)
	}

	newDB, err := getCustomizedConnection(dbName)
	if err != nil {
		log.Fatalln("fail to connect to newly created DB: ", dbName)
	}
	DatabaseSetupAndMigration(newDB)
	return newDB, dbName
}

// DropTempDB drops a temp db with given name. This should always be called after
// CreateTempDB. Abort program on any failure. This function can be called
// multiple times. It won't fail on deleting non-existing DB.
func DropTempDB(curDB *gorm.DB, dbName string) {
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
	sqlDB.Close()

	db, err := getDefaultDBConnection()

	if err != nil {
		log.Fatalln("cannot connect to DB")
	}
	db.Exec("DROP DATABASE " + dbName)
}

// Get DB instance for development
func GetDBDev() (db *gorm.DB, err error) {
	// TODO(jamie): move to .env
	return getCustomizedConnection("dev_jamie")
}

// Get DB instance for production
func GetDBProduction() (db *gorm.DB, err error) {
	// TODO(jamie): move to .env
	return getCustomizedConnection("dev_jamie")
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

	db.AutoMigrate(&model.Feed{}, &model.User{}, &model.Post{}, &model.Source{}, &model.SubSource{})
}

// IsDatabaseExist returns true on DB exist, returns false on not exist or error
func IsDatabaseExist(dbName string) (bool, error) {
	db, err := getDefaultDBConnection()
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
