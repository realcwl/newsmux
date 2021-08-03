package main

import (
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Luismorlan/newsmux/server"
	"github.com/Luismorlan/newsmux/server/middlewares"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	gintrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"
)

func init() {
	// Middlewares
	middlewares.Setup()

	utils.Logger.WithFields(
		logrus.Fields{"service": utils.ServiceName, "is_development": utils.IsDevelopment},
	).Info("api server initialized")
}

func cleanup() {
	utils.CloseProfiler()
	utils.CloseTracer()

	utils.Logger.WithFields(
		logrus.Fields{"service": utils.ServiceName, "is_development": utils.IsDevelopment},
	).Info("api server shutdown")
}

func main() {
	defer cleanup()

	// Default With the Logger and Recovery middleware already attached
	router := gin.Default()

	router.Use(gintrace.Middleware(utils.ServiceName))

	// TODO: remove once we fiture out how to test with jwt turned on
	// router.Use(middlewares.JWT())
	router.Use(middlewares.CorsWhitelist([]string{"http://localhost:3000"}))

	router.POST("/graphql", server.GraphqlHandler())
	// Setup graphql playground for debugging
	router.GET("/", func(c *gin.Context) {
		playground.Handler("GraphQL", "/graphql").ServeHTTP(c.Writer, c.Request)
	})
	// TODO(chenweilunster): Keep this for now for fast debug. Remove this debug
	// route once the application is fully implemented.
	router.GET("/ping", func(c *gin.Context) {
		fmt.Println(c.Request.Header.Get("sub"))
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	utils.Logger.WithFields(
		logrus.Fields{"service": utils.ServiceName, "is_development": utils.IsDevelopment},
	).Info("api server starts up")

	router.Run(":8080")
}
