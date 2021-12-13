package collector_instances

import (
	"fmt"

	"github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	twitterscraper "github.com/n0madic/twitter-scraper"
)

type TwitterApiCrawler struct {
	Sink sink.CollectedDataSink

	Scraper *twitterscraper.Scraper
}

// Crawl and publish for a single Twitter user.
func (t TwitterApiCrawler) ProcessSingleSubSource(
	subSource *protocol.PanopticSubSource, task *protocol.PanopticTask) {
	tweets, _, err := t.Scraper.FetchTweets(subSource.ExternalId, 20, "")
	if err != nil {
		Logger.Log.Errorln("fail to collect tweeter user", subSource.ExternalId, err)
		task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		return
	}
	for _, tweet := range tweets {
		t.ProcessSingleTweet(tweet, task)
	}
}

func (t TwitterApiCrawler) ProcessSingleTweet(tweet *twitterscraper.Tweet,
	task *protocol.PanopticTask) {
	workingContext := &working_context.ApiCollectorWorkingContext{
		SharedContext:   working_context.SharedContext{Task: task, IntentionallySkipped: false},
		ApiResponseItem: tweet,
	}
	if err := t.GetMessage(workingContext); err != nil {
		task.TaskMetadata.TotalMessageFailed++
		Logger.Log.Errorln(fmt.Sprintf("fail to collect twitter message from API response. message %s, err %s", collector.PrettyPrint(tweet), err))
		return
	}
	sink.PushResultToSinkAndRecordInTaskMetadata(t.Sink, workingContext)
}

func (t TwitterApiCrawler) GetMessage(workingContext *working_context.ApiCollectorWorkingContext) error {
	collector.InitializeApiCollectorResult(workingContext)
	tweet := workingContext.ApiResponseItem.(*twitterscraper.Tweet)
	post, err := ConvertTweetTreeToCrawledPost(tweet, t.Scraper)
	if err != nil {
		return err
	}
	workingContext.Result.Post = post
	workingContext.Task.TaskMetadata.TotalMessageCollected++
	return nil
}

func (t TwitterApiCrawler) CollectAndPublish(task *protocol.PanopticTask) {
	for _, subSource := range task.TaskParams.SubSources {
		t.ProcessSingleSubSource(subSource, task)
	}

	collector.SetErrorBasedOnCounts(task, "Twitter")
}
