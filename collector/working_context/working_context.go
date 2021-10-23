package working_context

import (
	"fmt"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/gocolly/colly"
)

type SharedContext struct {
	Task   *protocol.PanopticTask
	Result *protocol.CrawlerMessage
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
}

// This is the context we keep to be used for all steps
// for a post
type ApiCollectorWorkingContext struct {
	SharedContext

	PaginationInfo *PaginationInfo
	ApiUrl         string
	SubSource      *protocol.PanopticSubSource
	NewsType       protocol.PanopticSubSource_SubSourceType
}

func (sc *SharedContext) String() string {
	return fmt.Sprintf("==SharedContext is==\ntask: \n%s\nresult:\n%s\n", sc.Task.String(), sc.Result.String())
}
