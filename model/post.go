package model

import (
	"time"

	"gorm.io/gorm"
)

/*

Post is a piece of news crawler fetched

Id: primary key, use to identify a sub-source
CreatedAt: time when entity is created
DeletedAt: time when entity is deleted

Title: post's title in plain text
Content: post's content in plain text
SourceID:
Source: source website for example "twitter", "weibo", "Caixin",  "belongs-to" relation
SubSourceID:
SubSource: for example a twitter user, weibo user, sub channel in Caixin etc., "belongs-to" relation

SharedFromPostID:
SharedFromPost:
		if the post is a user shared(like re-twitt) post, set this as the Post originally shared. Support multi-level sharing.
		also if the post is shared:
			Source is a FAKE one representing "shared".
				Name is "分享"
			SubSource representing "user".
				CreatorID is user_id
				Creator is User
				Name is user's name
				ExternalIdentifier is empty


SavedByUser: mark when user save the post, "many-to-many" relation
PublishedFeeds: feeds that this post published to, "many-to-many" relation

Cursor: The auto-inc global-unique index to keep the relative order of posts
*/

type Post struct {
	Id               string `gorm:"primaryKey"`
	CreatedAt        time.Time
	DeletedAt        gorm.DeletedAt
	Title            string
	Content          string
	SourceID         string     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Source           Source     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	SubSourceID      *string    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	SubSource        *SubSource `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	SharedFromPostID *string
	SharedFromPost   *Post
	SavedByUser      []*User `json:"saved_by_user" gorm:"many2many;"`
	PublishedFeeds   []*Feed `json:"published_feeds" gorm:"many2many;"`
	Cursor           int32   `gorm:"autoIncrement"`
	OriginUrl        string
}
