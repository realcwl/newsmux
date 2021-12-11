package main

import (
	Twitter "github.com/Luismorlan/newsmux/collector/webhook/twitter"
	"github.com/gin-gonic/gin"
)

func AddTwitterWebhook(rg *gin.RouterGroup) {
	twitter := rg.Group("/twitter")

	twitter.GET("/", Twitter.HandleTwitterCRC)
	twitter.POST("/", Twitter.HandleTwitterMessage)
}
