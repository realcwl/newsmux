package model

import (
	"time"
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
type Channel struct {
	Id             string    `gorm:"primaryKey"`
	CreatedAt      time.Time `gorm:"<-:create"`
	UpdatedAt      time.Time
	Name           string
	WebhookUrl     string
	ChannelSlackId string `gorm:"uniqueIndex"`

	// Team is not used in our current product
	// We save slack id and names here for future extensibility
	// We will move it to a separate model when we are more serious about it
	TeamSlackId     string
	TeamSlackName   string
	SubscribedFeeds []*Feed `json:"subscribed_feeds" gorm:"many2many;constraint:OnDelete:CASCADE;"`
}
