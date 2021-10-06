package collector

type CollectorBuilder struct{}

// Crawler Collectors
func (CollectorBuilder) NewJin10Crawler(s CollectedDataSink) CrawlerCollector {
	return &Jin10Crawler{sink: s}
}

// // API Collectors
// func (CollectorBuilder) NewKuailansiCrawler(s CollectedDataSink) ApiCollector {
// 	return &SomeAPICollector{sink: s}
// }

// // RSS Collectors
// func (CollectorBuilder) NewXXXXCrawler(s CollectedDataSink) RssCollector {
// 	return &SomeRssCollector{sink: s}
// }
