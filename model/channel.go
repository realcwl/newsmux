package model

import (
	"time"
)

/*

Channel is a data model for slack channels

Id: primary key, use to identify a channel
CreatedAt: time when entity is created
UpdatedAt: time when Feed is updated.
Name: the channel's slack name
WebhookUrl: for pushing messages to a channel
ChannelSlackId: this is the ID provided by Slack

TeamSlackId, TeamSlackName: the info of the slack team this channel is in
SubscribedFeeds: all feeds this channel subscribed to "many-to-many" relation
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
