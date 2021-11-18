package sink

import (
	"github.com/Luismorlan/newsmux/collector/validation"
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

// Push a result into data sink and update task metadata on working context
// - On Success: Increment collected message counter on task.
// - On Failure: Log error and increment failure message counter on task.
func PushResultToSinkAndRecordInTaskMetadata(s CollectedDataSink, workingContext interface{}) {
	var shared_context *working_context.SharedContext
	switch workingContext := workingContext.(type) {
	case *working_context.CrawlerWorkingContext:
		shared_context = &workingContext.SharedContext
	case *working_context.ApiCollectorWorkingContext:
		shared_context = &workingContext.SharedContext
	case *working_context.RssCollectorWorkingContext:
		shared_context = &workingContext.SharedContext
	}

	if shared_context.IntentionallySkipped {
		shared_context.Task.TaskMetadata.TotalMessageSkipped++
		return
	}

	if err := validation.ValidateSharedContext(shared_context); err != nil {
		shared_context.Task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		shared_context.Task.TaskMetadata.TotalMessageFailed++
		Logger.Log.Errorf("crawled message failed validation, Error: %s", err)
		return
	}

	if err := s.Push(shared_context.Result); err != nil {
		shared_context.Task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_FAILURE
		shared_context.Task.TaskMetadata.TotalMessageFailed++
		Logger.Log.Errorf("fail to publish message %s to Sink. Task: %s, Error: %s", shared_context.Result.String(), shared_context.Task.String(), err)
		return
	}
	shared_context.Task.TaskMetadata.TotalMessageCollected++
}
