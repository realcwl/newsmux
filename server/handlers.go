package server

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/server/resolver"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// GraphqlHandler is the universal handler for all GraphQL queries issued from
// client, by default it binds to a POST method.
func GraphqlHandler() gin.HandlerFunc {
	dsn := "host=newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com user=root password=b5OKda1Twb1r dbname=test_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.Debug().AutoMigrate(&model.Feed{}, &model.User{})

	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolver.Resolver{
		DB: db,
	}}))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
