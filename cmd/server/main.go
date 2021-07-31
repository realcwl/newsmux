package main

import (
	"fmt"
	"net/http"

	"github.com/Luismorlan/newsmux/models"
	"github.com/Luismorlan/newsmux/server"
	"github.com/Luismorlan/newsmux/server/middlewares"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	middlewares.Setup()
}

func main() {
	dsn := "host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=test_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.User{}, &models.Feed{})

	// Default With the Logger and Recovery middleware already attached
	router := gin.Default()

	router.Use(middlewares.JWT())

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
