package collector

type CollectorBuilder struct{}

// Crawler Collectors
func (CollectorBuilder) NewJin10Crawler(s CollectedDataSink) CrawlerCollector {
	return &Jin10Crawler{sink: s}
}

func (CollectorBuilder) NewWeiboApiCollector(s CollectedDataSink, store CollectedFileStore) ApiCollector {
	return &WeiboApiCollector{sink: s, imageStore: store}
}

// func (CollectorBuilder) NewKuailansiCrawler(s CollectedDataSink) RssCollector {
// 	return &SomeAPICollector{sink: s}
// }
