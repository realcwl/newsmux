package server

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/server/resolver"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// HealthCheckHandler returns 200 whenever server is up
func HealthcheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	}
}

// GraphqlHandler is the universal handler for all GraphQL queries issued from
// client, by default it binds to a POST method.
func GraphqlHandler() gin.HandlerFunc {
	// TODO(jamie): check if env is dev or prod
	db, err := utils.GetDBConnection()
	if err != nil {
		// TODO(Jamie): check env and move to datadog if it is prod
		panic("failed to connect database")
	}

	utils.DatabaseSetupAndMigration(db)

	h := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &resolver.Resolver{
		DB:             db,
		SeedStateChans: resolver.NewSeedStateChannels(),
		SignalChans:    resolver.NewSignalChannels(),
	}}))

	h.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO(chenweilunster): Perform a fine-grain check over CORS
				return true
			},
		},
	})
	h.AddTransport(transport.GET{})
	h.AddTransport(transport.POST{})

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
