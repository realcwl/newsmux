package collector_instances

import (
	"regexp"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const TWITTER_SOURCE_ID = "a19df1ae-3c80-4ffc-b8e6-cefb3a6a3c27"

// Package level cache that stores the profile result. This is to avoid fetching
// static profile information multiple times, which can introduce extreme
// latency.
var profileCache = make(map[string]*twitterscraper.Profile)

func GetUserProfile(username string, scraper *twitterscraper.Scraper) (
	*twitterscraper.Profile, error) {
	if profile, ok := profileCache[username]; ok {
		return profile, nil
	}
	// Fetch and store
	profile, err := scraper.GetProfile(username)
	if err != nil {
		return nil, err
	}
	profileCache[username] = &profile
	return &profile, nil
}

// Sometimes Twitter content would return links directly in text, in which case
// we want to remove.
// e.g. "https://t.co/sIGZPDyx76"
func RemoveTwitterLink(content string) string {
	reg := regexp.MustCompile(`https:\/\/t.co\/[A-Za-z0-9]*`)
	linkRemoved := reg.ReplaceAllString(content, "")
	return strings.TrimSpace(strings.ReplaceAll(linkRemoved, "  ", " "))
}

func GetTwitterContent(tweet *twitterscraper.Tweet) string {
	// Retweet should not have actual content
	if tweet.IsRetweet {
		return ""
	}
	linkRemoved := RemoveTwitterLink(tweet.Text)
	linkRemoved += "\n" + strings.Join(tweet.URLs, "\n")
	return linkRemoved
}

func GetTwitterDedupId(tweet *twitterscraper.Tweet) string {
	return tweet.ID
}

func GetTwitterCreationTime(tweet *twitterscraper.Tweet) *timestamppb.Timestamp {
	return timestamppb.New(time.Unix(tweet.Timestamp, 0))
}

func GetTwitterImageUrls(tweet *twitterscraper.Tweet) []string {
	return tweet.Photos
}

// Convert a tweet to crawled message together with the tweet it is refering to
// (quote/retweet), stored as the SharedFromCrawledPost field. This function
// will not convert reply thread, which will be dealt with in another function.
func ConvertTweetToCrawledPost(tweet *twitterscraper.Tweet, scraper *twitterscraper.Scraper) (*protocol.CrawlerMessage_CrawledPost, error) {
	post, err := ConvertSingleTweetToCrawledPost(tweet, scraper)
	if err != nil {
		return nil, err
	}

	var sharedPost *protocol.CrawlerMessage_CrawledPost
	if tweet.IsRetweet {
		sharedPost, err = ConvertSingleTweetToCrawledPost(tweet.RetweetedStatus, scraper)
		if err != nil {
			return nil, err
		}
	} else if tweet.IsQuoted {
		sharedPost, err = ConvertSingleTweetToCrawledPost(tweet.QuotedStatus, scraper)
		if err != nil {
			return nil, err
		}
	}
	post.SharedFromCrawledPost = sharedPost
	return post, nil
}

// Convert from Tweet object to CralwedMessage without constructing the inner
// shared post.
func ConvertSingleTweetToCrawledPost(tweet *twitterscraper.Tweet, scraper *twitterscraper.Scraper) (*protocol.CrawlerMessage_CrawledPost, error) {
	profile, err := GetUserProfile(tweet.Username, scraper)
	if err != nil {
		Logger.Log.Errorln("fail to get profile for user", tweet.Username)
		return nil, err
	}

	post := &protocol.CrawlerMessage_CrawledPost{
		Content:            GetTwitterContent(tweet),
		DeduplicateId:      GetTwitterDedupId(tweet),
		ImageUrls:          GetTwitterImageUrls(tweet),
		ContentGeneratedAt: GetTwitterCreationTime(tweet),
		SubSource: &protocol.CrawledSubSource{
			Name:       profile.Name,
			AvatarUrl:  profile.Avatar,
			SourceId:   TWITTER_SOURCE_ID,
			ExternalId: profile.Username,
			OriginUrl:  profile.URL,
		},
		OriginUrl: tweet.PermanentURL,
	}

	return post, nil
}

// This function will parse and convert a single tweet object into a crawled
// message, together with the reply chain, and quote/retweet in each layer.
// This is the entry point for most of the tweet
func ConvertTweetTreeToCrawledPost(
	root *twitterscraper.Tweet, scraper *twitterscraper.Scraper) (*protocol.CrawlerMessage_CrawledPost, error) {
	post, err := ConvertTweetToCrawledPost(root, scraper)
	if err != nil {
		return nil, err
	}
	// Fast return if the post is a leaf in the reply chain.
	if !root.IsReply {
		return post, nil
	}
	replyTweet, err := ConvertTweetTreeToCrawledPost(root.InReplyToStatus, scraper)
	if err != nil {
		return nil, err
	}

	post.ReplyTo = replyTweet

	return post, nil
}
