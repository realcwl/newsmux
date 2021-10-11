package model

import (
	"time"

	"gorm.io/datatypes"
)

/*

Feed is a data model for a column of newsfeed

Id: primary key, use to identify a feed
CreatedAt: time when entity is created
UpdatedAt: time when Feed is updated. This timestamp is used to determine whether
this feed is unchanged.
CreatorID:
Creator: user who added this source, "belongs-to" relation

Name: feed's display name (title)
Subscribers: all users who subscribed to this feed, "many-to-many" relation
Posts: all posts published to this feed, "many-to-many" relation

-- Sources: All sources this feed is listening to, "many-to-many" relationship.
-- We don't keep sources, since we assume there is always a sub-source "default" for each source
-- For those sources without subsource like wall-street-news, we use the "default" subsource only

SubSources: All subsources this feed is listening to, "many-to-many" relationship.
	Do not only rely on subsource to infer source, so that we can have Feed only subscribe to source
*/
type Feed struct {
	Id                   string    `gorm:"primaryKey"`
	CreatedAt            time.Time `gorm:"<-:create"`
	UpdatedAt            time.Time
	CreatorID            string `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Creator              User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name                 string
	Subscribers          []*User      `json:"subscribers" gorm:"many2many;constraint:OnDelete:CASCADE;"`
	Posts                []*Post      `json:"posts" gorm:"many2many:post_feed_publishes;constraint:OnDelete:CASCADE;"`
	SubSources           []*SubSource `json:"subSources" gorm:"many2many:feed_subsources;constraint:OnDelete:CASCADE;"`
	Visibility           Visibility   `json:"visibility" gorm:"default:'PRIVATE';"`
	FilterDataExpression datatypes.JSON
}

func (Feed) IsFeedSeedStateInterface() {}
