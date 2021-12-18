package collector_instances

import (
	"encoding/json"
	"testing"

	"github.com/Luismorlan/newsmux/protocol"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/stretchr/testify/assert"
)

const replyChain = `
{
	"Hashtags": null,
	"HTML": "3",
	"ID": "1467216303161180162",
	"InReplyToStatus": {
					"Hashtags": null,
					"HTML": "2",
					"ID": "1467216274493034499",
					"InReplyToStatus": {
									"Hashtags": null,
									"HTML": "1",
									"ID": "1467216248891011074",
									"InReplyToStatus": null,
									"IsQuoted": false,
									"IsPin": false,
									"IsReply": false,
									"IsRetweet": false,
									"Likes": 0,
									"PermanentURL": "https://twitter.com/RnrCapital/status/1467216248891011074",
									"Photos": null,
									"Place": null,
									"QuotedStatus": null,
									"Replies": 1,
									"Retweets": 0,
									"RetweetedStatus": null,
									"Text": "1",
									"TimeParsed": "2021-12-04T19:36:27Z",
									"Timestamp": 1638646587,
									"URLs": null,
									"UserID": "1460342597218553856",
									"Username": "RnrCapital",
									"Videos": null
					},
					"IsQuoted": false,
					"IsPin": false,
					"IsReply": true,
					"IsRetweet": false,
					"Likes": 0,
					"PermanentURL": "https://twitter.com/RnrCapital/status/1467216274493034499",
					"Photos": null,
					"Place": null,
					"QuotedStatus": null,
					"Replies": 3,
					"Retweets": 0,
					"RetweetedStatus": null,
					"Text": "2",
					"TimeParsed": "2021-12-04T19:36:34Z",
					"Timestamp": 1638646594,
					"URLs": null,
					"UserID": "1460342597218553856",
					"Username": "RnrCapital",
					"Videos": null
	},
	"IsQuoted": false,
	"IsPin": false,
	"IsReply": true,
	"IsRetweet": false,
	"Likes": 0,
	"PermanentURL": "https://twitter.com/RnrCapital/status/1467216303161180162",
	"Photos": null,
	"Place": null,
	"QuotedStatus": null,
	"Replies": 0,
	"Retweets": 0,
	"RetweetedStatus": null,
	"Text": "3",
	"TimeParsed": "2021-12-04T19:36:40Z",
	"Timestamp": 1638646600,
	"URLs": null,
	"UserID": "1460342597218553856",
	"Username": "RnrCapital",
	"Videos": null
} 
`

const replyWithAndQuote = `
{
	"Hashtags": null,
	"HTML": "test\u003cbr\u003e\u003ca href=\"https://twitter.com/RnrCapital/status/1467210334679175170\"\u003ehttps://t.co/LBlxV4Zheb\u003c/a\u003e",
	"ID": "1467217405193568260",
	"InReplyToStatus": {
					"Hashtags": null,
					"HTML": "2",
					"ID": "1467216274493034499",
					"InReplyToStatus": {
									"Hashtags": null,
									"HTML": "1",
									"ID": "1467216248891011074",
									"InReplyToStatus": null,
									"IsQuoted": false,
									"IsPin": false,
									"IsReply": false,
									"IsRetweet": false,
									"Likes": 0,
									"PermanentURL": "https://twitter.com/RnrCapital/status/1467216248891011074",
									"Photos": null,
									"Place": null,
									"QuotedStatus": null,
									"Replies": 1,
									"Retweets": 0,
									"RetweetedStatus": null,
									"Text": "1",
									"TimeParsed": "2021-12-04T19:36:27Z",
									"Timestamp": 1638646587,
									"URLs": null,
									"UserID": "1460342597218553856",
									"Username": "RnrCapital",
									"Videos": null
					},
					"IsQuoted": false,
					"IsPin": false,
					"IsReply": true,
					"IsRetweet": false,
					"Likes": 0,
					"PermanentURL": "https://twitter.com/RnrCapital/status/1467216274493034499",
					"Photos": null,
					"Place": null,
					"QuotedStatus": null,
					"Replies": 3,
					"Retweets": 0,
					"RetweetedStatus": null,
					"Text": "2",
					"TimeParsed": "2021-12-04T19:36:34Z",
					"Timestamp": 1638646594,
					"URLs": null,
					"UserID": "1460342597218553856",
					"Username": "RnrCapital",
					"Videos": null
	},
	"IsQuoted": true,
	"IsPin": false,
	"IsReply": true,
	"IsRetweet": false,
	"Likes": 0,
	"PermanentURL": "https://twitter.com/RnrCapital/status/1467217405193568260",
	"Photos": null,
	"Place": null,
	"QuotedStatus": {
					"Hashtags": null,
					"HTML": "quote_1",
					"ID": "1467210334679175170",
					"InReplyToStatus": null,
					"IsQuoted": true,
					"IsPin": false,
					"IsReply": false,
					"IsRetweet": false,
					"Likes": 0,
					"PermanentURL": "https://twitter.com/RnrCapital/status/1467210334679175170",
					"Photos": null,
					"Place": null,
					"QuotedStatus": {
									"Hashtags": null,
									"HTML": "ngmi\u003cbr\u003egmi\u003cbr\u003egm\u003cbr\u003em\u003cbr\u003e⚪️",
									"ID": "1467204501899784195",
									"InReplyToStatus": null,
									"IsQuoted": true,
									"IsPin": false,
									"IsReply": false,
									"IsRetweet": false,
									"Likes": 151,
									"PermanentURL": "https://twitter.com/RogerDickerman/status/1467204501899784195",
									"Photos": null,
									"Place": null,
									"QuotedStatus": null,
									"Replies": 4,
									"Retweets": 15,
									"RetweetedStatus": null,
									"Text": "ngmi\ngmi\ngm\nm\n⚪️",
									"TimeParsed": "2021-12-04T18:49:47Z",
									"Timestamp": 1638643787,
									"URLs": null,
									"UserID": "214226213",
									"Username": "RogerDickerman",
									"Videos": null
					},
					"Replies": 0,
					"Retweets": 0,
					"RetweetedStatus": null,
					"Text": "quote_1",
					"TimeParsed": "2021-12-04T19:12:57Z",
					"Timestamp": 1638645177,
					"URLs": null,
					"UserID": "1460342597218553856",
					"Username": "RnrCapital",
					"Videos": null
	},
	"Replies": 0,
	"Retweets": 0,
	"RetweetedStatus": null,
	"Text": "test\nhttps://t.co/LBlxV4Zheb",
	"TimeParsed": "2021-12-04T19:41:03Z",
	"Timestamp": 1638646863,
	"URLs": [
					"https://twitter.com/RnrCapital/status/1467210334679175170"
	],
	"UserID": "1460342597218553856",
	"Username": "RnrCapital",
	"Videos": null
}
`

const completeThread = `
{
	"Hashtags": null,
	"HTML": "test\u003cbr\u003e\u003ca href=\"https://twitter.com/RnrCapital/status/1467210334679175170\"\u003ehttps://t.co/LBlxV4Zheb\u003c/a\u003e",
	"ID": "1467217405193568260",
	"InReplyToStatus": {
					"Hashtags": null,
					"HTML": "2",
					"ID": "1467216274493034499",
					"InReplyToStatus": {
									"Hashtags": null,
									"HTML": "1",
									"ID": "1467216248891011074",
									"InReplyToStatus": null,
									"IsQuoted": false,
									"IsPin": false,
									"IsReply": false,
									"IsRetweet": false,
									"Likes": 0,
									"PermanentURL": "https://twitter.com/RnrCapital/status/1467216248891011074",
									"Photos": null,
									"Place": null,
									"QuotedStatus": null,
									"Replies": 1,
									"Retweets": 0,
									"RetweetedStatus": null,
									"Text": "1",
									"TimeParsed": "2021-12-04T19:36:27Z",
									"Timestamp": 1638646587,
									"URLs": null,
									"UserID": "1460342597218553856",
									"Username": "RnrCapital",
									"Videos": null
					},
					"IsQuoted": false,
					"IsPin": false,
					"IsReply": true,
					"IsRetweet": false,
					"Likes": 0,
					"PermanentURL": "https://twitter.com/RnrCapital/status/1467216274493034499",
					"Photos": null,
					"Place": null,
					"QuotedStatus": null,
					"Replies": 3,
					"Retweets": 0,
					"RetweetedStatus": null,
					"Text": "2",
					"TimeParsed": "2021-12-04T19:36:34Z",
					"Timestamp": 1638646594,
					"URLs": null,
					"UserID": "1460342597218553856",
					"Username": "RnrCapital",
					"Videos": null
	},
	"IsQuoted": true,
	"IsPin": false,
	"IsReply": true,
	"IsRetweet": false,
	"Likes": 0,
	"PermanentURL": "https://twitter.com/RnrCapital/status/1467217405193568260",
	"Photos": null,
	"Place": null,
	"QuotedStatus": {
					"Hashtags": null,
					"HTML": "quote_1",
					"ID": "1467210334679175170",
					"InReplyToStatus": null,
					"IsQuoted": true,
					"IsPin": false,
					"IsReply": false,
					"IsRetweet": false,
					"Likes": 0,
					"PermanentURL": "https://twitter.com/RnrCapital/status/1467210334679175170",
					"Photos": null,
					"Place": null,
					"QuotedStatus": {
									"Hashtags": null,
									"HTML": "ngmi\u003cbr\u003egmi\u003cbr\u003egm\u003cbr\u003em\u003cbr\u003e⚪️",
									"ID": "1467204501899784195",
									"InReplyToStatus": null,
									"IsQuoted": true,
									"IsPin": false,
									"IsReply": false,
									"IsRetweet": false,
									"Likes": 151,
									"PermanentURL": "https://twitter.com/RogerDickerman/status/1467204501899784195",
									"Photos": null,
									"Place": null,
									"QuotedStatus": null,
									"Replies": 4,
									"Retweets": 15,
									"RetweetedStatus": null,
									"Text": "ngmi\ngmi\ngm\nm\n⚪️",
									"TimeParsed": "2021-12-04T18:49:47Z",
									"Timestamp": 1638643787,
									"URLs": null,
									"UserID": "214226213",
									"Username": "RogerDickerman",
									"Videos": null
					},
					"Replies": 0,
					"Retweets": 0,
					"RetweetedStatus": null,
					"Text": "quote_1",
					"TimeParsed": "2021-12-04T19:12:57Z",
					"Timestamp": 1638645177,
					"URLs": null,
					"UserID": "1460342597218553856",
					"Username": "RnrCapital",
					"Videos": null
	},
	"Replies": 0,
	"Retweets": 0,
	"RetweetedStatus": null,
	"Text": "test\nhttps://t.co/LBlxV4Zheb",
	"TimeParsed": "2021-12-04T19:41:03Z",
	"Timestamp": 1638646863,
	"URLs": [
					"https://twitter.com/RnrCapital/status/1467210334679175170"
	],
	"UserID": "1460342597218553856",
	"Username": "RnrCapital",
	"Videos": null
}
`

const incompleteThread1 = `
{
	"Hashtags": null,
	"HTML": "2",
	"ID": "1467216274493034499",
	"InReplyToStatus": {
					"Hashtags": null,
					"HTML": "1",
					"ID": "1467216248891011074",
					"InReplyToStatus": null,
					"IsQuoted": false,
					"IsPin": false,
					"IsReply": false,
					"IsRetweet": false,
					"Likes": 0,
					"PermanentURL": "https://twitter.com/RnrCapital/status/1467216248891011074",
					"Photos": null,
					"Place": null,
					"QuotedStatus": null,
					"Replies": 1,
					"Retweets": 0,
					"RetweetedStatus": null,
					"Text": "1",
					"TimeParsed": "2021-12-04T19:36:27Z",
					"Timestamp": 1638646587,
					"URLs": null,
					"UserID": "1460342597218553856",
					"Username": "RnrCapital",
					"Videos": null
	},
	"IsQuoted": false,
	"IsPin": false,
	"IsReply": true,
	"IsRetweet": false,
	"Likes": 0,
	"PermanentURL": "https://twitter.com/RnrCapital/status/1467216274493034499",
	"Photos": null,
	"Place": null,
	"QuotedStatus": null,
	"Replies": 3,
	"Retweets": 0,
	"RetweetedStatus": null,
	"Text": "2",
	"TimeParsed": "2021-12-04T19:36:34Z",
	"Timestamp": 1638646594,
	"URLs": null,
	"UserID": "1460342597218553856",
	"Username": "RnrCapital",
	"Videos": null
}
`

const incompleteThread2 = `
{
	"Hashtags": null,
	"HTML": "1",
	"ID": "1467216248891011074",
	"InReplyToStatus": null,
	"IsQuoted": false,
	"IsPin": false,
	"IsReply": false,
	"IsRetweet": false,
	"Likes": 0,
	"PermanentURL": "https://twitter.com/RnrCapital/status/1467216248891011074",
	"Photos": null,
	"Place": null,
	"QuotedStatus": null,
	"Replies": 1,
	"Retweets": 0,
	"RetweetedStatus": null,
	"Text": "1",
	"TimeParsed": "2021-12-04T19:36:27Z",
	"Timestamp": 1638646587,
	"URLs": null,
	"UserID": "1460342597218553856",
	"Username": "RnrCapital",
	"Videos": null
}
`

func TestRemoveTwitterLink(t *testing.T) {
	input := `this is https://t.co/sIGZPDyx76 but not https://t.co/sIGZPDyx7asd this one`
	assert.Equal(t, RemoveTwitterLink(input), "this is but not this one")
}

func TestConvertTweetTreeToCrawledPost_ReplyChain(t *testing.T) {
	scraper := twitterscraper.New()
	tweet := &twitterscraper.Tweet{}
	err := json.Unmarshal([]byte(replyChain), tweet)
	assert.Nil(t, err)
	res, err := ConvertTweetTreeToCrawledPost(tweet, scraper, &protocol.PanopticTask{TaskParams: &protocol.TaskParams{SourceId: "source_i"}})
	assert.Nil(t, err)
	assert.Equal(t, res.Content, "3")
	assert.Equal(t, res.ContentGeneratedAt.Seconds, int64(1638646600+2*TimeOffsetSecond))
	assert.Equal(t, res.ReplyTo.Content, "2")
	assert.Equal(t, res.ReplyTo.ContentGeneratedAt.Seconds, int64(1638646594+TimeOffsetSecond))
	assert.Equal(t, res.ReplyTo.ReplyTo.Content, "1")
	assert.Equal(t, res.ReplyTo.ReplyTo.ContentGeneratedAt.Seconds, int64(1638646587))
	assert.Nil(t, res.ReplyTo.ReplyTo.ReplyTo)
}

func TestConvertTweetTreeToCrawledPost_ReplyChainWithQuote(t *testing.T) {
	scraper := twitterscraper.New()
	tweet := &twitterscraper.Tweet{}
	err := json.Unmarshal([]byte(replyWithAndQuote), tweet)
	assert.Nil(t, err)
	res, err := ConvertTweetTreeToCrawledPost(tweet, scraper, &protocol.PanopticTask{TaskParams: &protocol.TaskParams{SourceId: "source_i"}})
	assert.Nil(t, err)
	assert.Equal(t, res.Content, "test")
	assert.Equal(t, res.ReplyTo.Content, "2")
	assert.Equal(t, res.ReplyTo.ReplyTo.Content, "1")
	assert.Nil(t, res.ReplyTo.ReplyTo.ReplyTo)
	assert.Equal(t, res.SharedFromCrawledPost.Content, "quote_1 https://twitter.com/RogerDickerman/status/1467204501899784195")
}

func TestFilterIncompleteTweet(t *testing.T) {
	tweet1 := &twitterscraper.Tweet{}
	tweet2 := &twitterscraper.Tweet{}
	tweet3 := &twitterscraper.Tweet{}

	assert.Nil(t, json.Unmarshal([]byte(completeThread), tweet1))
	assert.Nil(t, json.Unmarshal([]byte(incompleteThread1), tweet2))
	assert.Nil(t, json.Unmarshal([]byte(incompleteThread2), tweet3))

	res := FilterIncompleteTweet([]*twitterscraper.Tweet{tweet1, tweet2, tweet3})
	assert.Equal(t, len(res), 1)
	assert.Equal(t, res[0].ID, "1467217405193568260")
}

func TestCalculateTweetSignature(t *testing.T) {
	tweet := &twitterscraper.Tweet{}
	assert.Nil(t, json.Unmarshal([]byte(completeThread), tweet))
	// combination of all tweets in the thread
	assert.Equal(t, CalcTweetSignature(tweet), "146721740519356826014672162744930344991467216248891011074")
}
