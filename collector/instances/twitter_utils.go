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

const (
	// Offset the tweet's creation time in each depth by 5 second, so that the
	// creation time will not be the same, and thus tweets can be sorted by the
	// creation time to determine the parentship. This is kinda hacky but save us
	// lots of code in API Server where we don't need to explicitly record the
	// index of a post in thread.
	TimeOffsetSecond = 5
)

// Package level cache that stores the profile result. This is to avoid fetching
// static profile information multiple times, which can introduce extreme
// latency.
// The key here is user id.
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

// For most cases, twitter content is just the text field. In cases where links
// are inserted into a single tweet, we join the links together with the text.
// In the case
func GetTwitterContent(tweet *twitterscraper.Tweet, isQuoted bool) string {
	// Retweet should not have actual content
	if tweet.IsRetweet {
		return ""
	}

	baseText := RemoveTwitterLink(tweet.Text)

	// Append urls that are not part of the quoted tweet.
	for _, URL := range tweet.URLs {
		if tweet.QuotedStatus != nil && URL == tweet.QuotedStatus.PermanentURL {
			continue
		}
		baseText += "\n" + URL
	}

	if !isQuoted || !tweet.IsQuoted || tweet.QuotedStatus == nil {
		return baseText
	}
	// In the case that this is a quote post, we mimic how Twitter deals with this
	// case: https://twitter.com/RnrCapital/status/1467217405193568260
	return baseText + " " + tweet.QuotedStatus.PermanentURL
}

func GetTwitterDedupId(tweet *twitterscraper.Tweet) string {
	return tweet.ID
}

func GetTwitterCreationTime(tweet *twitterscraper.Tweet, depth int) *timestamppb.Timestamp {
	return timestamppb.New(time.Unix(tweet.Timestamp+int64(depth*TimeOffsetSecond), 0))
}

func GetTwitterImageUrls(tweet *twitterscraper.Tweet) []string {
	return tweet.Photos
}

// Convert a tweet to crawled message together with the tweet it is refering to
// (quote/retweet), stored as the SharedFromCrawledPost field. This function
// will not convert reply thread, which will be dealt with in another function.
func ConvertTweetToCrawledPost(tweet *twitterscraper.Tweet, scraper *twitterscraper.Scraper, depth *int, task *protocol.PanopticTask) (*protocol.CrawlerMessage_CrawledPost, error) {
	post, err := ConvertSingleTweetToCrawledPost(tweet, scraper, false, depth, task)
	if err != nil {
		return nil, err
	}

	var sharedPost *protocol.CrawlerMessage_CrawledPost
	if tweet.IsRetweet {
		sharedPost, err = ConvertSingleTweetToCrawledPost(tweet.RetweetedStatus, scraper, false, depth, task)
		if err != nil {
			return nil, err
		}
	} else if tweet.IsQuoted {
		sharedPost, err = ConvertSingleTweetToCrawledPost(tweet.QuotedStatus, scraper, true, depth, task)
		if err != nil {
			return nil, err
		}
	}
	post.SharedFromCrawledPost = sharedPost
	return post, nil
}

// Convert from Tweet object to CralwedMessage without constructing the inner
// shared post.
func ConvertSingleTweetToCrawledPost(
	tweet *twitterscraper.Tweet, scraper *twitterscraper.Scraper, isQuoted bool, depth *int, task *protocol.PanopticTask) (
	*protocol.CrawlerMessage_CrawledPost, error) {
	profile, err := GetUserProfile(tweet.Username, scraper)
	if err != nil {
		Logger.Log.Errorln("fail to get profile for user", tweet.Username)
		return nil, err
	}

	post := &protocol.CrawlerMessage_CrawledPost{
		Content:            GetTwitterContent(tweet, isQuoted),
		DeduplicateId:      GetTwitterDedupId(tweet),
		ImageUrls:          GetTwitterImageUrls(tweet),
		ContentGeneratedAt: GetTwitterCreationTime(tweet, *depth),
		SubSource: &protocol.CrawledSubSource{
			Name:       profile.Name,
			AvatarUrl:  profile.Avatar,
			SourceId:   task.TaskParams.SourceId,
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
	root *twitterscraper.Tweet, scraper *twitterscraper.Scraper, task *protocol.PanopticTask) (*protocol.CrawlerMessage_CrawledPost, error) {
	depth := 0
	return ConvertTweetTreeToCrawledPostInternal(root, scraper, &depth, task)
}

func ConvertTweetTreeToCrawledPostInternal(
	root *twitterscraper.Tweet, scraper *twitterscraper.Scraper, depth *int, task *protocol.PanopticTask) (*protocol.CrawlerMessage_CrawledPost, error) {
	var replyTweet *protocol.CrawlerMessage_CrawledPost
	var err error

	// Fast return if the post is a leaf in the reply chain.
	if root.IsReply && root.InReplyToStatus != nil {
		replyTweet, err = ConvertTweetTreeToCrawledPostInternal(root.InReplyToStatus, scraper, depth, task)
		if err != nil {
			return nil, err
		}
	}

	post, err := ConvertTweetToCrawledPost(root, scraper, depth, task)
	if err != nil {
		return nil, err
	}
	if replyTweet != nil {
		post.ReplyTo = replyTweet
	}

	// increase depth by 1 for backtracking
	*depth++
	return post, nil
}

// When user created a thread, each tweet in the thread will be returned as an
// array. For example, when user created a thread a - b - c, the returned array
// will be:
// [
//  c - b - a,
//  b - a,
//  a
// ]
// In this case, we should filter out incomplete thread, and just keep the
// c - b - a part.
// The input tweets *MUST* always be reverse chrononical order. This is very
// important because we assume the later twitter should never "contain" the
// previous tweet.
func FilterIncompleteTweet(tweets []*twitterscraper.Tweet) []*twitterscraper.Tweet {
	res := []*twitterscraper.Tweet{}
	for _, tweet := range tweets {
		if IsTweetIncluded(tweet, res) {
			continue
		}
		res = append(res, tweet)
	}
	return res
}

func IsTweetIncluded(needle *twitterscraper.Tweet, hay []*twitterscraper.Tweet) bool {
	sigNeedle := CalcTweetSignature(needle)
	for _, tweet := range hay {
		sigHay := CalcTweetSignature(tweet)
		if strings.HasSuffix(sigHay, sigNeedle) {
			return true
		}
	}

	return false
}

// Return the id concatenation of all tweets in the thread.
func CalcTweetSignature(tweet *twitterscraper.Tweet) string {
	var sb strings.Builder
	for tweet != nil {
		sb.WriteString(tweet.ID)
		tweet = tweet.InReplyToStatus
	}
	return sb.String()
}
