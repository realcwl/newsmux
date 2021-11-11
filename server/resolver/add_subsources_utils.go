package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WeiboSearchApiResponse struct {
	Ok   int `json:"ok"`
	Data struct {
		Cards []struct {
			CardGroup []struct {
				User struct {
					ID              int64  `json:"id"`
					ScreenName      string `json:"screen_name"`
					ProfileImageURL string `json:"profile_image_url"`
					ProfileURL      string `json:"profile_url"`
					Description     string `json:"description"`
					AvatarHd        string `json:"avatar_hd"`
				} `json:"user"`
			} `json:"card_group"`
			ShowType int `json:"show_type"`
		} `json:"cards"`
	} `json:"data"`
}

// constructSeedStateFromUser constructs SeedState with model.User with
// pre-populated SubscribedFeeds.
func AddWeiboSubsourceImp(db *gorm.DB, ctx context.Context, input model.AddWeiboSubSourceInput) (subSource *model.SubSource, err error) {
	var weiboSource model.Source
	queryWeiboSourceIdResult := db.
		Where("name = ?", "微博").
		First(&weiboSource)
	if queryWeiboSourceIdResult.RowsAffected == 0 {
		return nil, fmt.Errorf("weibo source not exist")
	}

	weiboSourceId := weiboSource.Id
	queryResult := db.
		Where("name = ? AND source_id = ?", input.Name, weiboSourceId).
		First(&subSource)

	if queryResult.RowsAffected != 0 {
		return nil, fmt.Errorf("subsource already exists: %+v", subSource)
	}

	externalId, err := GetWeiboExternalIdFromName(input.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get external id: %v", err)
	}

	// Create new SubSource
	subSource = &model.SubSource{
		Id:                 uuid.New().String(),
		Name:               input.Name,
		ExternalIdentifier: externalId,
		SourceID:           weiboSourceId,
		IsFromSharedPost:   false,
	}
	queryResult = db.Create(subSource)
	if queryResult.RowsAffected != 1 {
		return nil, fmt.Errorf("failed to add subsource: %+v", subSource)
	}
	return subSource, nil
}

func GetWeiboExternalIdFromName(name string) (string, error) {
	var client collector.HttpClient
	// weibo search API is weird in a way that it has type and q params encoded as url but other params not
	url := "https://m.weibo.cn/api/container/getIndex?containerid=100103type%3D1%26q%3D" + name + "&page_type=searchall"
	resp, err := client.Get(url)
	if err != nil {
		return "", utils.ImmediatePrintError(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", utils.ImmediatePrintError(err)
	}
	res := &WeiboSearchApiResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return "", utils.ImmediatePrintError(err)
	}
	if res.Ok != 1 {
		return "", fmt.Errorf("response not success: %v", res)
	}
	if len(res.Data.Cards) == 0 || len(res.Data.Cards[0].CardGroup) == 0 {
		return "", fmt.Errorf("response empty: %v", res)
	}

	if res.Data.Cards[0].CardGroup[0].User.ScreenName == name {
		return fmt.Sprint(res.Data.Cards[0].CardGroup[0].User.ID), nil
	}
	return "", fmt.Errorf("name not found: %v", name)
}