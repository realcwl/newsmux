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
	assert.Equal(t, subSource.Name, "Elon Musk")
	assert.Equal(t, subSource.ExternalIdentifier, "elonmusk")
}

// func TestAddSubSourceImp_Weibo(t *testing.T) {
// 	db, _ := utils.CreateTempDB(t)
// 	assert.Nil(t, db.Create(&model.User{
// 		Id: "user_id",
// 	}).Error)
// 	assert.Nil(t, db.Create(&model.Source{
// 		Id:        collector.WeiboSourceId,
// 		Name:      "Weibo",
// 		CreatorID: "user_id",
// 	}).Error)
// 	subSource, err := AddSubSourceImp(db, context.Background(), model.AddSubSourceInput{
// 		SourceID: collector.WeiboSourceId,
// 		// This code assumes that Elon Musk never delete his account :P
// 		SubSourceUserName: "庄时利和",
// 	})

// 	assert.Nil(t, err)
// 	assert.Equal(t, subSource.Name, "庄时利和")
// 	assert.Equal(t, subSource.ExternalIdentifier, "1728715190")
// }

func TestAddSubSourceImp_UnsupportedSource(t *testing.T) {
	db, _ := utils.CreateTempDB(t)
	assert.Nil(t, db.Create(&model.User{
		Id: "user_id",
	}).Error)
	assert.Nil(t, db.Create(&model.Source{
		Id:        collector.Kr36SourceId,
		Name:      "Kr36",
		CreatorID: "user_id",
	}).Error)
	_, err := AddSubSourceImp(db, context.Background(), model.AddSubSourceInput{
		SourceID: collector.Kr36SourceId,
		// This code assumes that Elon Musk never delete his account :P
		SubSourceUserName: "超级重要的新闻",
	})
	assert.NotNil(t, err)
}
