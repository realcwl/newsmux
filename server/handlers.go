package server

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/server/resolver"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/gin-gonic/gin"
)

// GraphqlHandler is the universal handler for all GraphQL queries issued from
// client, by default it binds to a POST method.
func GraphqlHandler() gin.HandlerFunc {

	// TODO(jamie): check if env is dev or prod
	db, err := utils.GetDBProduction()
	if err != nil {
		// TODO(Jamie): check env and move to datadog if it is prod
		panic("failed to connect database")
	}

	utils.DatabaseSetupAndMigration(db)

	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolver.Resolver{
		DB:             db,
		SeedStateChans: resolver.NewSeedStateChannels(),
	}}))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
