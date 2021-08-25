package utils

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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
		CreateFeed struct {
			Id                   string `json:"id"`
			Name                 string `json:"name"`
			CreatedAt            string `json:"createdAt"`
			UpdatedAt            string `json:"updatedAt"`
			DeletedAt            string `json:"deletedAt"`
			FilterDataExpression string `json:"filterDataExpression"`
			SubSources           []struct {
				Id string `json:"id"`
			} `json:"subSources"`
		} `json:"createFeed"`
	}

	subSourceIdsStr, _ := json.MarshalIndent(subSourceIds, "", "  ")

	query := fmt.Sprintf(`mutation {
		createFeed(input:{userId:"%s" name:"%s" filterDataExpression:"%s" subSourceIds:%s}) {
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
	  `, userId, name, filterDataExpression, subSourceIdsStr)

	fmt.Println(query)

	// here the escape will happen, so in resp, the FilterDataExpression is already escaped
	client.MustPost(query, &resp)

	createTime, _ := parseGQLTimeString(resp.CreateFeed.CreatedAt)

	require.NotEmpty(t, resp.CreateFeed.Id)
	require.Equal(t, name, resp.CreateFeed.Name)
	require.Equal(t, len(subSourceIds), len(resp.CreateFeed.SubSources))
	// original expression after escape == received and parsed expression
	require.Equal(t, strings.ReplaceAll(filterDataExpression, `\`, ""), resp.CreateFeed.FilterDataExpression)
	require.Truef(t, time.Now().UnixNano() > createTime.UnixNano(), "time created wrong")
	require.Equal(t, "", resp.CreateFeed.DeletedAt)

	if len(subSourceIds) > 0 {
		require.Equal(t, subSourceIds[0], resp.CreateFeed.SubSources[0].Id)
	}

	return resp.CreateFeed.Id, resp.CreateFeed.UpdatedAt
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
