package collector_builder

import (
	. "github.com/Luismorlan/newsmux/collector"
	. "github.com/Luismorlan/newsmux/collector/instances"
)

type CollectorBuilder struct{}

// Crawler Collectors
func (CollectorBuilder) NewJin10Crawler(s CollectedDataSink) CrawlerCollector {
	return &Jin10Crawler{Sink: s}
}

func (CollectorBuilder) NewWeiboApiCollector(s CollectedDataSink, store CollectedFileStore) ApiCollector {
	return &WeiboApiCollector{Sink: s, ImageStore: store}
}

// func (CollectorBuilder) NewKuailansiCrawler(s CollectedDataSink) RssCollector {
// 	return &SomeAPICollector{sink: s}
// }
