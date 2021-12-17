package bot

// This handler is to handle all user interactions from slack client(ex. click buttons in message blocks)
// https://api.slack.com/interactivity/handling

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Luismorlan/newsmux/model"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// slack go package has an outdated payload struct(action is a list, message is not a struct etc...), so we to redefine it
// gopkg: https://github.com/slack-go/slack/blob/4981f65787e6ea296375fe3dbcbb882c890ce66e/interactions.go#L34-L69
// real slack payload: https://api.slack.com/reference/interaction-payloads/block-actions#examples
type ActionText struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji"`
}

type Action struct {
	// We only care about the value of an action
	ActionId string     `json:"action_id"`
	Value    string     `json:"value"`
	Text     ActionText `json:"text"`
}

func (a Action) IsSubsribe() bool {
	return a.Text.Text == SubscribeButtonText
}

func (a Action) GetFeedId() string {
	return a.ActionId
}

func (a Action) GetFeedName() string {
	return a.Value
}

type SlackInteractionPayload struct {
	Type        slack.InteractionType `json:"type"`
	User        slack.User            `json:"user"`
	Channel     slack.Channel         `json:"channel"`
	ResponseURL string                `json:"response_url"`
	Container   slack.Container       `json:"container"`
	Actions     []Action              `json:"actions"`
}

func parseRequestToInteractionPayload(body io.ReadCloser) (*SlackInteractionPayload, error) {
	bodybytes, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	payload := SlackInteractionPayload{}
	const prefix = "payload="
	// https://api.slack.com/interactivity/handling#payloads
	// Slack sent this interaction post request in a weird format
	// Instead of a normal json body, they put "payload" param in request body
	// and encode the json with url escape characters
	if !strings.HasPrefix(string(bodybytes), prefix) {
		return nil, fmt.Errorf("unsupported request")
	}

	unescaped, err := url.QueryUnescape(string(bodybytes[len(prefix):]))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(unescaped), &payload)
	if err != nil {
		return nil, err
	}
	return &payload, nil
}

func InteractionHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		payload, err := parseRequestToInteractionPayload(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format"})
			return
		}

		// https://api.slack.com/interactivity/handling#acknowledgment_response
		// Slack ask the bot to acknowledge a valid interaction payload
		c.JSON(http.StatusOK, gin.H{"ok": true})

		if len(payload.Actions) == 0 {
			Logger.Log.Errorln("invalid payload without any action", payload)
		}

		action := payload.Actions[0]

		var channel model.Channel
		res := db.Where("channel_slack_id = ?", payload.Container.ChannelID).First(&channel)

		if res.RowsAffected == 0 {
			// This should never happen
			webhookMsg := &slack.WebhookMessage{
				Text: "The bot is not added to this channel yet, please add bot to this channel first: " + os.Getenv("BOT_ADDING_URL"),
			}
			slack.PostWebhook(payload.ResponseURL, webhookMsg)
			Logger.Log.Errorf("get interaction from a non-existing channel: %s", payload.Container.ChannelID)
			return
		}

		var responseText string
		if action.IsSubsribe() {
			db.Transaction(func(tx *gorm.DB) error {
				db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.ChannelFeedSubscription{
					ChannelID: channel.Id,
					FeedID:    action.GetFeedId(),
				})
				return nil
			})
			responseText = fmt.Sprintf("Successfully subscribed to %s", action.GetFeedName())
		} else {
			db.Where("feed_id = ?", action.GetFeedId()).Delete(&model.ChannelFeedSubscription{})
			responseText = fmt.Sprintf("%s is unsubscribed", action.GetFeedName())
		}

		// For testing purpose only
		// if action.IsSubsribe() {

		// var post *model.Post
		// db.Preload("SubSource").Preload("SharedFromPost").Preload("SharedFromPost.SubSource").Where("id=?", "c566d53c-a5df-4524-aae1-6e1f23d9aaa6").First(&post)
		// PushPostViaWebhook(*post, channel.WebhookUrl)

		// var post2 *model.Post
		// db.Preload("SubSource").Where("id=?", "8720596d-4962-47e0-8177-16189b19b329").First(&post2)
		// PushPostViaWebhook(*post2, channel.WebhookUrl)

		// var post3 *model.Post
		// db.Preload("SubSource").Where("id=?", "ffffe72e-935c-4c4a-a615-14e80ac71702").First(&post3)
		// PushPostViaWebhook(*post3, channel.WebhookUrl)

		// var post4 *model.Post
		// db.Preload("SubSource").Where("id=?", "ffd5df1c-2920-41db-a927-febae788c08b").First(&post4)
		// PushPostViaWebhook(*post4, channel.WebhookUrl)
		// }

		webhookMsg := &slack.WebhookMessage{
			Text: responseText,
		}

		slack.PostWebhook(channel.WebhookUrl, webhookMsg)

	}
}
