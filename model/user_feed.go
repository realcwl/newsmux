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

OrderInPanel: what is the order of this feed in user's panel, from left to right marked as 1,2,3...

*/

type UserFeedSubscription struct {
	UserID       string `gorm:"primaryKey"`
	FeedID       string `gorm:"primaryKey"`
	CreatedAt    time.Time
	DeletedAt    gorm.DeletedAt
	OrderInPanel int
}

func (UserFeedSubscription) BeforeCreate(db *gorm.DB) error {
	return nil
}
