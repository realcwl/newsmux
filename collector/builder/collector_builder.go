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

func (CollectorBuilder) NewWeiboApiCollector(s CollectedDataSink, imageStore CollectedFileStore) ApiCollector {
	return &WeiboApiCollector{Sink: s, ImageStore: imageStore}
}

func (CollectorBuilder) NewZsxqApiCollector(s CollectedDataSink, imageStore CollectedFileStore, fileStore CollectedFileStore) ApiCollector {
	return &ZsxqApiCollector{Sink: s, ImageStore: imageStore, FileStore: fileStore}
}

func (CollectorBuilder) NewWallstreetNewsApiCollector(s CollectedDataSink) ApiCollector {
	return &WallstreetApiCollector{Sink: s}
}

// func (CollectorBuilder) NewKuailansiCrawler(s CollectedDataSink) RssCollector {
// 	return &SomeAPICollector{sink: s}
// }
func (CollectorBuilder) NewKuailansiApiCollector(s CollectedDataSink) DataCollector {
	return &KuailansiApiCrawler{Sink: s}
}
