package bot

// This handler is to handle all oauth requests when user added our bot to any channels
// https://api.slack.com/docs/slack-button

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/Luismorlan/newsmux/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
			"client_secret": {"1a9fcd3aa4b4949292b5c254174dd3fel"},
			"code":          {code},
			"redirect_uri":  {"https://alto.qingtan.ltd/auth"},
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
