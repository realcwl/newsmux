package resolver

import (
	"github.com/Luismorlan/newsmux/utils"
	"gorm.io/gorm"
)

const (
	DefaultSubSourceName = "default"
)

// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB               *gorm.DB
	RedisStatusStore *utils.RedisStatusStore
	SignalChans      *SignalChannels
}
