package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const tweeterRequestBaseUri = `https://api.twitter.com/2/users/%s/tweets`

// See postman: https://web.postman.co/workspace/Twitter-API-Test~71d3eb28-55ff-4d43-8972-8f2bef7109a2/request/18412083-4f0f66df-11f8-4672-bb8b-7d3e6982b649
var getUserTweetsQueryParams = map[string]string{
	"expansions":   "attachments.media_keys,referenced_tweets.id",
	"media.fields": "preview_image_url,url",
}

type TwitterClient struct {
	// HttpClient that is used to actually make request
	client *http.Client

	// Bearer token used to actually make Twitter request
	bearerToken string
}

func NewTwitterClient(client *http.Client, bearerToken string) *TwitterClient {
	return &TwitterClient{
		client:      client,
		bearerToken: bearerToken,
	}
}

func ParseIntoGetUserTweetsResponse(bytes []byte) (*GetUserTweetsResponse, error) {
	res := &GetUserTweetsResponse{}
	err := json.Unmarshal(bytes, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Get user posts by user id, in the form of Twitter GetUserTweets format.
func (t *TwitterClient) GetUserTweets(uid string) (*GetUserTweetsResponse, error) {
	req := t.constructGetUserTweetsRequest(uid)
	// Send req using http Client
	res, err := t.client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}

	return ParseIntoGetUserTweetsResponse(body)
}

func (t *TwitterClient) constructGetUserTweetsRequest(uid string) *http.Request {
	url := fmt.Sprintf(tweeterRequestBaseUri, uid)
	var bearer = "Bearer " + t.bearerToken

	req, _ := http.NewRequest("GET", url, nil)

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)

	for k, v := range getUserTweetsQueryParams {
		req.URL.Query().Add(k, v)
	}

	return req
}
