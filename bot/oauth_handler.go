package bot

// This handler is to handle all oauth requests when user added our bot to any channels
// https://api.slack.com/docs/slack-button

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"

	"github.com/Luismorlan/newsmux/model"
	Logger "github.com/Luismorlan/newsmux/utils/log"
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
			Logger.Log.Error("got an oauth request without code")
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}
		data := url.Values{
			"client_id":     {os.Getenv("BOT_CLIENT_ID")},
			"client_secret": {os.Getenv("BOT_CLIENT_SECRET")},
			"code":          {code},
			"redirect_uri":  {os.Getenv("BOT_REDIRECT_URL")},
		}

		resp, err := http.PostForm("https://slack.com/api/oauth.v2.access", data)
		if err != nil {
			Logger.Log.Error("got invalid oauth code", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is invalid"})
			return
		}

		defer resp.Body.Close()
		slackResp := new(SlackOAuthResponse)
		json.NewDecoder(resp.Body).Decode(&slackResp)

		if !slackResp.Ok {
			Logger.Log.Error("failed to fetch channel info from slack", slackResp)
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch the channel info from slack, please contact the tech team"})
			return
		}

		Logger.Log.Info("Bot added to a channel", slackResp)

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
			Logger.Log.Error("failed to save the channel to backend, contact tech please", err)
			return
		}

		c.Data(200, "application/json; charset=utf-8", []byte("Bot is successfully added. check your slack channel now!"))
	}
}
