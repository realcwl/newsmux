package main

import (
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Luismorlan/newsmux/server"
	"github.com/Luismorlan/newsmux/server/middlewares"
	. "github.com/Luismorlan/newsmux/utils"
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
	CloseProfiler()
	CloseTracer()
	Log.Info("api server shutdown")
}

func main() {
	defer cleanup()

	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}

	// Default With the Logger and Recovery middleware already attached
	router := gin.Default()

	router.Use(cors.Default())
	router.Use(gintrace.Middleware(ServiceName))
	if !ByPassAuth {
		router.Use(middlewares.JWT())
	}

	handler := server.GraphqlHandler()
	router.POST("/graphql", handler)
	router.GET("/subscription", handler)

	// Setup graphql playground for debugging
	router.GET("/", func(c *gin.Context) {
		playground.Handler("GraphQL", "/graphql").ServeHTTP(c.Writer, c.Request)
	})
	router.GET("/sub", func(c *gin.Context) {
		playground.Handler("Subscription", "/subscription").ServeHTTP(c.Writer, c.Request)
	})

	// TODO(chenweilunster): Keep this for now for fast debug. Remove this debug
	// route once the application is fully implemented.
	router.GET("/ping", func(c *gin.Context) {
		fmt.Println(c.Request.Header.Get("sub"))
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	Log.Info("api server starts up")
	router.Run(":8080")
}
