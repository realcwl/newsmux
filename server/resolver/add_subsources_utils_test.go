package resolver

import (
	"context"
	"testing"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/stretchr/testify/assert"
)

func TestAddSubSourceImp_Twitter(t *testing.T) {
	db, _ := utils.CreateTempDB(t)
	assert.Nil(t, db.Create(&model.User{
		Id: "user_id",
	}).Error)
	assert.Nil(t, db.Create(&model.Source{
		Id:        collector.TwitterSourceId,
		Name:      "Twitter",
		CreatorID: "user_id",
	}).Error)
	subSource, err := AddSubSourceImp(db, context.Background(), model.AddSubSourceInput{
		SourceID: collector.TwitterSourceId,
		// This code assumes that Elon Musk never delete his account :P
		SubSourceUserName: "elonmusk",
	})
	assert.Nil(t, err)
	assert.Equal(t, subSource.Name, "elonmusk")
	assert.Equal(t, subSource.ExternalIdentifier, "44196397")
}
