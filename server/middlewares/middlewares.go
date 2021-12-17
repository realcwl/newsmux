package middlewares

import (
	"context"
	"log"
	"net/http"

	"github.com/Luismorlan/newsmux/utils"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/gin-gonic/gin"
)

var (
	// cognitoClient is a thread safe client that performs user authorization
	// based on jwt token. Before using this client, make sure it's initialized
	// correctly.
	cognitoClient *cognitoidentityprovider.Client
)

// Setup initialized all package scoped variables that are needed to perform
// middleware functionalities, such as Cognito client. This function must be
// called before any middleware is used.
func Setup() {
	client, err := createCognitoClient()
	if err != nil {
		// Abort directly if the Cognito isn't setup successfully, which is crucial
		// for server side authorization.
		// TODO(chenweilunster): migrate this to Datadog once cloud logging has been
		// setup.
		log.Fatalf("fail to setup Cognito client: %s", err.Error())
	}
	setCognitoClient(client)
}

// createCognitoClient creates a default client with aws config located in path
// ~/.aws/config, and return error on error.
func createCognitoClient() (*cognitoidentityprovider.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return cognitoidentityprovider.NewFromConfig(cfg), nil
}

func setCognitoClient(client *cognitoidentityprovider.Client) {
	cognitoClient = client
}

// JWT middleware fetch user jwt in the http header, looking for field "token".
// It then parse the JWT and add a new field "sub" stores user's id. It returns
// error on token not provided or token is invalid (wrong token or expired).
func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		jwt := c.Query("token")

		if jwt == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": utils.ErrorTokenAuthFail,
				"msg":  "empty jwt token",
			})
			c.Abort()
			return
		}

		user, err := cognitoClient.GetUser(context.TODO(), &cognitoidentityprovider.GetUserInput{AccessToken: &jwt})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": utils.ErrorTokenAuthFail,
				"msg":  err.Error(),
			})
			c.Abort()
			return
		}

		// Successfully validated the jwt token, replace the header field "token"
		// with the user's sub (id).
		c.Request.Header.Del("token")
		c.Request.Header.Add("sub", *user.Username)

		// before request
		c.Next()
	}
}
