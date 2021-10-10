package collector

import (
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/gocolly/colly"
)

type CollectedDataSink interface {
	Push(msg *protocol.CrawlerMessage) error
}

type DataCollector interface {
	CollectAndPublish(task *protocol.PanopticTask)
	GetMessage(task *protocol.PanopticTask, elem *colly.HTMLElement, originUrl string) (*protocol.CrawlerMessage, error)
}

// To make sure the interface is implemented
// we use builder to get collector which can enforce the Interface for
// Crawler, API and RSS collector instances
type CrawlerCollector interface {
	DataCollector

	// All implementation functions should output error
	// errors will be reported for debugging
	GetQueryPath() string
	GetStartUri() string
	GetContent(task *protocol.PanopticTask, elem *colly.HTMLElement) (string, error)
	GetDedupId(task *protocol.PanopticTask, content string, id string) (string, error)
	GetGeneratedTime(task *protocol.PanopticTask, elem *colly.HTMLElement) (time.Time, error)
	GetLevel(elem *colly.HTMLElement) (protocol.PanopticSubSource_SubSourceType, error)
	GetImageUrls(task *protocol.PanopticTask, elem *colly.HTMLElement) ([]string, error)
	GetFileUrls(task *protocol.PanopticTask, elem *colly.HTMLElement) ([]string, error)

	IsRequested(task *protocol.PanopticTask, level protocol.PanopticSubSource_SubSourceType) bool
}

type ApiCollector interface {
	DataCollector
	// TODO: implement api collector
}

type RssCollector interface {
	DataCollector
	// TODO: implement rss collector
}
