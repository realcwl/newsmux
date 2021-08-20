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

*/
type SubSource struct {
	Id                 string `gorm:"primaryKey"`
	CreatedAt          time.Time
	DeletedAt          gorm.DeletedAt
	CreatorID          string `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Creator            User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name               string
	ExternalIdentifier string
	SourceID           string `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	IconUrl            string
	Feeds              []*Feed `json:"feeds" gorm:"many2many:feed_subsources;"`
}

func (SubSource) IsSubSourceSeedStateInterface() {}
