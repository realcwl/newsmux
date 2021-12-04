package model

import (
	"time"

	"gorm.io/gorm"
)

/*

Source is a data model for a news source

Example: twitter, weibo

Id: primary key, use to identify a source
CreatedAt: time when entity is created
DeletedAt: time when entity is deleted
CreatorID:
Creator: user who added this source, "belongs-to" relation, if user row is deleted, this will be set to null

Name: the display name of the source for example "twitter"
Domain: the domain of a source, for example "twitter.com"
SubSources: sub sources in this source, for example followed twitter users are subsource of "twitter", "has-many" relation
*/

type Source struct {
	Id         string `gorm:"primaryKey"`
	CreatedAt  time.Time
	DeletedAt  gorm.DeletedAt
	CreatorID  string `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Creator    User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name       string
	Domain     string
	SubSources []SubSource `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (Source) IsSourceSeedStateInterface() {}
