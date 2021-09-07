package utils

import (
	"os"
	"testing"

	"github.com/Luismorlan/newsmux/utils/dotenv"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	dotenv.LoadDotEnvsInTests()
	os.Exit(m.Run())
}

func TestCreateAndDrop(t *testing.T) {
	db, dbName := CreateTempDB(t)

	exists, err := IsDatabaseExist(dbName)
	assert.Nil(t, err)
	assert.True(t, exists)

	dropTempDB(db, dbName)

	exists, err = IsDatabaseExist(dbName)
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestIsDatabaseExist(t *testing.T) {
	exists, err := IsDatabaseExist("postgres")
	assert.Nil(t, err)
	assert.True(t, exists)

	exists, err = IsDatabaseExist("DOES_NOT_EXIST")
	assert.Nil(t, err)
	assert.False(t, exists)
}
