package collector_builder

import (
	. "github.com/Luismorlan/newsmux/collector"
	"github.com/Luismorlan/newsmux/collector/file_store"
	. "github.com/Luismorlan/newsmux/collector/instances"
	"github.com/Luismorlan/newsmux/collector/sink"
)

type CollectorBuilder struct{}

// Crawler Collectors
func (CollectorBuilder) NewJin10Crawler(s sink.CollectedDataSink) CrawlerCollector {
	return &Jin10Crawler{Sink: s}
}

func (CollectorBuilder) NewWeiboApiCollector(s sink.CollectedDataSink, store file_store.CollectedFileStore) ApiCollector {
	return &WeiboApiCollector{Sink: s, ImageStore: store}
}

func (CollectorBuilder) NewWallstreetNewsApiCollector(s sink.CollectedDataSink) ApiCollector {
	return &WallstreetApiCollector{Sink: s}
}

// func (CollectorBuilder) NewKuailansiCrawler(s CollectedDataSink) RssCollector {
// 	return &SomeAPICollector{sink: s}
// }
func (CollectorBuilder) NewKuailansiApiCollector(s sink.CollectedDataSink) DataCollector {
	return &KuailansiApiCrawler{Sink: s}
}
