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

*/

type UserFeedSubscription struct {
	UserID    string `gorm:"primaryKey"`
	FeedID    string `gorm:"primaryKey"`
	CreatedAt time.Time

	// order of this feed in user's panel, from left to right marked as 0,1,2,3...
	OrderInPanel int `gorm:"default:0"`
}

func (UserFeedSubscription) BeforeCreate(db *gorm.DB) error {
	return nil
}
