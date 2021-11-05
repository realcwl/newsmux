package test

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Luismorlan/newsmux/collector"
	. "github.com/Luismorlan/newsmux/collector"
	. "github.com/Luismorlan/newsmux/collector/builder"
	"github.com/Luismorlan/newsmux/collector/file_store"
	. "github.com/Luismorlan/newsmux/collector/handler"
	. "github.com/Luismorlan/newsmux/collector/instances"
	"github.com/Luismorlan/newsmux/collector/sink"
	"github.com/Luismorlan/newsmux/collector/working_context"
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
func TestJin10Ads(t *testing.T) {
	// var elem colly.HTMLElement
	var s = sink.NewStdErrSink()
	var builder CollectorBuilder
	crawler := builder.NewJin10Crawler(s)
	taskId := "task_1"
	sourceId := "a882eb0d-0bde-401a-b708-a7ce352b7392"
	task := GetFakeTask(taskId, sourceId, "快讯", protocol.PanopticSubSource_FLASHNEWS)

	t.Run("[get message from dom element][flash] html with title and <br/>", func(t *testing.T) {
		htmlWithTitle := `<div data-v-c7711130="" data-v-1bfa56cb="" id="flash20211105105528759100" class="jin-flash-item-container is-normal"><!----><div data-v-c7711130="" class="jin-flash-item flash is-important"><div data-v-2b138c9f="" data-v-c7711130="" class="detail-btn"><div data-v-2b138c9f="" class="detail-btn_container"><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span></div></div><!----><div data-v-c7711130="" class="item-time">10:55:28</div><div data-v-c7711130="" class="item-right is-common"><div data-v-c7711130="" class="right-top"><!----><div data-v-c7711130="" class="right-content"><div data-v-c7711130=""><b>【双11·击穿底价】新用户开通88折，最高加赠41天，相当于享85折优惠！老用户续费更低至6.2元/日！非农数据公布在即，成为VIP，立即查收交易策略＞＞</b></div></div><!----><!----><div data-v-1696ba0d="" data-v-c7711130="" class="flash-remark"><!----><a data-v-1696ba0d="" href="https://www.jin10.com/activities/vip_promotion/double-eleven/index.html" target="_blank" class="remark-item"><i data-v-1696ba0d="" class="jin-icon iconfont icon-zhujielianjie"></i><span data-v-1696ba0d="" class="remark-item-title">相关链接 &gt;&gt;</span></a><!----><!----></div></div></div><div data-v-47f123d2="" data-v-c7711130="" class="flash-item-share" style="display: none;"><a data-v-47f123d2="" href="javascript:void('openShare')" class="share-btn"><i data-v-47f123d2="" class="jin-icon iconfont icon-fenxiang"></i><span data-v-47f123d2="">分享</span></a><a data-v-47f123d2="" href="https://flash.jin10.com/detail/20211105105528759100" target="_blank"><i data-v-47f123d2="" class="jin-icon iconfont icon-xiangqing"></i><span data-v-47f123d2="">详情</span></a><a data-v-47f123d2="" href="javascript:void('copyFlash')"><i data-v-47f123d2="" class="jin-icon iconfont icon-fuzhi"></i><span data-v-47f123d2="">复制</span></a><!----></div></div></div>`
		elem := GetMockHtmlElem(htmlWithTitle, "flash20211105105528759100")
		ctx := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: &task, IntentionallySkipped: false},
			Element:       elem, OriginUrl: "a.com"}
		err := crawler.GetMessage(ctx)
		require.NoError(t, err)
		require.True(t, ctx.IntentionallySkipped)
	})
}

func TestJin10CrawlerWithTitle(t *testing.T) {
	// var elem colly.HTMLElement
	var s = sink.NewStdErrSink()
	var builder CollectorBuilder
	crawler := builder.NewJin10Crawler(s)
	taskId := "task_1"
	sourceId := "a882eb0d-0bde-401a-b708-a7ce352b7392"
	task := GetFakeTask(taskId, sourceId, "快讯", protocol.PanopticSubSource_FLASHNEWS)

	t.Run("[get message from dom element][flash] html with title and <br/>", func(t *testing.T) {
		htmlWithTitle := `<div data-v-c7711130="" data-v-471802f2="" id="flash20210926132904521100" class="jin-flash-item-container is-normal"><!----><div data-v-c7711130="" data-relevance="标普" class="jin-flash-item flash is-relevance"><div data-v-2b138c9f="" data-v-c7711130="" class="detail-btn"><div data-v-2b138c9f="" class="detail-btn_container"><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span></div></div><!----><div data-v-c7711130="" class="item-time">13:29:04</div><div data-v-c7711130="" class="item-right is-common"><div data-v-c7711130="" class="right-top"><!----><div data-v-c7711130="" class="right-content"><div data-v-c7711130=""><b>标普：美国经济正在降温 但仍具有弹性</b><br>美国经济已经有所降温，但仍然具有弹性。将美国2021年和2022年实际GDP增速预期分别调整至5.7%和4.1%，此前在6月报告中的预期分别为6.7%和3.7%。尽管美国经济仍处于过热状态，但随着夏季结束，美国经济已经开始降温。供应中断仍是美国经济放缓的主要原因，而德尔塔变种病毒现在是另一个拖累因素。目前的GDP预测仍将是1984年以来的最高水平。预计美联储将在12月开始缩减资产购买规模，并在2022年12月加息，随后分别在2023年和2024年加息两次。</div></div><!----><!----><!----></div></div><div data-v-47f123d2="" data-v-c7711130="" class="flash-item-share" style="display: none;"><a data-v-47f123d2="" href="javascript:void('openShare')" class="share-btn"><i data-v-47f123d2="" class="jin-icon iconfont icon-fenxiang"></i><span data-v-47f123d2="">分享</span></a><a data-v-47f123d2="" href="https://flash.jin10.com/detail/20210926132904521100" target="_blank"><i data-v-47f123d2="" class="jin-icon iconfont icon-xiangqing"></i><span data-v-47f123d2="">详情</span></a><a data-v-47f123d2="" href="javascript:void('copyFlash')"><i data-v-47f123d2="" class="jin-icon iconfont icon-fuzhi"></i><span data-v-47f123d2="">复制</span></a><!----></div></div></div>`
		elem := GetMockHtmlElem(htmlWithTitle, "flash20210926132904521100")
		ctx := &working_context.CrawlerWorkingContext{
			SharedContext: working_context.SharedContext{Task: &task, IntentionallySkipped: false},
			Element:       elem, OriginUrl: "a.com"}
		err := crawler.GetMessage(ctx)
		msg := ctx.Result
		require.NoError(t, err)
		require.NotNil(t, msg)

		require.Equal(t, "7e85d9a10e1ac1dbbf9c4c14989a9c6f", msg.Post.DeduplicateId)
		require.Equal(t, "标普：美国经济正在降温 但仍具有弹性 美国经济已经有所降温，但仍然具有弹性。将美国2021年和2022年实际GDP增速预期分别调整至5.7%和4.1%，此前在6月报告中的预期分别为6.7%和3.7%。尽管美国经济仍处于过热状态，但随着夏季结束，美国经济已经开始降温。供应中断仍是美国经济放缓的主要原因，而德尔塔变种病毒现在是另一个拖累因素。目前的GDP预测仍将是1984年以来的最高水平。预计美联储将在12月开始缩减资产购买规模，并在2022年12月加息，随后分别在2023年和2024年加息两次。", msg.Post.Content)
		require.Equal(t, 0, len(msg.Post.ImageUrls))
		require.Equal(t, 0, len(msg.Post.FilesUrls))
		require.Equal(t, "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png", msg.Post.SubSource.AvatarUrl)
		require.Equal(t, "快讯", msg.Post.SubSource.Name)
		require.Equal(t, "a.com", msg.Post.OriginUrl)
		require.Equal(t, sourceId, msg.Post.SubSource.SourceId)

		tm, _ := time.Parse(Jin10DateFormat, "20210926-05:29:04")
		require.Equal(t, tm.Unix(), msg.Post.ContentGeneratedAt.AsTime().Unix())
	})
}

func TestJin10CrawlerWithImage(t *testing.T) {
	// var elem colly.HTMLElement
	var s = sink.NewStdErrSink()
	var builder CollectorBuilder
	crawler := builder.NewJin10Crawler(s)
	taskId := "task_1"
	sourceId := "a882eb0d-0bde-401a-b708-a7ce352b7392"

	task := GetFakeTask(taskId, sourceId, "要闻", protocol.PanopticSubSource_KEYNEWS)

	t.Run("[get message from dom element][keynews] html with image", func(t *testing.T) {
		htmlWithImage := `<div data-v-c7711130="" data-v-471802f2="" id="flash20210925215015057100" class="jin-flash-item-container is-normal"><!----><div data-v-c7711130="" class="jin-flash-item flash is-important"><div data-v-2b138c9f="" data-v-c7711130="" class="detail-btn"><div data-v-2b138c9f="" class="detail-btn_container"><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span></div></div><!----><div data-v-c7711130="" class="item-time">21:50:15</div><div data-v-c7711130="" class="item-right is-common"><div data-v-c7711130="" class="right-top"><!----><div data-v-c7711130="" class="right-content"><div data-v-c7711130="">孟晚舟乘坐的中国政府包机抵达深圳宝安机场。欢迎回家！（人民日报）</div></div><div data-v-c7711130="" class="right-pic img-intercept flash-pic"><div data-v-c7711130="" class="img-container show-shadow"><img data-v-c7711130="" data-src="https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite" src="https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite" lazy="loaded"></div></div><!----><!----></div></div><div data-v-47f123d2="" data-v-c7711130="" class="flash-item-share" style="display: none;"><a data-v-47f123d2="" href="javascript:void('openShare')" class="share-btn"><i data-v-47f123d2="" class="jin-icon iconfont icon-fenxiang"></i><span data-v-47f123d2="">分享</span></a><a data-v-47f123d2="" href="https://flash.jin10.com/detail/20210925215015057100" target="_blank"><i data-v-47f123d2="" class="jin-icon iconfont icon-xiangqing"></i><span data-v-47f123d2="">详情</span></a><a data-v-47f123d2="" href="javascript:void('copyFlash')"><i data-v-47f123d2="" class="jin-icon iconfont icon-fuzhi"></i><span data-v-47f123d2="">复制</span></a><!----></div></div></div>`
		elem := GetMockHtmlElem(htmlWithImage, "flash20210925215015057100")
		ctx := &working_context.CrawlerWorkingContext{SharedContext: working_context.SharedContext{Task: &task, IntentionallySkipped: false}, Element: elem, OriginUrl: "a.com"}
		err := crawler.GetMessage(ctx)
		msg := ctx.Result

		require.NoError(t, err)
		require.NotNil(t, msg)

		require.Equal(t, "7170aaae523ca3d0bc2b2b92bfead0d4", msg.Post.DeduplicateId)
		require.Equal(t, "孟晚舟乘坐的中国政府包机抵达深圳宝安机场。欢迎回家！（人民日报）", msg.Post.Content)
		require.Equal(t, 1, len(msg.Post.ImageUrls))
		require.Equal(t, "https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite", msg.Post.ImageUrls[0])
		require.Equal(t, 0, len(msg.Post.FilesUrls))
		require.Equal(t, "https://newsfeed-logo.s3.us-west-1.amazonaws.com/jin10.png", msg.Post.SubSource.AvatarUrl)
		require.Equal(t, "要闻", msg.Post.SubSource.Name)
		require.Equal(t, "a.com", msg.Post.OriginUrl)
		require.Equal(t, sourceId, msg.Post.SubSource.SourceId)

		tm, _ := time.Parse(Jin10DateFormat, "20210925-13:50:15")
		require.Equal(t, tm.Unix(), msg.Post.ContentGeneratedAt.AsTime().Unix())
	})
}

func TestJin10CrawlerNotMatchingRequest(t *testing.T) {
	// var elem colly.HTMLElement
	var s = sink.NewStdErrSink()
	var builder CollectorBuilder
	crawler := builder.NewJin10Crawler(s)
	taskId := "task_1"
	sourceId := "a882eb0d-0bde-401a-b708-a7ce352b7392"
	// Asking for Flash news
	task := GetFakeTask(taskId, sourceId, "快讯", protocol.PanopticSubSource_FLASHNEWS)
	t.Run("[get message from dom element][keynews] not request specified", func(t *testing.T) {
		// Got Key news
		htmlWithImage := `<div data-v-c7711130="" data-v-471802f2="" id="flash20210925215015057100" class="jin-flash-item-container is-normal"><!----><div data-v-c7711130="" class="jin-flash-item flash is-important"><div data-v-2b138c9f="" data-v-c7711130="" class="detail-btn"><div data-v-2b138c9f="" class="detail-btn_container"><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span><span data-v-2b138c9f=""></span></div></div><!----><div data-v-c7711130="" class="item-time">21:50:15</div><div data-v-c7711130="" class="item-right is-common"><div data-v-c7711130="" class="right-top"><!----><div data-v-c7711130="" class="right-content"><div data-v-c7711130="">孟晚舟乘坐的中国政府包机抵达深圳宝安机场。欢迎回家！（人民日报）</div></div><div data-v-c7711130="" class="right-pic img-intercept flash-pic"><div data-v-c7711130="" class="img-container show-shadow"><img data-v-c7711130="" data-src="https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite" src="https://flash-scdn.jin10.com/16f8ddbe-1b4b-4c1b-a3d1-844e466edb67.jpg/lite" lazy="loaded"></div></div><!----><!----></div></div><div data-v-47f123d2="" data-v-c7711130="" class="flash-item-share" style="display: none;"><a data-v-47f123d2="" href="javascript:void('openShare')" class="share-btn"><i data-v-47f123d2="" class="jin-icon iconfont icon-fenxiang"></i><span data-v-47f123d2="">分享</span></a><a data-v-47f123d2="" href="https://flash.jin10.com/detail/20210925215015057100" target="_blank"><i data-v-47f123d2="" class="jin-icon iconfont icon-xiangqing"></i><span data-v-47f123d2="">详情</span></a><a data-v-47f123d2="" href="javascript:void('copyFlash')"><i data-v-47f123d2="" class="jin-icon iconfont icon-fuzhi"></i><span data-v-47f123d2="">复制</span></a><!----></div></div></div>`
		elem := GetMockHtmlElem(htmlWithImage, "flash20210925215015057100")
		ctx := &working_context.CrawlerWorkingContext{SharedContext: working_context.SharedContext{Task: &task}, Element: elem, OriginUrl: "a.com"}
		err := crawler.GetMessage(ctx)
		require.NoError(t, err)
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
					SourceId:     "a882eb0d-0bde-401a-b708-a7ce352b7392",
					SubSources: []*protocol.PanopticSubSource{
						{
							Name:       "快讯",
							Type:       protocol.PanopticSubSource_FLASHNEWS,
							ExternalId: "1",
						},
					},
				},
				TaskMetadata: &protocol.TaskMetadata{ConfigName: "test_jin10_config_1"},
			},
			{
				TaskId:          "456",
				DataCollectorId: protocol.PanopticTask_COLLECTOR_JINSHI,
				TaskParams: &protocol.TaskParams{
					HeaderParams: []*protocol.KeyValuePair{},
					Cookies:      []*protocol.KeyValuePair{},
					SourceId:     "a882eb0d-0bde-401a-b708-a7ce352b7392",
					SubSources: []*protocol.PanopticSubSource{
						{
							Name:       "要闻",
							Type:       protocol.PanopticSubSource_KEYNEWS,
							ExternalId: "2",
						},
					},
					Params: &protocol.TaskParams_JinshiTaskParams{
						JinshiTaskParams: &protocol.JinshiTaskParams{
							SkipKeyWords: []string{"【黄金操作策略】"},
						},
					},
				},
				TaskMetadata: &protocol.TaskMetadata{
					ConfigName: "test_jin10_config_2",
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
	require.GreaterOrEqual(t, job.Tasks[0].TaskMetadata.TotalMessageFailed, int32(0))
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

func TestS3Store(t *testing.T) {
	s, err := file_store.NewS3FileStore(file_store.TestS3Bucket)
	require.NoError(t, err)

	s.SetCustomizeFileNameFunc(func(in, f string) string {
		return "test"
	})
	s.SetCustomizeFileExtFunc(func(in, f string) string {
		return ".jpg"
	})
	key, err := s.GenerateS3KeyFromUrl("https://tvax3.sinaimg.cn//crop.0.0.512.512.180//670a19b6ly8gm410azbeaj20e80e83yo.jpg", "")
	require.NoError(t, err)
	require.Equal(t, "test.jpg", key)
}

func TestLocalStore(t *testing.T) {
	fs, err := file_store.NewLocalFileStore("unit_test")
	require.NoError(t, err)

	fs.SetCustomizeFileNameFunc(func(in, f string) string {
		return "test"
	})
	fs.SetCustomizeFileExtFunc(func(in, f string) string {
		return ".jpg"
	})
	key, err := fs.GenerateFileNameFromUrl("https://tvax3.sinaimg.cn//crop.0.0.512.512.180//670a19b6ly8gm410azbeaj20e80e83yo.jpg", "")
	require.NoError(t, err)
	require.Equal(t, "test.jpg", key)
	_, err = fs.FetchAndStore("https://tvax3.sinaimg.cn//crop.0.0.512.512.180//670a19b6ly8gm410azbeaj20e80e83yo.jpg", "")
	require.NoError(t, err)
	require.FileExists(t, file_store.TmpFileDirPrefix+"unit_test/test.jpg")
	err = os.Remove(file_store.TmpFileDirPrefix + "unit_test/test.jpg")
	if err != nil {
		log.Fatal(err)
	}
	fs.CleanUp()
	require.NoDirExists(t, file_store.TmpFileDirPrefix+"unit_test")
}

func TestWeiboCollectorHandler(t *testing.T) {
	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{{
			TaskId:          "123",
			DataCollectorId: protocol.PanopticTask_COLLECTOR_WEIBO,
			TaskParams: &protocol.TaskParams{
				HeaderParams: []*protocol.KeyValuePair{},
				Cookies:      []*protocol.KeyValuePair{},
				SourceId:     "0129417c-4987-45c9-86ac-d6a5c89fb4f7",
				SubSources: []*protocol.PanopticSubSource{
					{
						Name:       "庄时利和",
						Type:       protocol.PanopticSubSource_USERS,
						ExternalId: "1728715190",
					},
					{
						Name:       "子陵在听歌",
						Type:       protocol.PanopticSubSource_USERS,
						ExternalId: "1251560221",
					},
					{
						Name:       "一水亦方",
						Type:       protocol.PanopticSubSource_USERS,
						ExternalId: "2349367491",
					},
				},
				Params: &protocol.TaskParams_WeiboTaskParams{
					WeiboTaskParams: &protocol.WeiboTaskParams{
						MaxPages: 2,
					},
				},
			},
			TaskMetadata: &protocol.TaskMetadata{
				ConfigName: "test_weibo_config",
			},
		},
		},
	}
	var handler DataCollectJobHandler
	err := handler.Collect(&job)
	fmt.Println("job", job.String())
	require.NoError(t, err)
	require.Equal(t, 1, len(job.Tasks))
	require.Equal(t, "123", job.Tasks[0].TaskId)
	require.Greater(t, job.Tasks[0].TaskMetadata.TotalMessageCollected, int32(0))
	require.GreaterOrEqual(t, job.Tasks[0].TaskMetadata.TotalMessageFailed, int32(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskStartTime.Seconds, int64(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskEndTime.Seconds, int64(0))
	require.Equal(t, protocol.TaskMetadata_STATE_SUCCESS, job.Tasks[0].TaskMetadata.ResultState)
}

func TestWallstreetArticleCollectorHandler(t *testing.T) {
	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{{
			TaskId:          "123",
			DataCollectorId: protocol.PanopticTask_COLLECTOR_WALLSTREET_ARTICLE,
			TaskParams: &protocol.TaskParams{
				HeaderParams: []*protocol.KeyValuePair{},
				Cookies:      []*protocol.KeyValuePair{},
				SourceId:     "66251821-ef9a-464c-bde9-8b2fd8ef2405",
				SubSources: []*protocol.PanopticSubSource{
					{
						Name:       "股市",
						Type:       protocol.PanopticSubSource_ARTICLE,
						ExternalId: "shares",
					},
					{
						Name:       "债市",
						Type:       protocol.PanopticSubSource_ARTICLE,
						ExternalId: "bonds",
					},
				},
			},
			TaskMetadata: &protocol.TaskMetadata{
				ConfigName: "test_wallstreet_config",
			},
		},
		},
	}
	var handler DataCollectJobHandler
	err := handler.Collect(&job)
	require.NoError(t, err)
	require.Equal(t, 1, len(job.Tasks))
	require.Equal(t, "123", job.Tasks[0].TaskId)
	require.Greater(t, job.Tasks[0].TaskMetadata.TotalMessageCollected, int32(0))
	require.GreaterOrEqual(t, job.Tasks[0].TaskMetadata.TotalMessageFailed, int32(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskStartTime.Seconds, int64(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskEndTime.Seconds, int64(0))
	require.Equal(t, protocol.TaskMetadata_STATE_SUCCESS, job.Tasks[0].TaskMetadata.ResultState)
}

func TestWallstreetNewsCollectorHandler(t *testing.T) {
	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{{
			TaskId:          "123",
			DataCollectorId: protocol.PanopticTask_COLLECTOR_WALLSTREET_NEWS,
			TaskParams: &protocol.TaskParams{
				HeaderParams: []*protocol.KeyValuePair{},
				Cookies:      []*protocol.KeyValuePair{},
				SourceId:     "66251821-ef9a-464c-bde9-8b2fd8ef2405",
				SubSources: []*protocol.PanopticSubSource{
					{
						Name:       "要闻",
						Type:       protocol.PanopticSubSource_KEYNEWS,
						ExternalId: "",
					},
					{
						Name:       "快讯",
						Type:       protocol.PanopticSubSource_FLASHNEWS,
						ExternalId: "",
					},
				},
				Params: &protocol.TaskParams_WallstreetNewsTaskParams{
					WallstreetNewsTaskParams: &protocol.WallstreetNewsTaskParams{
						Channels: []string{"a-stock-channel", "us-stock-channel", "hk-stock-channel", "goldc-channel%2Coil-channel%2Ccommodity-channel"},
						Limit:    3,
					},
				},
			},
			TaskMetadata: &protocol.TaskMetadata{
				ConfigName: "test_wallstreet_config",
			},
		},
		},
	}
	var handler DataCollectJobHandler
	err := handler.Collect(&job)
	fmt.Println("job", job.String())
	require.NoError(t, err)
	require.Equal(t, 1, len(job.Tasks))
	require.Equal(t, "123", job.Tasks[0].TaskId)
	require.Greater(t, job.Tasks[0].TaskMetadata.TotalMessageCollected, int32(0))
	require.GreaterOrEqual(t, job.Tasks[0].TaskMetadata.TotalMessageFailed, int32(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskStartTime.Seconds, int64(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskEndTime.Seconds, int64(0))
	require.Equal(t, protocol.TaskMetadata_STATE_SUCCESS, job.Tasks[0].TaskMetadata.ResultState)
}
func TestTimeUtil(t *testing.T) {
	timeStrInBeijingTime := "20060102-15:04:05"
	time, err := collector.ParseGenerateTime(timeStrInBeijingTime, Jin10DateFormat, ChinaTimeZone, "test")
	require.NoError(t, err)

	// equal to utc time 1136185445 == "Mon Jan 02 2006 07:04:05 GMT+0000"
	require.Equal(t, "seconds:1136185445", time.String())
}

func TestCaUsArticleCollectorHandler(t *testing.T) {
	job := protocol.PanopticJob{
		Tasks: []*protocol.PanopticTask{{
			TaskId:          "123",
			DataCollectorId: protocol.PanopticTask_COLLECTOR_CAUS_ARTICLE,
			TaskParams: &protocol.TaskParams{
				HeaderParams: []*protocol.KeyValuePair{
					{Key: "content-type", Value: "application/json;charset=UTF-8"},
					{Key: "user-agent", Value: "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36"},
					{Key: "uu_token", Value: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiIxNjIwMTgzNzczNjIzIiwiZXhwIjoxNjUxMjg3NzczfQ.09H378f2mfbQCpmnkTwFqhRnP9YHBymJxc9PGn9fZ8w"},
				},
				Cookies:  []*protocol.KeyValuePair{},
				SourceId: "1c6ab31c-aebe-40ba-833d-7cc2d977e5a1",
				SubSources: []*protocol.PanopticSubSource{
					{
						Name: "商业",
						Type: protocol.PanopticSubSource_ARTICLE,
					},
				},
			},
			TaskMetadata: &protocol.TaskMetadata{
				ConfigName: "test_caus_config",
			},
		},
		},
	}
	var handler DataCollectJobHandler
	err := handler.Collect(&job)
	fmt.Println("job", job.String())
	require.NoError(t, err)
	require.Equal(t, 1, len(job.Tasks))
	require.Equal(t, "123", job.Tasks[0].TaskId)
	require.Greater(t, job.Tasks[0].TaskMetadata.TotalMessageCollected, int32(0))
	require.GreaterOrEqual(t, job.Tasks[0].TaskMetadata.TotalMessageFailed, int32(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskStartTime.Seconds, int64(0))
	require.Greater(t, job.Tasks[0].TaskMetadata.TaskEndTime.Seconds, int64(0))
	require.Equal(t, protocol.TaskMetadata_STATE_SUCCESS, job.Tasks[0].TaskMetadata.ResultState)
}

type TestImageStore struct {
	file_store.S3FileStore
}

func (s *TestImageStore) FetchAndStore(url, fileName string) (string, error) {
	key, err := s.GenerateS3KeyFromUrl(url, fileName)
	fmt.Println(key)
	if err != nil {
		return "", err
	}
	return key, nil
}

func TestOffloadImageSourceFromHtml(t *testing.T) {
	var imageStore TestImageStore
	imageStore.SetCustomizeFileExtFunc(func(url string, fileName string) string {
		var re = regexp.MustCompile(`\%3D(.*)\%22`)
		found := re.FindStringSubmatch(url)
		return "." + found[0][3:len(found[0])-3]
	})
	originalHtml := `
	<![CDATA[ <section style="margin-right: 8px;margin-left: 8px;white-space: normal;" data-mpa-powered-by="yiban.io"><img referrerpolicy="no-referrer" data-cropselx1="0" data-cropselx2="578" data-cropsely1="0" data-cropsely2="233" data-fileid="506378232" data-ratio="0.4033333333333333" src="https://aaaaaaaabg%2F640%3Fwx_fmt%3Dpng%22" data-type="gif" data-w="600"> ]]> 
	`
	var re = regexp.MustCompile(`\<\!*\-*\[CDATA\[(.*)\]\]>`)
	final := re.ReplaceAllString(originalHtml, `$1`)
	res, err := OffloadImageSourceFromHtml(final, &imageStore)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res))
	require.NoError(t, err)
	require.Equal(t, "https://d20uffqoe1h0vv.cloudfront.net/6931eb26801b733de4fbc1b95043a26d.png", doc.Find("img").AttrOr("src", "notset"))
}
