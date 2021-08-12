package utils

import "testing"

func TestCreateAndDestroy(t *testing.T) {
	_, dbName := CreateTempDB()
	DropTempDB(dbName)
}
