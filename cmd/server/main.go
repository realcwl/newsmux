package main

import (
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Luismorlan/newsmux/server"
	"github.com/Luismorlan/newsmux/server/middlewares"
	"github.com/Luismorlan/newsmux/utils/dotenv"
	. "github.com/Luismorlan/newsmux/utils/flag"
	. "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	gintrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"
)

func init() {
	// Middlewares
	middlewares.Setup()

	Log.Info("api server initialized")
}

func cleanup() {
	Log.Info("api server shutdown")
}

func main() {
	ParseFlags()
	InitLogger()

	defer cleanup()

	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}

	// Default With the Logger and Recovery middleware already attached
	router := gin.Default()

	router.Use(cors.Default())
	router.Use(gintrace.Middleware(*ServiceName))
	if !*ByPassAuth {
		router.Use(middlewares.JWT())
	}

	handler := server.GraphqlHandler()
	router.POST("/api/graphql", handler)
	router.GET("/api/subscription", handler)

	router.GET("/api/healthcheck", server.HealthcheckHandler())

	// Setup graphql playground for debugging
	router.GET("/playground", func(c *gin.Context) {
		playground.Handler("GraphQL", "/api/graphql").ServeHTTP(c.Writer, c.Request)
	})
	router.GET("/playground/sub", func(c *gin.Context) {
		playground.Handler("Subscription", "/api/subscription").ServeHTTP(c.Writer, c.Request)
	})

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Newsfeed server - API not found"})
	})

	Log.Info("api server starts up")
	router.Run(":8080")
}
