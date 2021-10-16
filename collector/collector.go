package collector

import (
	"github.com/Luismorlan/newsmux/protocol"
	"github.com/gocolly/colly"
)

type CollectedDataSink interface {
	Push(msg *protocol.CrawlerMessage) error
}

type CollectedFileStore interface {
	FetchAndStore(url string) error
}

// This is the contxt we keep to be used for all the steps
// Initialized with task and element
// All steps can put additional information into this object to pass down to next step
type CrawlerWorkingContext struct {
	Task           *protocol.PanopticTask
	Element        *colly.HTMLElement
	OriginUrl      string
	ExternalPostId string
	NewsType       protocol.PanopticSubSource_SubSourceType
	// final result of crawled message for each news
	Result *protocol.CrawlerMessage
}

type DataCollector interface {
	CollectAndPublish(task *protocol.PanopticTask)
}

// To make sure the interface is implemented
// we use builder to get collector which can enforce the Interface for
// Crawler, API and RSS collector instances
type CrawlerCollector interface {
	DataCollector

	GetMessage(*CrawlerWorkingContext) error

	// All implementation functions should output error
	// errors will be reported for debugging
	GetQueryPath() string
	GetStartUri() string

	UpdateContent(workingContext *CrawlerWorkingContext) error
	UpdateDedupId(workingContext *CrawlerWorkingContext) error
	UpdateGeneratedTime(workingContext *CrawlerWorkingContext) error
	UpdateNewsType(workingContext *CrawlerWorkingContext) error
	UpdateImageUrls(workingContext *CrawlerWorkingContext) error
	UpdateFileUrls(workingContext *CrawlerWorkingContext) error

	IsRequested(workingContext *CrawlerWorkingContext) bool
}

type ApiCollector interface {
	DataCollector
	// TODO: implement api collector
}

type RssCollector interface {
	DataCollector
	// TODO: implement rss collector
}

// Shared Func type for file stores
type ProcessUrlBeforeFetchFuncType func(string) string
type CustomizeFileNameFuncType func(string) string
type CustomizeFileExtFuncType func(string) string
