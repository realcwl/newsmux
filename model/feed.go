package model

import (
	"time"

	"gorm.io/gorm"
)

/*

Feed is a data model for a column of newsfeed

Id: primary key, use to identify a feed
CreatedAt: time when entity is created
DeletedAt: time when entity is deleted
CreatorID:
Creator: user who added this source, "belongs-to" relation

Name: feed's display name (title)
Subscribers: all users who subscribed to this feed, "many-to-many" relation
Posts: all posts published to this feed, "many-to-many" relation
Sources: All sources this feed is listening to, "many-to-many" relationship.

*/
type Feed struct {
	Id          string `gorm:"primaryKey"`
	CreatedAt   time.Time
	DeletedAt   gorm.DeletedAt
	CreatorID   string `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Creator     User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name        string
	Subscribers []*User   `json:"subscribers" gorm:"many2many;"`
	Posts       []*Post   `json:"posts" gorm:"many2many;"`
	Sources     []*Source `json:"sources" gorm:"many2many;"`
}

func (Feed) IsFeedSeedStateInterface() {}
