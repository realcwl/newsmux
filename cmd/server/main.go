package main

import (
	"github.com/Luismorlan/newsmux/server"
	"github.com/gin-gonic/gin"
)

func main() {
	// Default With the Logger and Recovery middleware already attached
	router := gin.Default()

	router.POST("/graphql", server.GraphqlHandler())
	router.Run(":8080")
}
