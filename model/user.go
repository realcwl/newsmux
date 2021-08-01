package model

import "time"

type User struct {
	Id              string `gorm:"primaryKey"`
	CreatedAt       time.Time
	DeletedAt       time.Time
	Name            string
	Age             int
	SubscribedFeeds []*Feed `json:"subscribed_feeds" gorm:"many2many:user_feed_subscription;"`
}
