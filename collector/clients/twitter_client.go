package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const GetUserTimelineBaseUri = `https://api.twitter.com/1.1/statuses/user_timeline.json?user_id=%s`

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

func ParseIntoGetUserTimelineResponse(bytes []byte) (*UserTimelineResponses, error) {
	res := &UserTimelineResponses{}
	err := json.Unmarshal(bytes, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Get user posts by user id, in the form of Twitter GetUserTweets format.
func (t *TwitterClient) GetUserTweets(uid string) (*UserTimelineResponses, error) {
	req := t.constructGetUserTimelineRequest(uid)
	// Send req using http Client
	res, err := t.client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}

	fmt.Println(string(body))

	return ParseIntoGetUserTimelineResponse(body)
}

func (t *TwitterClient) constructGetUserTimelineRequest(uid string) *http.Request {
	url := fmt.Sprintf(GetUserTimelineBaseUri, uid)
	var bearer = "Bearer " + t.bearerToken

	req, _ := http.NewRequest("GET", url, nil)

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)

	return req
}
