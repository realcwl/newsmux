package model

import (
	"time"

	"gorm.io/gorm"
)

/*

SubSource is a data model for a news sub-source

Example: twitter users, weibo users

Id: primary key, use to identify a sub-source
CreatedAt: time when entity is created
DeletedAt: time when entity is deleted
CreatorID:
Creator: user who added this source, "belongs-to" relation

Name: the display name of the source for example "twitter"
ExternalIdentifier: the id used in source website, for example, user id used by weibo internally.
SourceID: sub-source belong to this source, "has-many" relation, if source row is deleted, subsource is deleted

AvatarUrl: user profile image url
OriginUrl: link to user page
IsFromSharedPost: is the subsource from shared post, if so front end will ignore when display, and crawler won't crawl for it
*/
type SubSource struct {
	Id                      string `gorm:"primaryKey"`
	CreatedAt               time.Time
	DeletedAt               gorm.DeletedAt
	Name                    string
	ExternalIdentifier      string
	SourceID                string `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AvatarUrl               string
	OriginUrl               string
	Feeds                   []*Feed `json:"feeds" gorm:"many2many:feed_subsources;constraint:OnDelete:CASCADE;"`
	IsFromSharedPost        bool
	CustomizedCrawlerParams *string
}

func (SubSource) IsSubSourceSeedStateInterface() {}
