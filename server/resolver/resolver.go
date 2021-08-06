package resolver

import (
	"gorm.io/gorm"
)

//go:generate go run github.com/99designs/gqlgen

// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB             *gorm.DB
	SeedStateChans *SeedStateChannels
}
