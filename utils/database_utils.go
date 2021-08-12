package utils

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Create a temp DB for testing, not that this function should only be called
// in a testing environment, and should almost always Destroy the temp DB
// immediatly after usage (via DestroyTempDB).
func CreateTempDB() (*gorm.DB, string) {
	dsn := "host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalln("cannot connect to DB")
	}

	dbName := "testonly_" + uuid.New().String()
	db = db.Exec("CREATE DATABASE " + dbName)
	if db.Error != nil {
		log.Fatalln("fail to create temp DB with name: ", dbName)
	}
	return db, dbName
}

// Drop a temp db with given name. This should always be called after
// CreateTempDB
func DropTempDB(dbName string) {
	dsn := "host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalln("cannot connect to DB")
	}
	db.Exec("DROP DATABASE " + dbName)
}
