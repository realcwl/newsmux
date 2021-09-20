package main

import (
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

	Log.Info("crawler initialized")
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

	router.GET("/api/healthcheck", server.HealthcheckHandler())

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"message": "Newsfeed crawler - API not found"})
	})

	Log.Info("api server starts up")
	router.Run(":8080")
}
