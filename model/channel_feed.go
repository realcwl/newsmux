package model

import (
	"time"

	"gorm.io/gorm"
)

/*

UserFeedSubscription is a "many-to-many" relation of user's subscription to a feed

UserID: user id
FeedID: feed id
CreatedAt: time when relation is created
DeletedAt: time when relation is deleted

*/

type ChannelFeedSubscription struct {
	ChannelID string `gorm:"primaryKey"`
	FeedID    string `gorm:"primaryKey"`
	CreatedAt time.Time
}

func (ChannelFeedSubscription) BeforeCreate(db *gorm.DB) error {
	return nil
}
