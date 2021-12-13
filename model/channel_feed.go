package model

import (
	"time"

	"gorm.io/gorm"
)

/*

ChannelFeedSubscription is a "many-to-many" relation of channel's subscription to a feed

ChannelId: channel id
FeedID: feed id
CreatedAt: time when relation is created

*/

type ChannelFeedSubscription struct {
	ChannelID string `gorm:"primaryKey"`
	FeedID    string `gorm:"primaryKey"`
	CreatedAt time.Time
}

func (ChannelFeedSubscription) BeforeCreate(db *gorm.DB) error {
	return nil
}
