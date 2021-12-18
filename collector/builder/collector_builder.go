package collector_builder

import (
	. "github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/file_store"
	. "github.com/Luismorlan/newsmux/collector/instances"
	"github.com/Luismorlan/newsmux/collector/sink"
	twitterscraper "github.com/n0madic/twitter-scraper"
)

type CollectorBuilder struct{}

func (CollectorBuilder) NewCaUsArticleCrawlerCollector(s sink.CollectedDataSink) CrawlerCollector {
	return &CaUsArticleCrawler{Sink: s}
}

// Crawler Collectors
func (CollectorBuilder) NewJin10Crawler(s sink.CollectedDataSink) CrawlerCollector {
	return &Jin10Crawler{Sink: s}
}

func (CollectorBuilder) NewZsxqApiCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore, fileStore file_store.CollectedFileStore) ApiCollector {
	return &ZsxqApiCollector{Sink: s, ImageStore: imageStore, FileStore: fileStore}
}

func (CollectorBuilder) NewWeiboApiCollector(s sink.CollectedDataSink, store file_store.CollectedFileStore) ApiCollector {
	return &WeiboApiCollector{Sink: s, ImageStore: store}
}

func (CollectorBuilder) NewWallstreetNewsApiCollector(s sink.CollectedDataSink) ApiCollector {
	return &WallstreetApiCollector{Sink: s}
}

func (CollectorBuilder) NewKuailansiApiCollector(s sink.CollectedDataSink) DataCollector {
	return &KuailansiApiCrawler{Sink: s}
}

func (CollectorBuilder) NewJinseApiCollector(s sink.CollectedDataSink) DataCollector {
	return &JinseApiCrawler{Sink: s}
}

func (CollectorBuilder) NewWeixinRssCollector(s sink.CollectedDataSink, imageStore file_store.CollectedFileStore) DataCollector {
	return &WeixinArticleRssCollector{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewWisburgCrawler(s sink.CollectedDataSink) DataCollector {
	return &WisburgCrawler{Sink: s}
}

func (CollectorBuilder) NewKe36ApiCollector(s sink.CollectedDataSink) DataCollector {
	return &Kr36ApiCollector{Sink: s}
}

func (CollectorBuilder) NewWallstreetNewsArticleCollector(s sink.CollectedDataSink) DataCollector {
	return &WallstreetArticleCollector{Sink: s}
}

func (CollectorBuilder) NewCaUsNewsCrawlerCollector(s sink.CollectedDataSink) DataCollector {
	return &CaUsNewsCrawler{Sink: s}
}

func (CollectorBuilder) NewCaixinCrawler(s sink.CollectedDataSink) DataCollector {
	return &CaixinCollector{Sink: s}
}

func (CollectorBuilder) NewGelonghuiCrawler(s sink.CollectedDataSink) DataCollector {
	return &GelonghuiCrawler{Sink: s}
}

func (CollectorBuilder) NewClsNewsCrawlerCollector(s sink.CollectedDataSink) DataCollector {
	return &ClsNewsCrawler{Sink: s}
}

func (CollectorBuilder) NewCustomizedSourceCrawlerCollector(s sink.CollectedDataSink) DataCollector {
	return &CustomizedSourceCrawler{Sink: s}
}

func (CollectorBuilder) NewCustomizedSubSourceCollector(s sink.CollectedDataSink) DataCollector {
	return &CustomizedSubSourceCrawler{Sink: s}
}

func (CollectorBuilder) NewTwitterCollector(s sink.CollectedDataSink) DataCollector {
	return &TwitterApiCrawler{Sink: s, Scraper: twitterscraper.New()}
}
