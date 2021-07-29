package server

import (
	"github.com/Luismorlan/newsmux/server/graphql"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/gin-gonic/gin"
	"github.com/graph-gophers/graphql-go/relay"
)

// GraphqlHandler is the universal handler for all GraphQL queries issued from
// client, by default it binds to a POST method.
func GraphqlHandler() gin.HandlerFunc {
	schemaString := graphql.GetGQLSchema()
	h := &relay.Handler{
		Schema: utils.ParseGraphQLSchema(schemaString, &RootResolver{}),
	}

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
