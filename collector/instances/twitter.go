package collector_instances

import (
	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/clients"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

type TwitterApiCrawler struct {
	Sink sink.CollectedDataSink

	// A thin wrapper upon http.Client to make request to Twitter V2 API.
	Client *clients.TwitterClient
}

func (t TwitterApiCrawler) GetTwitterType(
	tweet *clients.GetUserTweetsResponseData, res *clients.GetUserTweetsResponse) clients.TweetType {
	if len(tweet.ReferencedTweets) == 0 {
		return clients.TWEET_TYPE_SINGLE
	}

	if len(tweet.ReferencedTweets) == 1 {
		switch referenceType := tweet.ReferencedTweets[0].Type; referenceType {
		case clients.REFERENCE_TYPE_QUOTED:
			return clients.TWEET_TYPE_QUOTE
		case clients.REFERENCE_TYPE_RETWEETED:
			return clients.TWEET_TYPE_RETWEET
		case clients.REFERENCE_TYPE_REPLIED_TO:
			return clients.TWEET_TYPE_THREAD
		}
	}

	// As of now, I haven't seen any tweeter containing more than one referenced
	// tweeter. Thus in this case
	return clients.TWEET_TYPE_UNKNOWN
}

func (t TwitterApiCrawler) ProcessSingleTwitterPost(
	tweet *clients.GetUserTweetsResponseData, res *clients.GetUserTweetsResponse) {
}

// Crawl and publish for a single Twitter user.
func (t TwitterApiCrawler) ProcessSingleSubSource(
	subSource *protocol.PanopticSubSource, task *protocol.PanopticTask) {
	res, err := t.Client.GetUserTweets(subSource.ExternalId)
	if err != nil {
		Logger.Log.Errorln("fail to collect tweeter user", subSource.ExternalId)
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
	}
	for _, tweet := range res.Data {
		Logger.Log.Infoln(collector.PrettyPrint(tweet))
	}
}

func (t TwitterApiCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	for _, subSource := range task.TaskParams.SubSources {
		t.ProcessSingleSubSource(subSource, task)
	}

	collector.SetErrorBasedOnCounts(task, "Twitter")
}
