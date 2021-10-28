package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/server/resolver"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/slack-go/slack"
	"gorm.io/gorm/clause"
)

// HealthCheckHandler returns 200 whenever server is up
func HealthcheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	}
}

type CommandForm struct {
	Command string `form:"command" binding:"required"`
}

func BotCommandHandler() gin.HandlerFunc {
	db, err := utils.GetDBConnection()
	if err != nil {
		panic("failed to connect to database")
	}

	utils.DatabaseSetupAndMigration(db)

	return func(c *gin.Context) {
		var form CommandForm
		c.Bind(&form)
		switch form.Command {
		case "/feeds":
			var feeds []*model.Feed
			if err := db.Preload(clause.Associations).Where("visibility = 'GLOBAL'").Find(&feeds).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"text": "failed to get public feeds. please contact tech"})
			}
			sort.Slice(feeds, func(i, j int) bool {
				return len(feeds[i].Subscribers) > len(feeds[j].Subscribers)
			})

			/* Building message body */
			// subscribe section
			divSection := slack.NewDividerBlock()
			blocks := []slack.Block{divSection}

			subscribeBtnText := slack.NewTextBlockObject("plain_text", "Subscribe", false, false)
			subscribeBtnEle := slack.NewButtonBlockElement("", "click_me_123", subscribeBtnText)
			for i := 0; i < len(feeds); i++ {
				optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t _%s_ \t `%d`", feeds[i].Name, feeds[i].Creator.Name, len(feeds[i].Subscribers)), false, false)
				optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
				blocks = append(blocks, optionSection)
			}

			unsubscribeBtnText := slack.NewTextBlockObject("plain_text", "Unsubscribe", false, false)
			unsubscribeBtnEle := slack.NewButtonBlockElement("", "click_me_123", unsubscribeBtnText)

			// unsubscribe section
			blocks = append(blocks, divSection)
			// TODO(boning): this is a demo for unsubscribe, will delete it when we have real channel feed table
			optionSample := slack.NewTextBlockObject("mrkdwn", "*恒大足球* \t _Jamie_ \t `5`", false, false)
			optionSection := slack.NewSectionBlock(optionSample, nil, slack.NewAccessory(unsubscribeBtnEle))

			blocks = append(blocks, optionSection)

			msg := slack.NewBlockMessage(blocks...)
			b, err := json.MarshalIndent(msg, "", "    ")
			if err != nil {
				fmt.Println(err)
				return
			}

			c.Data(http.StatusOK, "application/json", b)

		default:
			c.JSON(http.StatusNotFound, gin.H{
				"response_type": "ephemeral",
				"text":          "Sorry, slash commando, that's an unknown command",
			})
		}
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
		DB:          db,
		SignalChans: resolver.NewSignalChannels(),
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
