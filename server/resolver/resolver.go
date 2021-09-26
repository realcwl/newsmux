package resolver

import (
	"gorm.io/gorm"
)

const (
	DefaultSubSourceName = "default"
)

// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB             *gorm.DB
	SeedStateChans *SeedStateChannels
}
