package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/Luismorlan/newsmux/model"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SUBSCRIBE_BUTTON_TEXT   = "Subscribe"
	UNSUBSCRIBE_BUTTON_TEXT = "Unsubscribe"
)

type CommandForm struct {
	Command   string `form:"command" binding:"required"`
	ChannelId string `form:"channel_id" binding:"required"`
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
	return a.Text.Text == SUBSCRIBE_BUTTON_TEXT
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

func InteractionHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload := new(SlackInteractionPayload)
		bodybytes, _ := ioutil.ReadAll(c.Request.Body)

		// https://api.slack.com/interactivity/handling#payloads
		// Slack sent this interaction post request in a weird format
		// Instead of a normal json body, they put "payload" param in request body
		// and encode the json with url escape characters
		if string(bodybytes[0:8]) != "payload=" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown interaction"})
			return
		}

		unescaped, err := url.QueryUnescape(string(bodybytes[8:]))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format"})
			return
		}

		err = json.Unmarshal([]byte(unescaped), &payload)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format"})
			return
		}

		// https://api.slack.com/interactivity/handling#acknowledgment_response
		// Slack ask the bot to acknowledge a valid interaction payload
		c.JSON(http.StatusOK, gin.H{"ok": true})

		if len(payload.Actions) == 0 {
			Logger.Log.Errorf("invalid payload without any action", payload)
		}

		action := payload.Actions[0]

		var channel model.Channel
		res := db.Where("channel_slack_id = ?", payload.Container.ChannelID).First(&channel)

		if res.RowsAffected == 0 {
			// This should never happen
			Logger.Log.Errorf("get interaction from a non-existing channel: %s", payload.Container.ChannelID)
			return
		}

		if action.IsSubsribe() {
			db.Transaction(func(tx *gorm.DB) error {
				db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.ChannelFeedSubscription{
					ChannelID: channel.Id,
					FeedID:    action.GetFeedId(),
				})
				return nil
			})
		} else {
			db.Where("feed_id = ?", action.GetFeedId()).Delete(&model.ChannelFeedSubscription{})
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to save the channel to backend, contact tech please"})
			return
		}

		var responseText string
		if action.IsSubsribe() {
			responseText = fmt.Sprintf("Successfully subscribed to %s", action.GetFeedName())
		} else {
			responseText = fmt.Sprintf("%s is unsubscribed", action.GetFeedName())
		}

		webhookMsg := &slack.WebhookMessage{
			Text: responseText,
		}

		slack.PostWebhook(channel.WebhookUrl, webhookMsg)

		var post *model.Post
		db.Preload("SubSource").Preload("SharedFromPost").Preload("SharedFromPost.SubSource").Where("id=?", "c566d53c-a5df-4524-aae1-6e1f23d9aaa6").First(&post)
		PushPostViaWebhook(*post, channel.WebhookUrl)

		var post2 *model.Post
		db.Preload("SubSource").Where("id=?", "ef56625f-6433-45cb-a06a-f70c03bd1907").First(&post2)
		PushPostViaWebhook(*post2, channel.WebhookUrl)

		var post3 *model.Post
		db.Preload("SubSource").Where("id=?", "ffffe72e-935c-4c4a-a615-14e80ac71702").First(&post3)
		PushPostViaWebhook(*post3, channel.WebhookUrl)

		var post4 *model.Post
		db.Preload("SubSource").Where("id=?", "ffd5df1c-2920-41db-a927-febae788c08b").First(&post4)
		PushPostViaWebhook(*post4, channel.WebhookUrl)
	}
}

func BotCommandHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var form CommandForm
		c.Bind(&form)
		fmt.Println("form", form)
		switch form.Command {
		case "/feeds":
			// TODO(boning): not familiar with gorm syntax, will merge the two db query below to one
			var allFeeds []*model.Feed
			if err := db.Preload("Creator").Where("visibility = 'GLOBAL'").Find(&allFeeds).Order("subscribers desc").Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"text": "failed to get public feeds. please contact tech"})
				return
			}

			var subscribedFeeds []*model.Feed
			db.Model(&model.ChannelFeedSubscription{}).
				Select("feeds.id").
				Joins("INNER JOIN feeds ON feeds.id = channel_feed_subscriptions.feed_id").
				Joins("INNER JOIN channels ON channels.id = channel_feed_subscriptions.channel_id").
				Where("channels.channel_slack_id = ?", form.ChannelId).
				Find(&subscribedFeeds)

			subscribedFeedIds := map[string]struct{}{}
			for _, feed := range subscribedFeeds {
				subscribedFeedIds[feed.Id] = struct{}{}
			}

			var unsubscribedFeeds []*model.Feed
			subscribedFeeds = []*model.Feed{}

			for _, feed := range allFeeds {
				if _, ok := subscribedFeedIds[feed.Id]; ok {
					subscribedFeeds = append(subscribedFeeds, feed)
				} else {
					unsubscribedFeeds = append(unsubscribedFeeds, feed)
				}
			}

			/* Building message body
			examples can be found here https://github.com/slack-go/slack/tree/master/examples/blocks
			*/
			// subscribe section
			divSection := slack.NewDividerBlock()
			blocks := []slack.Block{divSection}

			for _, feed := range unsubscribedFeeds {
				subscribeBtnText := slack.NewTextBlockObject("plain_text", SUBSCRIBE_BUTTON_TEXT, false, false)
				subscribeBtnEle := slack.NewButtonBlockElement(feed.Id, feed.Name, subscribeBtnText)
				optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t _%s_", feed.Name, feed.Creator.Name), false, false)
				optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
				blocks = append(blocks, optionSection)
			}

			blocks = append(blocks, divSection)

			for _, feed := range subscribedFeeds {
				subscribeBtnText := slack.NewTextBlockObject("plain_text", UNSUBSCRIBE_BUTTON_TEXT, false, false)
				subscribeBtnEle := slack.NewButtonBlockElement(feed.Id, feed.Name, subscribeBtnText)
				optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t _%s_", feed.Name, feed.Creator.Name), false, false)
				optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
				blocks = append(blocks, optionSection)
			}

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
