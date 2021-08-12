package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateAndDrop(t *testing.T) {
	_, dbName := CreateTempDB()

	exists, err := IsDatabaseExist(dbName)
	assert.Nil(t, err)
	assert.True(t, exists)

	DropTempDB(dbName)

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
