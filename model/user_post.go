package model

import (
	"time"

	"gorm.io/gorm"
)

/*

UserPostSave is a "many-to-many" relation of user save a post

UserID: user id
PostID: post id
CreatedAt: time when relation is created
DeletedAt: time when relation is deleted

*/

type UserPostSave struct {
	UserID    string `gorm:"primaryKey"`
	PostID    string `gorm:"primaryKey"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func (UserPostSave) BeforeCreate(db *gorm.DB) error {
	return nil
}
