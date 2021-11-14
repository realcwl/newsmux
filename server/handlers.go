package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/Luismorlan/newsmux/model"
	"github.com/Luismorlan/newsmux/server/graph/generated"
	"github.com/Luismorlan/newsmux/server/resolver"
	"github.com/Luismorlan/newsmux/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
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

type SlackTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SlackIncomingWebhook struct {
	Channel   string `json:"channel"`
	ChannelId string `json:"channel_id"`
	Url       string `json:"url"`
}

type SlackOAuthResponse struct {
	Ok              bool                 `json:"ok"`
	AppId           string               `json:"app_id"`
	IncomingWebhook SlackIncomingWebhook `json:"incoming_webhook"`
	Team            SlackTeam            `json:"team"`
}

func AuthHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code, ok := c.GetQuery("code")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}
		data := url.Values{
			"client_id":     {"2628263675187.2627083261045"},
			"client_secret": {"1a9fcd3aa4b4949292b5c254174dd3fe"},
			"code":          {code},
			"redirect_uri":  {"http://localhost:8080/auth"},
		}

		resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", data)
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is invalid"})
			return
		}

		defer resp.Body.Close()
		slackResp := new(SlackOAuthResponse)
		json.NewDecoder(resp.Body).Decode(&slackResp)

		if !slackResp.Ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch the channel info from slack, please contact the tech team"})
			return
		}

		if slackResp.IncomingWebhook.Channel[0] != '#' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "the bot has to be added to a channel(not a user) currently, please readded it"})
			return
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "channel_slack_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"name", "webhook_url", "updated_at"}),
			}).Create(&model.Channel{
				Id:             uuid.New().String(),
				Name:           slackResp.IncomingWebhook.Channel,
				ChannelSlackId: slackResp.IncomingWebhook.ChannelId,
				TeamSlackId:    slackResp.Team.ID,
				TeamSlackName:  slackResp.Team.Name,
				WebhookUrl:     slackResp.IncomingWebhook.Url,
			})
			return nil
		})

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to save the channel to backend, contact tech please"})
			return
		}

		c.Data(200, "application/json; charset=utf-8", []byte("Bot is successfully added. check your slack channel now!"))
	}
}

func SubscriberHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.GetQuery("channelId")
	}
}

func BotCommandHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var form CommandForm
		c.Bind(&form)
		switch form.Command {
		case "/feeds":
			var feeds []*model.Feed
			if err := db.Preload("Creator").Preload("SubSources").Where("visibility = 'GLOBAL'").Find(&feeds).Order("subscribers desc").Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"text": "failed to get public feeds. please contact tech"})
				return
			}
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
	db, err := utils.GetDBConnection()
	if err != nil {
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
