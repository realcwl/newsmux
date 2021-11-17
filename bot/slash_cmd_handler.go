package bot

// This handler is for slack slash commands
// https://api.slack.com/interactivity/slash-commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/Luismorlan/newsmux/model"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

const (
	SUBSCRIBE_BUTTON_TEXT   = "Subscribe"
	UNSUBSCRIBE_BUTTON_TEXT = "Unsubscribe"
)

type CommandForm struct {
	Command   string `form:"command" binding:"required"`
	ChannelId string `form:"channel_id" binding:"required"`
	UserId    string `form:"user_id" binding:"required"`
}

func BotCommandHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var form CommandForm
		c.Bind(&form)
		switch form.Command {
		case "/news":
			var user model.User
			if err := db.Debug().Model(&model.User{}).Preload("SubscribedFeeds.Creator", "slack_id != ?", form.UserId).
				Preload("SubscribedFeeds.SubscribedChannels", "channel_slack_id = ?", form.ChannelId).
				Where("slack_id = ?", form.UserId).
				First(&user).Error; err != nil {
				Logger.Log.Error("failed to get user's feeds", err)
				c.JSON(http.StatusNotFound, gin.H{"text": "failed to get public feeds. please contact tech"})
				return
			}

			// subscribe section
			divSection := slack.NewDividerBlock()
			blocks := []slack.Block{divSection}

			sort.Slice(user.SubscribedFeeds, func(i, j int) bool {
				return user.SubscribedFeeds[i].Name < user.SubscribedFeeds[j].Name
			})

			for _, feed := range user.SubscribedFeeds {
				if len(feed.SubscribedChannels) == 0 {
					creator := ""
					if feed.Creator.Name != "" {
						creator = fmt.Sprintf("_%s_", feed.Creator.Name)
					}
					subscribeBtnText := slack.NewTextBlockObject("plain_text", SUBSCRIBE_BUTTON_TEXT, false, false)
					subscribeBtnEle := slack.NewButtonBlockElement(feed.Id, feed.Name, subscribeBtnText)
					optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t %s", feed.Name, creator), false, false)
					optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
					blocks = append(blocks, optionSection)
				}
			}

			blocks = append(blocks, divSection)

			for _, feed := range user.SubscribedFeeds {
				if len(feed.SubscribedChannels) == 1 {
					creator := ""
					if feed.Creator.Name != "" {
						creator = fmt.Sprintf("_%s_", feed.Creator.Name)
					}
					subscribeBtnText := slack.NewTextBlockObject("plain_text", UNSUBSCRIBE_BUTTON_TEXT, false, false)
					subscribeBtnEle := slack.NewButtonBlockElement(feed.Id, feed.Name, subscribeBtnText)
					optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t %s", feed.Name, creator), false, false)
					optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
					blocks = append(blocks, optionSection)
				}
			}

			msg := slack.NewBlockMessage(blocks...)
			b, err := json.MarshalIndent(msg, "", "    ")
			if err != nil {
				Logger.Log.Error("failed to build the message", err)
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
