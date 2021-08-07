package model

import (
	"time"

	"gorm.io/gorm"
)

/*

User is a data model for a newsfeed user

Id: primary key, use to identify a user
CreatedAt: time when entity is created
DeletedAt: time when entity is deleted

Name: name of a user, can be changed, don't need to be unique
AvartarUrl: User's icon URL.
SubscribedFeeds: feeds that this user subscribed, "many-to-many" relation
SavedPosts: posts that this user saved, "many-to-many" relation
SharedPosts: posts that this user shared, "many-to-many" relation

*/

type User struct {
	Id              string `gorm:"primaryKey"`
	CreatedAt       time.Time
	DeletedAt       gorm.DeletedAt
	Name            string
	AvartarUrl      string
	SubscribedFeeds []*Feed `json:"subscribed_feeds" gorm:"many2many;"`
	SavedPosts      []*Post `json:"saved_posts" gorm:"many2many;"`
}

func (User) IsUserSeedStateInterface() {}
