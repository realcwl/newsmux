package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Luismorlan/newsmux/model"
)

const (
	// develop based on this json structure, if there is any changes in json structure
	// please also change this in order to keep unit test still alive
	DataExpressionJsonForTest = `
	{
		"id":"1",
		"expr":{
			"allOf":[
				{
					"id":"1.1",
					"expr":{
					"anyOf":[
						{
							"id":"1.1.1",
							"expr":{
								"pred":{
								"type":"LITERAL",
								"param":{
									"text":"bitcoin"
								}
								}
							}
						},
						{
							"id":"1.1.2",
							"expr":{
								"pred":{
								"type":"LITERAL",
								"param":{
									"text":"以太坊"
								}
								}
							}
						}
					]
					}
				},
				{
					"id":"1.2",
					"expr":{
					"notTrue":{
						"id":"1.2.1",
						"expr":{
							"pred":{
								"type":"LITERAL",
								"param":{
								"text":"马斯克"
								}
							}
						}
					}
					}
				}
			]
		}
	}
	`

	EmptyExpressionJson = `
	{
		"id": "1"
	}
	`

	PureIdExpressionJson = `
	{
		"id":"1",
		"expr":{
			"allOf":[
				{
					"id":"1.1",
					"expr":{
						"anyOf":[
							{
								"id":"1.1.1",
								"expr":{
									"pred":{
										"type":"LITERAL",
										"param":{
											"text":"bitcoin"
										}
									}
								}
							},
							{
								"id":"1.1.2",
								"expr":{
									"pred":{
										"type":"LITERAL",
										"param":{
											"text":"以太坊"
										}
									}
								}
							},
							{
								"id": "1.1.3",
								"expr": {
									"notTrue": {
										"id": "1.1.3.1"
									}
								}
							},
							{
								"id": "1.1.4"
							}
						]
					}
				},
				{
					"id":"1.2",
					"expr":{
						"notTrue":{
							"id":"1.2.1",
							"expr":{
								"pred":{
									"type":"LITERAL",
									"param":{
										"text":"马斯克"
									}
								}
							}
						}
					}
				}, 
				{
					"id": "1.3"
				}
			]
		}
	}
	`
)

// create user with name, do sanity checks and returns its Id
func TestCreateUserAndValidate(t *testing.T, name string, userId string, db *gorm.DB, client *client.Client) (id string) {
	var resp struct {
		CreateUser struct {
			Id         string `json:"id"`
			Name       string `json:"name"`
			CreatedAt  string `json:"createdAt"`
			DeletedAt  string `json:"deletedAt"`
			SavedPosts []struct {
				Id string `json:"id"`
			}
			SubscribedFeeds []struct {
				Id string `json:"id"`
			}
		} `json:"createUser"`
	}

	client.MustPost(fmt.Sprintf(`mutation {
		createUser(input:{name:"%s" id: "%s"}) {
		  id
		  name
		  createdAt
		  deletedAt
		  savedPosts {
			  id
		  }
		  subscribedFeeds {
			  id
		  }
		}
	  }
	  `, name, userId), &resp)

	createTime, _ := time.Parse("2021-08-08T14:32:50-07:00", resp.CreateUser.CreatedAt)

	require.NotEmpty(t, resp.CreateUser.Id)
	require.Equal(t, name, resp.CreateUser.Name)
	require.Equal(t, 0, len(resp.CreateUser.SavedPosts))
	require.Equal(t, 0, len(resp.CreateUser.SubscribedFeeds))
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.CreateUser.DeletedAt)

	return resp.CreateUser.Id

}

// create feed with name, do sanity checks and returns its Id
// filterDataExpression in graphql input should have escapes \
//
func TestCreateFeedAndValidate(t *testing.T, userId string, name string, filterDataExpression string, subSourceIds []string, visibility model.Visibility, db *gorm.DB, client *client.Client) (id string, updatedAt string) {
	var resp struct {
		UpsertFeed struct {
			Id                   string           `json:"id"`
			Name                 string           `json:"name"`
			CreatedAt            string           `json:"createdAt"`
			UpdatedAt            string           `json:"updatedAt"`
			DeletedAt            string           `json:"deletedAt"`
			FilterDataExpression string           `json:"filterDataExpression"`
			Visibility           model.Visibility `json:"visibility"`
		} `json:"upsertFeed"`
	}

	subSourceIdsStr, _ := json.MarshalIndent(subSourceIds, "", "  ")
	compactedBuffer := new(bytes.Buffer)
	// json needs to be compact into one line in order to comply with graphql
	err := json.Compact(compactedBuffer, []byte(filterDataExpression))
	if err != nil {
		fmt.Println(err)
	}
	compactEscapedjson := strings.ReplaceAll(compactedBuffer.String(), `"`, `\"`)

	query := fmt.Sprintf(`mutation {
		upsertFeed(input:{userId:"%s" name:"%s" filterDataExpression:"%s" subSourceIds:%s visibility:%s}) {
		  id
		  name
		  createdAt
		  updatedAt
		  filterDataExpression
		  visibility
		}
	  }
	  `, userId, name, compactEscapedjson, subSourceIdsStr, visibility)

	// here the escape will happen, so in resp, the FilterDataExpression is already escaped
	client.MustPost(query, &resp)

	createTime, _ := parseGQLTimeString(resp.UpsertFeed.CreatedAt)

	require.NotEmpty(t, resp.UpsertFeed.Id)
	require.Equal(t, name, resp.UpsertFeed.Name)

	// resp.UpsertFeed.FilterDataExpression do not need to replace the `\`
	// reason is:
	// graphql needs to escape `"`, thus the client will always needs to escape strings
	// once gqlgen gets a input string, it will do un-escape first
	// when resolvers get a string field, it is already un-escaped
	// so in DB the string is not escaped
	// when we return from DB, gqlgen will automatically un-escape
	// making FilterDataExpression a string without escape
	compactEscapedjson = strings.ReplaceAll(compactEscapedjson, `\`, ``)
	jsonEqual, err := AreJSONsEqual(compactEscapedjson, resp.UpsertFeed.FilterDataExpression)
	if err != nil {
		fmt.Println(err)
	}
	require.Truef(t, jsonEqual, "data expression invalid")

	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.UpsertFeed.DeletedAt)
	require.Equal(t, visibility, resp.UpsertFeed.Visibility)

	var f model.Feed
	db.Preload("SubSources").First(&f, "id = ?", resp.UpsertFeed.Id)
	require.Equal(t, len(subSourceIds), len(f.SubSources))

	return resp.UpsertFeed.Id, resp.UpsertFeed.UpdatedAt
}

func TestUpdateFeed(t *testing.T, feed model.Feed, db *gorm.DB, client *client.Client) string {
	var resp struct {
		UpsertFeed struct {
			Id                   string           `json:"id"`
			Name                 string           `json:"name"`
			CreatedAt            string           `json:"createdAt"`
			UpdatedAt            string           `json:"updatedAt"`
			FilterDataExpression string           `json:"filterDataExpression"`
			Visibility           model.Visibility `json:"visibility"`
		} `json:"upsertFeed"`
	}

	var subSourceIds []string
	for _, subsource := range feed.SubSources {
		subSourceIds = append(subSourceIds, subsource.Id)
	}
	subSourceIdsStr, _ := json.MarshalIndent(subSourceIds, "", "  ")

	compactEscapedjson := "{}"
	dataExpression, err := feed.FilterDataExpression.MarshalJSON()
	if err != nil {
		fmt.Println(err)
	}
	if err == nil && string(dataExpression) != "null" && len(dataExpression) > 0 {
		compactedBuffer := new(bytes.Buffer)
		// json needs to be compact into one line in order to comply with graphql
		json.Compact(compactedBuffer, dataExpression)
		compactEscapedjson = strings.ReplaceAll(compactedBuffer.String(), `"`, `\"`)
	}

	// make the create-update gap more obvious
	time.Sleep(2 * time.Second)
	query := fmt.Sprintf(`mutation {
		upsertFeed(input:{feedId:"%s" userId:"%s" name:"%s" filterDataExpression:"%s" subSourceIds:%s visibility:%s}) {
		  id
		  name
		  createdAt
		  updatedAt
		  filterDataExpression
		  visibility
		}
	  }
	  `, feed.Id, feed.CreatorID, feed.Name, compactEscapedjson, subSourceIdsStr, feed.Visibility)

	// here the escape will happen, so in resp, the FilterDataExpression is already escaped
	client.MustPost(query, &resp)

	createTime, _ := parseGQLTimeString(resp.UpsertFeed.CreatedAt)
	updatedTime, _ := parseGQLTimeString(resp.UpsertFeed.UpdatedAt)

	oldCreatedAt, _ := parseGQLTimeString(serializeGQLTime(feed.CreatedAt))
	require.Truef(t, oldCreatedAt.Equal(createTime), "created time got changed, not expected")
	require.Truef(t, feed.CreatedAt.Before(updatedTime), "updated time should after created time")
	require.Equal(t, feed.Id, resp.UpsertFeed.Id)
	require.Equal(t, feed.Name, resp.UpsertFeed.Name)
	require.Equal(t, feed.Visibility, resp.UpsertFeed.Visibility)

	compactEscapedjson = strings.ReplaceAll(compactEscapedjson, `\`, ``)
	// resp.UpsertFeed.FilterDataExpression do not need to replace the `\`
	// reason is:
	// graphql needs to escape `"`, thus the client will always needs to escape strings
	// once gqlgen gets a input string, it will do un-escape first
	// when resolvers get a string field, it is already un-escaped
	// so in DB the string is not escaped
	// when we return from DB, gqlgen will automatically un-escape
	// making FilterDataExpression a string without escape
	jsonEqual, err := AreJSONsEqual(compactEscapedjson, resp.UpsertFeed.FilterDataExpression)
	if err != nil {
		fmt.Println(err)
	}
	require.Truef(t, jsonEqual, "data expression invalid")

	// var posts []string
	// for _, post := range resp.UpsertFeed.Posts {
	// 	posts = append(posts, post.Id)
	// }

	return resp.UpsertFeed.UpdatedAt
}

// create source with name, do sanity checks and returns its Id
func TestCreateSourceAndValidate(t *testing.T, userId string, name string, domain string, db *gorm.DB, client *client.Client) (id string) {
	var resp struct {
		CreateSource struct {
			Id        string `json:"id"`
			Name      string `json:"name"`
			Domain    string `json:"domain"`
			CreatedAt string `json:"createdAt"`
			DeletedAt string `json:"deletedAt"`
		} `json:"createSource"`
	}

	client.MustPost(fmt.Sprintf(`mutation {
		createSource(input:{userId:"%s" name:"%s" domain:"%s"}) {
		  id
		  name
		  domain
		  createdAt
		  deletedAt
		}
	  }
	  `, userId, name, domain), &resp)

	createTime, _ := time.Parse("2021-08-08T14:32:50-07:00", resp.CreateSource.CreatedAt)

	require.NotEmpty(t, resp.CreateSource.Id)
	require.Equal(t, name, resp.CreateSource.Name)
	require.Equal(t, domain, resp.CreateSource.Domain)
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.CreateSource.DeletedAt)

	return resp.CreateSource.Id
}

// create subsource with name, do sanity checks and returns its Id
func TestCreateSubSourceAndValidate(t *testing.T, userId string, name string, externalIdentifier string, sourceId string, isFromSharedPost bool, db *gorm.DB, client *client.Client) (id string) {
	var resp struct {
		UpsertSubSource struct {
			Id        string `json:"id"`
			Name      string `json:"name"`
			CreatedAt string `json:"createdAt"`
			DeletedAt string `json:"deletedAt"`
		} `json:"upsertSubSource"`
	}

	client.MustPost(fmt.Sprintf(`mutation {
		upsertSubSource(input:{name:"%s" externalIdentifier:"%s" sourceId:"%s" originUrl:"" avatarUrl:"", isFromSharedPost:%s}) {
		  id
		  name
		  createdAt
		  deletedAt
		}
	  }
	  `, name, externalIdentifier, sourceId, StringifyBoolean(isFromSharedPost)), &resp)

	createTime, _ := time.Parse("2021-08-08T14:32:50-07:00", resp.UpsertSubSource.CreatedAt)

	require.NotEmpty(t, resp.UpsertSubSource.Id)
	require.Equal(t, name, resp.UpsertSubSource.Name)
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.UpsertSubSource.DeletedAt)

	return resp.UpsertSubSource.Id
}

// create subsource with name, do sanity checks and returns its Id
func TestUpdateSubSourceAndValidate(t *testing.T, userId string, subSource *model.SubSource, db *gorm.DB, client *client.Client) (id string) {
	var resp struct {
		UpsertSubSource struct {
			Id        string `json:"id"`
			Name      string `json:"name"`
			OriginUrl string `json:"originUrl"`
			AvatarUrl string `json:"avatarUrl"`
			CreatedAt string `json:"createdAt"`
			DeletedAt string `json:"deletedAt"`
		} `json:"upsertSubSource"`
	}

	client.MustPost(fmt.Sprintf(`mutation {
		upsertSubSource(input:{name:"%s" externalIdentifier:"%s" sourceId:"%s" originUrl:"%s" avatarUrl:"%s", isFromSharedPost:false}) {
		  id
		  name
		  originUrl
		  avatarUrl
		  createdAt
		  deletedAt
		}
	  }
	  `, subSource.Name, subSource.ExternalIdentifier, subSource.SourceID, subSource.OriginUrl, subSource.AvatarUrl), &resp)

	createTime, _ := time.Parse("2021-08-08T14:32:50-07:00", resp.UpsertSubSource.CreatedAt)
	require.NotEmpty(t, resp.UpsertSubSource.Id)
	require.Equal(t, subSource.Name, resp.UpsertSubSource.Name)
	require.Equal(t, subSource.OriginUrl, resp.UpsertSubSource.OriginUrl)
	require.Equal(t, subSource.AvatarUrl, resp.UpsertSubSource.AvatarUrl)
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.UpsertSubSource.DeletedAt)

	return resp.UpsertSubSource.Id
}

// create subsource with title,content, do sanity checks and returns its Id
func TestCreatePostAndValidate(t *testing.T, title string, content string, subSourceId string, publishFeedId string, db *gorm.DB, client *client.Client) (id string, cursor int) {
	var resp struct {
		CreatePost struct {
			Id        string `json:"id"`
			Title     string `json:"title"`
			Content   string `json:"content"`
			Cursor    int    `json:"cursor"`
			CreatedAt string `json:"createdAt"`
			DeletedAt string `json:"deletedAt"`
		} `json:"createPost"`
	}

	publishFeedIds := `[]`
	if publishFeedId != "" {
		publishFeedIds = fmt.Sprintf(`["%s"]`, publishFeedId)
	}

	client.MustPost(fmt.Sprintf(`mutation {
		createPost(
			input: {
				title: "%s"
				content: "%s"
				subSourceId: "%s"
				feedsIdPublishTo: %s
			}
		) {
		  id
		  title
		  content
		  cursor
		  createdAt
		  deletedAt
		}
	  }
	  `, title, content, subSourceId, publishFeedIds), &resp)

	createTime, _ := time.Parse("2021-08-08T14:32:50-07:00", resp.CreatePost.CreatedAt)

	require.NotEmpty(t, resp.CreatePost.Id)
	require.Equal(t, title, resp.CreatePost.Title)
	require.Equal(t, content, resp.CreatePost.Content)
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.CreatePost.DeletedAt)

	return resp.CreatePost.Id, resp.CreatePost.Cursor
}

func AddPostToReplyChain(db *gorm.DB, post_id string, replyToIds []string) {
	if len(replyToIds) == 0 {
		return
	}
	post := &model.Post{
		Id: post_id,
	}
	replyThread := []model.Post{}
	for _, replyToId := range replyToIds {
		replyThread = append(replyThread, model.Post{Id: replyToId})
	}
	db.Model(post).Association("ReplyThread").Append(replyThread)
}

// create user to feed subscription, do sanity checks
func TestUserSubscribeFeedAndValidate(t *testing.T, userId string, feedId string, db *gorm.DB, client *client.Client) {
	var resp struct {
		Subscribe struct {
			Id string `json:"id"`
		} `json:"subscribe"`
	}

	client.MustPost(fmt.Sprintf(`mutation {
		subscribe(
			input: {
				userId: "%s"
				feedId: "%s"
			}
		) {
		  id
		}
	  }
	  `, userId, feedId), &resp)

	require.Equal(t, userId, resp.Subscribe.Id)
}

// create user to feed subscription, do sanity checks
func TestGetSubscriberCountAndValidate(t *testing.T, feedId string, count int, db *gorm.DB, client *client.Client) {
	var resp struct {
		AllVisibleFeeds []struct {
			Id              string `json:"id"`
			SubscriberCount int    `json:"subscriberCount"`
		} `json:"allVisibleFeeds"`
	}

	client.MustPost(`
		query {
			allVisibleFeeds {
				id
				subscriberCount
			}
		}
	`, &resp)

	for _, feed := range resp.AllVisibleFeeds {
		if feed.Id != feedId {
			continue
		}
		require.Equal(t, count, feed.SubscriberCount)
		break
	}
}

func TestDeleteFeedAndValidate(t *testing.T, userId string, feedId string, owner bool, db *gorm.DB, client *client.Client) {
	var resp struct {
		DeleteFeed struct {
			Id string `json:"id"`
		} `json:"deleteFeed"`
	}

	err := client.Post(fmt.Sprintf(`mutation {
		deleteFeed(
			input: {
				userId: "%s"
				feedId: "%s"
			}
		) {
		  id
		}
	  }
	  `, userId, feedId), &resp)

	if owner {
		require.Equal(t, feedId, resp.DeleteFeed.Id)
	} else {
		// Non owner should not delete owner's feed, but should still unsubscribe.
		require.Nil(t, err)
		sub := model.UserFeedSubscription{}
		rows := db.Model(&model.UserFeedSubscription{}).
			Where("user_id = ? AND feed_id = ?", userId, feedId).
			Find(&sub).RowsAffected
		require.Equal(t, rows, int64(0))
		feed := &model.Feed{}
		require.Equal(t, db.Model(&model.Feed{}).
			Where("id = ?", feedId).
			First(&feed).RowsAffected, int64(1))
	}
}

func TestQuerySubSources(t *testing.T, isFromSharedPost bool, isCustomized *bool, db *gorm.DB, client *client.Client) []model.SubSource {
	var resp struct {
		SubSources []model.SubSource `json:"subSources"`
	}

	customizedFilterStr := ""
	if isCustomized != nil {
		customizedFilterStr = fmt.Sprintf("isCustomized: %s", StringifyBoolean(*isCustomized))
	}
	query := fmt.Sprintf(`
	query {
		subSources(
		  input: {
			isFromSharedPost: %s
			%s
		  }
		) {
		  id
		  name
		  externalIdentifier
		  avatarUrl
		  originUrl
		  isFromSharedPost
		}
	}
	  `, StringifyBoolean(isFromSharedPost), customizedFilterStr)

	client.MustPost(query, &resp)
	return resp.SubSources
}

func TestQueryUserState(t *testing.T, userId string, feedIds []string, client *client.Client) {
	var res struct {
		UserState struct {
			User struct {
				Id              string `json:"id"`
				SubscribedFeeds []struct {
					Id string `json:"id"`
				} `json:"subscribedFeeds"`
			} `json:"user"`
		} `json:"userState"`
	}

	query := fmt.Sprintf(`
		query {
			userState(input: { userId: "%s" }) {
				user {
					id
					subscribedFeeds {
						id
					}
				}
			}
		}
	`, userId)

	client.MustPost(query, &res)

	require.Equal(t, res.UserState.User.Id, userId)
	Ids := []string{}
	for _, feed := range res.UserState.User.SubscribedFeeds {
		Ids = append(Ids, feed.Id)
	}

	require.True(t, StringSlicesContainSameElements(feedIds, Ids))
}

// delete subsource with id
func TestDeleteSubSourceAndValidate(t *testing.T, userId string, subsourceId string, db *gorm.DB, client *client.Client) {
	var resp struct {
		DeleteSubSource struct {
			Id               string `json:"id"`
			Name             string `json:"name"`
			IsFromSharedPost bool   `json:"isFromSharedPost"`
			CreatedAt        string `json:"createdAt"`
			DeletedAt        string `json:"deletedAt"`
		} `json:"deleteSubSource"`
	}

	client.MustPost(fmt.Sprintf(`mutation {
		deleteSubSource(input:{subsourceId:"%s"}) {
		  id
		  name
		  isFromSharedPost
		  createdAt
		  deletedAt
		}
	  }
	  `, subsourceId), &resp)

	fmt.Printf("\nResponse from resolver: %+v\n", resp)
	require.Equal(t, subsourceId, resp.DeleteSubSource.Id)
	require.True(t, resp.DeleteSubSource.IsFromSharedPost)
}
