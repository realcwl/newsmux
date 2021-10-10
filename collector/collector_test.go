package collector

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestMain(m *testing.M) {
	dotenv.LoadDotEnvsInTests()
	os.Exit(m.Run())
}

// Construct a HTMLElement according to html raw string and its id
func GetMockHtmlElem(s string, id string) *colly.HTMLElement {
	reader := strings.NewReader(s)
	node, err := html.Parse(reader)
	if err != nil {
		panic(err)
	}
	var targetNode *html.Node

	// find the html node with the specified id
	// doing this because the node from html.Parse by default has <html><body> ... <your elem>... </body></html>
	// need id to identify the elem
	var f func(*html.Node)
	f = func(n *html.Node) {
		for _, a := range n.Attr {
			if a.Key == "id" {
				if a.Val == id {
					targetNode = n
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(node)

	doc := goquery.NewDocumentFromNode(targetNode)
	elem := colly.NewHTMLElementFromSelectionNode(
		&colly.Response{
			Request: nil,
		},
		doc.Selection,
		targetNode,
		0)
	return elem
}

func GetFakeTask(taskId, sourceId, subSourceName string, subsourceType protocol.PanopticSubSource_SubSourceType) protocol.PanopticTask {
	return protocol.PanopticTask{
		TaskId:          taskId,
		DataCollectorId: protocol.PanopticTask_COLLECTOR_KUAILANSI,
		TaskParams: &protocol.TaskParams{
			HeaderParams: []*protocol.KeyValuePair{},
			Cookies:      []*protocol.KeyValuePair{},
			SourceId:     sourceId,
			SubSources: []*protocol.PanopticSubSource{
				{
					Name: subSourceName,
					Type: subsourceType,
				},
			},
		},
		TaskMetadata: &protocol.TaskMetadata{},
	}
}

func TestJin10CrawlerWithTitle(t *testing.T) {
	// var elem colly.HTMLElement
	var sink = NewStdErrSink()
	var builder CollectorBuilder
	crawler := builder.NewJin10Crawler(sink)
	taskId := "task_1"
	sourceId := "source_1"
	task := GetFakeTask(taskId, sourceId, "快讯", protocol.PanopticSubSource_FLASHNEWS)

	t.Run("[get message from dom element][flash] html with title and <br/>", func(t *testing.T) {
		htmlWithTitle := `<div data-v-c7711130="" data-v-471802f2="" id="flash20210926132904521100" class="jin-flash-item-container is-normal"><!----><div data-v-c7711130="" data-relevance="标普" class="jin-flash-item flash is-relevance"><div data-v-2b138c9f="" data-v-c7711130="" class="detail-btn"><div data-v-2b138c9f="" class="detail-btn_container"><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span></div></div><!----><div data-v-c7711130="" class="item-time">13:29:04</div><div data-v-c7711130="" class="item-right is-common"><div data-v-c7711130="" class="right-top"><!----><div data-v-c7711130="" class="right-content"><div data-v-c7711130=""><b>标普：美国经济正在降温 但仍具有弹性</b><br>美国经济已经有所降温，但仍然具有弹性。将美国2021年和2022年实际GDP增速预期分别调整至5.7%和4.1%，此前在6月报告中的预期分别为6.7%和3.7%。尽管美国经济仍处于过热状态，但随着夏季结束，美国经济已经开始降温。供应中断仍是美国经济放缓的主要原因，而德尔塔变种病毒现在是另一个拖累因素。目前的GDP预测仍将是1984年以来的最高水平。预计美联储将在12月开始缩减资产购买规模，并在2022年12月加息，随后分别在2023年和2024年加息两次。</div></div><!----><!----><!----></div></div><div data-v-47f123d2="" data-v-c7711130="" class="flash-item-share" style="display: none;"><a data-v-47f123d2="" href="javascript:void('openShare')" class="share-btn"><i data-v-47f123d2="" class="jin-icon iconfont icon-fenxiang"></i><span data-v-47f123d2="">分享</span></a><a data-v-47f123d2="" href="https://flash.jin10.com/detail/20210926132904521100" target="_blank"><i data-v-47f123d2="" class="jin-icon iconfont icon-xiangqing"></i><span data-v-47f123d2="">详情</span></a><a data-v-47f123d2="" href="javascript:void('copyFlash')"><i data-v-47f123d2="" class="jin-icon iconfont icon-fuzhi"></i><span data-v-47f123d2="">复制</span></a><!----></div></div></div>`
		elem := GetMockHtmlElem(htmlWithTitle, "flash20210926132904521100")
		msg, err := crawler.GetMessage(&task, elem)
		require.NoError(t, err)
		require.NotNil(t, msg)

		require.Equal(t, "d224fa4280bc24e72add1fb0811e2bfa", msg.Post.DeduplicateId)
		require.Equal(t, "标普：美国经济正在降温 但仍具有弹性 美国经济已经有所降温，但仍然具有弹性。将美国2021年和2022年实际GDP增速预期分别调整至5.7%和4.1%，此前在6月报告中的预期分别为6.7%和3.7%。尽管美国经济仍处于过热状态，但随着夏季结束，美国经济已经开始降温。供应中断仍是美国经济放缓的主要原因，而德尔塔变种病毒现在是另一个拖累因素。目前的GDP预测仍将是1984年以来的最高水平。预计美联储将在12月开始缩减资产购买规模，并在2022年12月加息，随后分别在2023年和2024年加息两次。", msg.Post.Content)
		require.Equal(t, 0, len(msg.Post.ImageUrls))
		require.Equal(t, 0, len(msg.Post.FilesUrls))
		require.Equal(t, "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png", msg.Post.SubSource.AvatarUrl)
		require.Equal(t, "快讯", msg.Post.SubSource.Name)
		require.Equal(t, sourceId, msg.Post.SubSource.SourceId)

		tm, _ := time.Parse(jin10DateFormat, "20210926-13:29:04")
		require.Equal(t, tm.Unix(), msg.Post.ContentGeneratedAt.AsTime().Unix())
	})
}

func TestJin10CrawlerWithImage(t *testing.T) {
	// var elem colly.HTMLElement
	var sink = NewStdErrSink()
	var builder CollectorBuilder
	crawler := builder.NewJin10Crawler(sink)
	taskId := "task_1"
	sourceId := "source_1"

	task := GetFakeTask(taskId, sourceId, "要闻", protocol.PanopticSubSource_KEYNEWS)

	t.Run("[get message from dom element][keynews] html with image", func(t *testing.T) {
		htmlWithImage := `<div data-v-c7711130="" data-v-471802f2="" id="flash20210925215015057100" class="jin-flash-item-container is-normal"><!----><div data-v-c7711130="" class="jin-flash-item flash is-important"><div data-v-2b138c9f="" data-v-c7711130="" class="detail-btn"><div data-v-2b138c9f="" class="detail-btn_container"><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span></div></div><!----><div data-v-c7711130="" class="item-time">21:50:15</div><div data-v-c7711130="" class="item-right is-common"><div data-v-c7711130="" class="right-top"><!----><div data-v-c7711130="" class="right-content"><div data-v-c7711130="">孟晚舟乘坐的中国政府包机抵达深圳宝安机场。欢迎回家！（人民日报）</div></div><div data-v-c7711130="" class="right-pic img-intercept flash-pic"><div data-v-c7711130="" class="img-container show-shadow"><img data-v-c7711130="" data-src="https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite" src="https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite" lazy="loaded"></div></div><!----><!----></div></div><div data-v-47f123d2="" data-v-c7711130="" class="flash-item-share" style="display: none;"><a data-v-47f123d2="" href="javascript:void('openShare')" class="share-btn"><i data-v-47f123d2="" class="jin-icon iconfont icon-fenxiang"></i><span data-v-47f123d2="">分享</span></a><a data-v-47f123d2="" href="https://flash.jin10.com/detail/20210925215015057100" target="_blank"><i data-v-47f123d2="" class="jin-icon iconfont icon-xiangqing"></i><span data-v-47f123d2="">详情</span></a><a data-v-47f123d2="" href="javascript:void('copyFlash')"><i data-v-47f123d2="" class="jin-icon iconfont icon-fuzhi"></i><span data-v-47f123d2="">复制</span></a><!----></div></div></div>`
		elem := GetMockHtmlElem(htmlWithImage, "flash20210925215015057100")
		msg, err := crawler.GetMessage(&task, elem)
		require.NoError(t, err)
		require.NotNil(t, msg)

		require.Equal(t, "0002eeca3afa2d000c07e5cdcc0e6333", msg.Post.DeduplicateId)
		require.Equal(t, "孟晚舟乘坐的中国政府包机抵达深圳宝安机场。欢迎回家！（人民日报）", msg.Post.Content)
		require.Equal(t, 1, len(msg.Post.ImageUrls))
		require.Equal(t, "https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite", msg.Post.ImageUrls[0])
		require.Equal(t, 0, len(msg.Post.FilesUrls))
		require.Equal(t, "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png", msg.Post.SubSource.AvatarUrl)
		require.Equal(t, "要闻", msg.Post.SubSource.Name)
		require.Equal(t, sourceId, msg.Post.SubSource.SourceId)

		tm, _ := time.Parse(jin10DateFormat, "20210925-21:50:15")
		require.Equal(t, tm.Unix(), msg.Post.ContentGeneratedAt.AsTime().Unix())
	})
}

func TestJin10CrawlerNotMatchingRequest(t *testing.T) {
	// var elem colly.HTMLElement
	var sink = NewStdErrSink()
	var builder CollectorBuilder
	crawler := builder.NewJin10Crawler(sink)
	taskId := "task_1"
	sourceId := "source_1"
	// Asking for Flash news
	task := GetFakeTask(taskId, sourceId, "快讯", protocol.PanopticSubSource_FLASHNEWS)
	t.Run("[get message from dom element][keynews] not request specified", func(t *testing.T) {
		// Got Key news
		htmlWithImage := `<div data-v-c7711130="" data-v-471802f2="" id="flash20210925215015057100" class="jin-flash-item-container is-normal"><!----><div data-v-c7711130="" class="jin-flash-item flash is-important"><div data-v-2b138c9f="" data-v-c7711130="" class="detail-btn"><div data-v-2b138c9f="" class="detail-btn_container"><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span></div></div><!----><div data-v-c7711130="" class="item-time">21:50:15</div><div data-v-c7711130="" class="item-right is-common"><div data-v-c7711130="" class="right-top"><!----><div data-v-c7711130="" class="right-content"><div data-v-c7711130="">孟晚舟乘坐的中国政府包机抵达深圳宝安机场。欢迎回家！（人民日报）</div></div><div data-v-c7711130="" class="right-pic img-intercept flash-pic"><div data-v-c7711130="" class="img-container show-shadow"><img data-v-c7711130="" data-src="https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite" src="https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite" lazy="loaded"></div></div><!----><!----></div></div><div data-v-47f123d2="" data-v-c7711130="" class="flash-item-share" style="display: none;"><a data-v-47f123d2="" href="javascript:void('openShare')" class="share-btn"><i data-v-47f123d2="" class="jin-icon iconfont icon-fenxiang"></i><span data-v-47f123d2="">分享</span></a><a data-v-47f123d2="" href="https://flash.jin10.com/detail/20210925215015057100" target="_blank"><i data-v-47f123d2="" class="jin-icon iconfont icon-xiangqing"></i><span data-v-47f123d2="">详情</span></a><a data-v-47f123d2="" href="javascript:void('copyFlash')"><i data-v-47f123d2="" class="jin-icon iconfont icon-fuzhi"></i><span data-v-47f123d2="">复制</span></a><!----></div></div></div>`
		elem := GetMockHtmlElem(htmlWithImage, "flash20210925215015057100")
		msg, err := crawler.GetMessage(&task, elem)
		require.Error(t, err)
		require.Nil(t, msg)
	})
}

func TestJin10CollectorHandler(t *testing.T) {
	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{
			{
				TaskId:          "123",
				DataCollectorId: protocol.PanopticTask_COLLECTOR_JINSHI,
				TaskParams: &protocol.TaskParams{
					HeaderParams: []*protocol.KeyValuePair{},
					Cookies:      []*protocol.KeyValuePair{},
					SourceId:     "123",
					SubSources: []*protocol.PanopticSubSource{
						{
							Name:       "快讯",
							Type:       protocol.PanopticSubSource_FLASHNEWS,
							ExternalId: "1",
						},
					},
				},
			},
			{
				TaskId:          "456",
				DataCollectorId: protocol.PanopticTask_COLLECTOR_JINSHI,
				TaskParams: &protocol.TaskParams{
					HeaderParams: []*protocol.KeyValuePair{},
					Cookies:      []*protocol.KeyValuePair{},
					SourceId:     "456",
					SubSources: []*protocol.PanopticSubSource{
						{
							Name:       "要闻",
							Type:       protocol.PanopticSubSource_KEYNEWS,
							ExternalId: "2",
						},
					},
				},
			},
		}}
	var handler DataCollectJobHandler
	err := handler.Collect(&job)
	fmt.Println("job", job.String())
	require.NoError(t, err)
	require.Equal(t, 2, len(job.Tasks))
	require.Equal(t, "123", job.Tasks[0].TaskId)
	require.Greater(t, job.Tasks[0].TaskMetadata.TotalMessageCollected, int32(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TotalMessageFailed, int32(0))
	require.Equal(t, "456", job.Tasks[1].TaskId)
	require.Greater(t, job.Tasks[0].TaskMetadata.TotalMessageCollected, int32(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskStartTime.Seconds, int64(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskEndTime.Seconds, int64(0))
}

func TestIpAddressFetch(t *testing.T) {
	var client HttpClient
	ip, err := GetCurrentIpAddress(client)
	require.NoError(t, err)
	fmt.Println("ip: ", ip)
	require.Greater(t, len(ip), 0)
}
