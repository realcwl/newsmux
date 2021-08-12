package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDBCreation(t *testing.T) {
	dsn := "host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	assert.Nil(t, err)

	db = db.Exec("CREATE DATABASE test_db;")
	assert.Nil(t, db.Error)
}
