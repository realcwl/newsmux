package main

import (
	"net/http"

	Twitter "github.com/Luismorlan/newsmux/collector/webhook/twitter"
	Flag "github.com/Luismorlan/newsmux/utils/flag"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gin-gonic/gin"
)

func main() {
	Flag.ParseFlags()
	Logger.InitLogger()

	router := gin.Default()

	// Add a debug route for testing and health check
	router.GET("/webhook/ping", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, "pong")
	})

	AddTwitterWebhook(router.Group("/webhook"))
	// Additional webhooks should be added below this line

	Logger.Log.Info("===== Webhook Server Started =====")
	router.Run(":7070")
}

func AddTwitterWebhook(rg *gin.RouterGroup) {
	twitter := rg.Group("/twitter")

	twitter.GET("/", Twitter.HandleTwitterCRC)
	twitter.POST("/", Twitter.HandleTwitterMessage)
}
