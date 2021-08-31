package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/Luismorlan/newsmux/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

	fmt.Printf("\nResponse from resolver: %+v\n", resp)

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
func TestCreateFeedAndValidate(t *testing.T, userId string, name string, filterDataExpression string, subSourceIds []string, db *gorm.DB, client *client.Client) (id string, updatedAt string) {
	var resp struct {
		UpsertFeed struct {
			Id                   string `json:"id"`
			Name                 string `json:"name"`
			CreatedAt            string `json:"createdAt"`
			UpdatedAt            string `json:"updatedAt"`
			DeletedAt            string `json:"deletedAt"`
			FilterDataExpression string `json:"filterDataExpression"`
			SubSources           []struct {
				Id string `json:"id"`
			} `json:"subSources"`
		} `json:"upsertFeed"`
	}

	subSourceIdsStr, _ := json.MarshalIndent(subSourceIds, "", "  ")
	compactedBuffer := new(bytes.Buffer)
	// json needs to be compact into one line in order to comply with graphql
	json.Compact(compactedBuffer, []byte(filterDataExpression))
	compactEscapedjson := strings.ReplaceAll(compactedBuffer.String(), `"`, `\"`)

	query := fmt.Sprintf(`mutation {
		upsertFeed(input:{userId:"%s" name:"%s" filterDataExpression:"%s" subSourceIds:%s}) {
		  id
		  name
		  createdAt
		  updatedAt
		  deletedAt
		  filterDataExpression
		  subSources {
			id
		  }
		}
	  }
	  `, userId, name, compactEscapedjson, subSourceIdsStr)

	fmt.Println(query)

	// here the escape will happen, so in resp, the FilterDataExpression is already escaped
	client.MustPost(query, &resp)

	createTime, _ := parseGQLTimeString(resp.UpsertFeed.CreatedAt)

	fmt.Println("=========================")
	fmt.Println(subSourceIds)
	fmt.Println(resp.UpsertFeed.SubSources)
	fmt.Println(resp)

	require.NotEmpty(t, resp.UpsertFeed.Id)
	require.Equal(t, name, resp.UpsertFeed.Name)
	require.Equal(t, len(subSourceIds), len(resp.UpsertFeed.SubSources))
	// for more detail see comments in TestUpdateFeedAndReturnPosts
	compactEscapedjson = strings.ReplaceAll(compactEscapedjson, `\`, ``)
	jsonEqual, err := AreJSONsEqual(compactEscapedjson, resp.UpsertFeed.FilterDataExpression)
	if err != nil {
		fmt.Println(err)
	}
	require.Truef(t, jsonEqual, "data expression invalid")

	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.UpsertFeed.DeletedAt)

	if len(subSourceIds) > 0 {
		require.Equal(t, subSourceIds[0], resp.UpsertFeed.SubSources[0].Id)
	}

	return resp.UpsertFeed.Id, resp.UpsertFeed.UpdatedAt
}

func TestUpdateFeedAndReturnPosts(t *testing.T, feed model.Feed, db *gorm.DB, client *client.Client) (postIds []string) {
	var resp struct {
		UpsertFeed struct {
			Id                   string `json:"id"`
			Name                 string `json:"name"`
			CreatedAt            string `json:"createdAt"`
			UpdatedAt            string `json:"updatedAt"`
			DeletedAt            string `json:"deletedAt"`
			FilterDataExpression string `json:"filterDataExpression"`
			SubSources           []struct {
				Id string `json:"id"`
			} `json:"subSources"`
			Subscribers []struct {
				Id string `json:"id"`
			} `json:"subscribers"`
			Posts []struct {
				Id string `json:"id"`
			} `json:"posts"`
		} `json:"upsertFeed"`
	}

	var subSourceIds []string
	for _, subsource := range feed.SubSources {
		subSourceIds = append(subSourceIds, subsource.Id)
	}
	subSourceIdsStr, _ := json.MarshalIndent(subSourceIds, "", "  ")

	compactEscapedjson := ""
	dataExpression, err := feed.FilterDataExpression.MarshalJSON()
	if err == nil && string(dataExpression) != "null" {
		compactedBuffer := new(bytes.Buffer)
		// json needs to be compact into one line in order to comply with graphql
		json.Compact(compactedBuffer, dataExpression)
		compactEscapedjson = strings.ReplaceAll(compactedBuffer.String(), `"`, `\"`)
	}

	// make the create-update gap more obvious
	time.Sleep(2 * time.Second)
	query := fmt.Sprintf(`mutation {
		upsertFeed(input:{feedId:"%s" userId:"%s" name:"%s" filterDataExpression:"%s" subSourceIds:%s}) {
		  id
		  name
		  createdAt
		  updatedAt
		  deletedAt
		  filterDataExpression
		  subSources {
			id
		  }
		  subscribers {
			  id
		  }
		  posts{
			  id
		  }
		}
	  }
	  `, feed.Id, feed.CreatorID, feed.Name, compactEscapedjson, subSourceIdsStr)

	fmt.Println(query)

	// here the escape will happen, so in resp, the FilterDataExpression is already escaped
	client.MustPost(query, &resp)

	createAt, _ := parseGQLTimeString(resp.UpsertFeed.CreatedAt)
	updatedAt, _ := parseGQLTimeString(resp.UpsertFeed.UpdatedAt)

	oldCreatedAt, _ := parseGQLTimeString(serializeGQLTime(feed.CreatedAt))
	require.Truef(t, oldCreatedAt.Equal(createAt), "created time got changed, not expected")
	require.Truef(t, feed.CreatedAt.Before(updatedAt), "updated time should after created time")
	require.Equal(t, feed.Id, resp.UpsertFeed.Id)
	require.Equal(t, feed.Name, resp.UpsertFeed.Name)

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

	var posts []string
	for _, post := range resp.UpsertFeed.Posts {
		posts = append(posts, post.Id)
	}

	return posts
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

	fmt.Printf("\nResponse from resolver: %+v\n", resp)
	createTime, _ := time.Parse("2021-08-08T14:32:50-07:00", resp.CreateSource.CreatedAt)

	require.NotEmpty(t, resp.CreateSource.Id)
	require.Equal(t, name, resp.CreateSource.Name)
	require.Equal(t, domain, resp.CreateSource.Domain)
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.CreateSource.DeletedAt)

	return resp.CreateSource.Id
}

// create subsource with name, do sanity checks and returns its Id
func TestCreateSubSourceAndValidate(t *testing.T, userId string, name string, externalIdentifier string, sourceId string, db *gorm.DB, client *client.Client) (id string) {
	var resp struct {
		CreateSubSource struct {
			Id        string `json:"id"`
			Name      string `json:"name"`
			CreatedAt string `json:"createdAt"`
			DeletedAt string `json:"deletedAt"`
		} `json:"createSubSource"`
	}

	client.MustPost(fmt.Sprintf(`mutation {
		createSubSource(input:{userId:"%s" name:"%s" externalIdentifier:"%s" sourceId:"%s"}) {
		  id
		  name
		  createdAt
		  deletedAt
		}
	  }
	  `, userId, name, externalIdentifier, sourceId), &resp)

	fmt.Printf("\nResponse from resolver: %+v\n", resp)

	createTime, _ := time.Parse("2021-08-08T14:32:50-07:00", resp.CreateSubSource.CreatedAt)

	require.NotEmpty(t, resp.CreateSubSource.Id)
	require.Equal(t, name, resp.CreateSubSource.Name)
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.CreateSubSource.DeletedAt)

	return resp.CreateSubSource.Id
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

	client.MustPost(fmt.Sprintf(`mutation {
		createPost(
			input: {
				title: "%s"
				content: "%s"
				subSourceId: "%s"
				feedsIdPublishTo: ["%s"]
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
	  `, title, content, subSourceId, publishFeedId), &resp)

	fmt.Printf("\nResponse from resolver: %+v\n", resp)

	createTime, _ := time.Parse("2021-08-08T14:32:50-07:00", resp.CreatePost.CreatedAt)

	require.NotEmpty(t, resp.CreatePost.Id)
	require.Equal(t, title, resp.CreatePost.Title)
	require.Equal(t, content, resp.CreatePost.Content)
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.CreatePost.DeletedAt)

	return resp.CreatePost.Id, resp.CreatePost.Cursor
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

	fmt.Printf("\nResponse from resolver: %+v\n", resp)

	require.Equal(t, userId, resp.Subscribe.Id)
}

// create user to feed subscription, do sanity checks
func TestFeedLinkSource(t *testing.T, sourceId string, feedId string, db *gorm.DB, client *client.Client) {

}
