package model

import "gorm.io/gorm"

// Post is a piece of news
// PostId: a uuid generated to identify a post internally
// GeneratedTS: unix time stamp when the post is generated,
//   either from user creation, crawler or other channel
// Title: post's title in plain text
// Content: post's content in plain text
// Source: source website for example "twitter", "weibo", "Caixin"
// SubSource: for example a twitter user, weibo user, sub channel in Caixin etc.,
type Post struct {
	gorm.Model
	Title     string `json:"title"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	SubSource string `json:"sub_source"`
}
