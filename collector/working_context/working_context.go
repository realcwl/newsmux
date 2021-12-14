package working_context

import (
	"fmt"

	"github.com/gocolly/colly"

	"github.com/Luismorlan/newsmux/protocol"
)

type SharedContext struct {
	Task                 *protocol.PanopticTask
	Result               *protocol.CrawlerMessage
	IntentionallySkipped bool
}

type PaginationInfo struct {
	CurrentPageCount int
	NextPageId       string
}

// This is the contxt we keep to be used for all the steps
// for a post
// Initialized with task and element
// All steps can put additional information into this object to pass down to next step
type CrawlerWorkingContext struct {
	SharedContext

	Element        *colly.HTMLElement
	OriginUrl      string
	ExternalPostId string
	NewsType       protocol.PanopticSubSource_SubSourceType
	// Optional, this is the working subsource we're dealing with.
	SubSource *protocol.PanopticSubSource
}

// This is the context we keep to be used for all steps
// for a post
type ApiCollectorWorkingContext struct {
	SharedContext

	PaginationInfo  *PaginationInfo
	ApiUrl          string
	SubSource       *protocol.PanopticSubSource
	NewsType        protocol.PanopticSubSource_SubSourceType
	ApiResponseItem interface{}
}

// This is the context we keep to be used for all steps
// for a post
type RssCollectorWorkingContext struct {
	SharedContext

	RssUrl          string
	SubSource       *protocol.PanopticSubSource
	RssResponseItem interface{}
}

func (sc *SharedContext) String() string {
	return fmt.Sprintf("SharedContext is: task: %s \n result: %s \n", sc.Task.String(), sc.Result.String())
}
