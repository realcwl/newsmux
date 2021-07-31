package main

import (
	"log"

	"github.com/Luismorlan/newsmux/models"
	"github.com/Luismorlan/newsmux/server"
	"github.com/Luismorlan/newsmux/server/graphql"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/graph-gophers/graphql-go/relay"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
  "github.com/Luismorlan/newsmux/server"
	"github.com/gin-gonic/gin"
)

func main() {
  dsn := "host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=test_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&models.User{}, &models.Feed{})

	schemaString := graphql.GetGQLSchema()
  
	// Default With the Logger and Recovery middleware already attached
	router := gin.Default()

	router.POST("/graphql", &relay.Handler{
		Schema: utils.ParseGraphQLSchema(schemaString, &server.RootResolver{}),
	})
	router.Run(":8080")
}
