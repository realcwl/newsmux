package main

import (
	"fmt"
	"net/http"

	"github.com/Luismorlan/newsmux/server"
	"github.com/Luismorlan/newsmux/server/middlewares"
	"github.com/gin-gonic/gin"
)

func init() {
	middlewares.Setup()
}

func main() {
	// Default With the Logger and Recovery middleware already attached
	router := gin.Default()

	// TODO: remove once we fiture out how to test with jwt turned on
	// router.Use(middlewares.JWT())
	router.Use(middlewares.CorsWhitelist([]string{"http://localhost:3000"}))

	router.POST("/graphql", server.GraphqlHandler())

	// TODO(chenweilunster): Keep this for now for fast debug. Remove this debug
	// route once the application is fully implemented.
	router.GET("/ping", func(c *gin.Context) {
		fmt.Println(c.Request.Header.Get("sub"))
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	router.Run(":8080")
}
