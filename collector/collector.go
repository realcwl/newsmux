package collector

import (
	"github.com/Luismorlan/newsmux/collector/working_context"
	"github.com/Luismorlan/newsmux/protocol"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DataCollector interface {
	CollectAndPublish(task *protocol.PanopticTask)
}

// To make sure the interface is implemented
// we use builder to get collector which can enforce the Interface for
// Crawler, API and RSS collector instances
type CrawlerCollector interface {
	DataCollector

	GetMessage(*working_context.CrawlerWorkingContext) error

	// All implementation functions should output error
	// errors will be reported for debugging
	GetQueryPath() string
	GetStartUri() string

	UpdateContent(workingContext *working_context.CrawlerWorkingContext) error
	UpdateDedupId(workingContext *working_context.CrawlerWorkingContext) error
	UpdateGeneratedTime(workingContext *working_context.CrawlerWorkingContext) error
	UpdateNewsType(workingContext *working_context.CrawlerWorkingContext) error
	UpdateImageUrls(workingContext *working_context.CrawlerWorkingContext) error
	UpdateFileUrls(workingContext *working_context.CrawlerWorkingContext) error
}

// In API collector API, not like Crawler where we usually
// only know what is the subsource(s) after checking the crawled page
// API usually able to explicitly ask for subsource, thus in the APIs
// we often can pass explicit subsource
type ApiCollector interface {
	DataCollector
	CollectOneSubsource(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource) error
	CollectOneSubsourceOnePage(
		task *protocol.PanopticTask,
		subsource *protocol.PanopticSubSource,
		paginationInfo *working_context.PaginationInfo,
	) error
	UpdateFileUrls(workingContext *working_context.ApiCollectorWorkingContext) error
	ConstructUrl(task *protocol.PanopticTask, subsource *protocol.PanopticSubSource, paginationInfo *working_context.PaginationInfo) string
}

type RssCollector interface {
	DataCollector
	// TODO: implement rss collector
}

// This is the main entry point that runs collection. It assumes that the task
// execution result is always SUCCESS, unless encountered error during
// collection.
func RunCollectorForTask(collector DataCollector, task *protocol.PanopticTask) {
	if task.TaskMetadata == nil {
		task.TaskMetadata = &protocol.TaskMetadata{}
	}

	task.TaskMetadata.TaskStartTime = timestamppb.Now()
	// Initially we assume this task is going to succeed,
	task.TaskMetadata.ResultState = protocol.TaskMetadata_STATE_SUCCESS
	collector.CollectAndPublish(task)
	task.TaskMetadata.TaskEndTime = timestamppb.Now()
}
