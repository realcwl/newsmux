package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/clients"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/google/uuid"
	twitterscraper "github.com/n0madic/twitter-scraper"
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

	// If the sub source already exists, it could either mean that we already
	// have this sub source, or that the sub source is hidden (due to isFromSharedPost).
	// In both case we update the the sub source, so that frontend will receive
	// a response and add the sub source to its list.
	if queryResult.RowsAffected != 0 {
		if err := db.Model(subSource).Update("is_from_shared_post", false).Error; err != nil {
			return nil, fmt.Errorf("failed to update SubSource: %v", err)
		}
		return subSource, nil
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

func AddSubSourceImp(db *gorm.DB, ctx context.Context, input model.AddSubSourceInput) (*model.SubSource, error) {
	source := model.Source{}
	if db.First(&source, "id = ?", input.SourceID).RowsAffected == 0 {
		return nil, fmt.Errorf("Source not found")
	}

	expectedSubSource, err := GetSubSourceFromName(input)
	if err != nil {
		return nil, err
	}

	existingSubSource := &model.SubSource{}
	queryResult := db.
		Where("name = ? AND source_id = ?", expectedSubSource.Name, source.Id).
		First(existingSubSource)

	// If the sub source already exists, it could either mean that we already
	// have this sub source, or that the sub source is hidden (due to isFromSharedPost).
	// In both case we update the the sub source, so that frontend will receive
	// a response and add the sub source to its list. Deduplication is handled in
	// the frontend.
	if queryResult.RowsAffected != 0 {
		if err := db.Model(existingSubSource).
			Update("is_from_shared_post", false).Error; err != nil {
			return nil, fmt.Errorf("failed to update SubSource: %v", err)
		}
		return existingSubSource, nil
	}

	queryResult = db.Create(expectedSubSource)
	if queryResult.RowsAffected != 1 {
		return nil, fmt.Errorf("failed to add subsource: %+v", err)
	}
	return expectedSubSource, nil
}

func GetWeiboSubSourceFromName(input model.AddSubSourceInput) (*model.SubSource, error) {
	externalId, err := GetWeiboExternalIdFromName(input.SubSourceUserName)
	if err != nil {
		return nil, err
	}

	return &model.SubSource{
		Id:                 uuid.New().String(),
		Name:               input.SubSourceUserName,
		ExternalIdentifier: externalId,
		SourceID:           input.SourceID,
		IsFromSharedPost:   false,
	}, nil
}

func GetWeiboExternalIdFromName(name string) (string, error) {
	client := clients.NewDefaultHttpClient()
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
		return "", fmt.Errorf("response empty: %v", collector.PrettyPrint(res))
	}

	if res.Data.Cards[0].CardGroup[0].User.ScreenName == name {
		return fmt.Sprint(res.Data.Cards[0].CardGroup[0].User.ID), nil
	}
	return "", fmt.Errorf("name not found: %v", name)
}

func GetTwitterSubSourceFromName(input model.AddSubSourceInput) (*model.SubSource, error) {
	profileName, err := GetTwitterUserProfileName(input.SubSourceUserName)
	if err != nil {
		return nil, err
	}
	return &model.SubSource{
		Id:                 uuid.New().String(),
		Name:               profileName,
		ExternalIdentifier: input.SubSourceUserName,
		SourceID:           input.SourceID,
		IsFromSharedPost:   false,
	}, nil
}

func GetTwitterUserProfileName(screenName string) (string, error) {
	profile, err := twitterscraper.New().GetProfile(screenName)
	if err != nil {
		return "", err
	}
	return profile.Name, nil
}

func GetSubSourceFromName(input model.AddSubSourceInput) (*model.SubSource, error) {
	switch input.SourceID {
	case collector.WeiboSourceId:
		return GetWeiboSubSourceFromName(input)
	case collector.TwitterSourceId:
		return GetTwitterSubSourceFromName(input)
	}
	return nil, errors.New("unsupported source")
}
