package main

import (
	"fmt"

	"github.com/Luismorlan/newsmux/collector"
	twitterscraper "github.com/n0madic/twitter-scraper"
)

func getAllTweets(name string) {
	tweets, _, _ := twitterscraper.New().FetchTweets(name, 20, "")
	fmt.Println(collector.PrettyPrint(tweets[2]))
	// for _, t := range tweets {
	// 	fmt.Println(collector.PrettyPrint(t))
	// }
}

func main() {
	name := "RnrCapital"
	getAllTweets(name)
}
