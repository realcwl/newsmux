package clients

// Defines the tweeter type of the tweet.
type TweetType int

// See here for the difference: https://lucid.app/lucidchart/a2666fe4-d9a4-48eb-9f7d-67358427f0d4/edit?invitationId=inv_fad004c2-75b2-4432-85bb-a236c677bcef
const (
	TWEET_TYPE_UNKNOWN = 0
	// When user just created a single tweet.
	TWEET_TYPE_SINGLE = 1
	// When user quote other's tweet (single tweet), and added some comment.
	TWEET_TYPE_QUOTE = 2
	// When user reply to self, creating a chain of twitter telling a long story.
	TWEET_TYPE_THREAD = 3
	// The lightweight way of sharing a tweet to you own homepage.
	TWEET_TYPE_RETWEET = 4
)

const (
	REFERENCE_TYPE_QUOTED = "quoted"

	REFERENCE_TYPE_RETWEETED = "retweeted"

	REFERENCE_TYPE_REPLIED_TO = "replied_to"
)

// Response used for getting tweeter response. See https://web.postman.co/workspace/Twitter-API-Test~71d3eb28-55ff-4d43-8972-8f2bef7109a2/request/18412083-4f0f66df-11f8-4672-bb8b-7d3e6982b649
type GetUserTweetsResponse struct {
	Data     []GetUserTweetsResponseData `json:"data"`
	Includes struct {
		Media []struct {
			MediaKey string `json:"media_key"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"media"`
		Tweets []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"tweets"`
	} `json:"includes"`
	Meta struct {
		OldestID    string `json:"oldest_id"`
		NewestID    string `json:"newest_id"`
		ResultCount int    `json:"result_count"`
		NextToken   string `json:"next_token"`
	} `json:"meta"`
}

type GetUserTweetsResponseData struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Attachments struct {
		MediaKeys []string `json:"media_keys"`
	} `json:"attachments,omitempty"`
	ReferencedTweets []struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"referenced_tweets,omitempty"`
}
