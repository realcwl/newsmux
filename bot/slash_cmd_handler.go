package bot

// This handler is for slack slash commands
// https://api.slack.com/interactivity/slash-commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Luismorlan/newsmux/model"
	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

const (
	SUBSCRIBE_BUTTON_TEXT   = "Subscribe"
	UNSUBSCRIBE_BUTTON_TEXT = "Unsubscribe"
)

type FeedResultForChannel struct {
	model.Feed
	IsSubscribed bool
}

type CommandForm struct {
	Command   string `form:"command" binding:"required"`
	ChannelId string `form:"channel_id" binding:"required"`
}

func BotCommandHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var form CommandForm
		c.Bind(&form)
		fmt.Println("form", form)
		switch form.Command {
		case "/feeds":
			var visibleFeeds []FeedResultForChannel
			if err := db.Debug().Model(&model.Feed{}).Preload("Creator").Select("feed.*, channel_feed_subscription").
				Joins("LEFT JOIN channel_feed_subscription ON feeds.id = channel_feed_subscrpition.feed_id").
				Joins("LEFT JOIN channels ON channels.id = channel_feed_subscriptions.channel_id").
				Where("visibility = 'GLOBAL'").Or("channels.channel_slack_id = ?", form.ChannelId).
				Or("creator.slack_id = ?").
				Find(&visibleFeeds).Order("subscribers desc").Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"text": "failed to get public feeds. please contact tech"})
				return
			}

			// var subscribedFeeds []*model.Feed
			// db.Model(&model.ChannelFeedSubscription{}).
			// 	Select("feeds.id").
			// 	Joins("INNER JOIN feeds ON feeds.id = channel_feed_subscriptions.feed_id").
			// 	Joins("INNER JOIN channels ON channels.id = channel_feed_subscriptions.channel_id").
			// 	Where("channels.channel_slack_id = ?", form.ChannelId).
			// 	Find(&subscribedFeeds)

			// subscribedFeedIds := map[string]struct{}{}
			// for _, feed := range subscribedFeeds {
			// 	subscribedFeedIds[feed.Id] = struct{}{}
			// }

			// var unsubscribedFeeds []*model.Feed
			// subscribedFeeds = []*model.Feed{}

			/* Building message body
			examples can be found here https://github.com/slack-go/slack/tree/master/examples/blocks
			*/
			// subscribe section
			divSection := slack.NewDividerBlock()
			blocks := []slack.Block{divSection}

			for _, feed := range visibleFeeds {
				if feed.IsSubscribed {
					subscribeBtnText := slack.NewTextBlockObject("plain_text", SUBSCRIBE_BUTTON_TEXT, false, false)
					subscribeBtnEle := slack.NewButtonBlockElement(feed.Id, feed.Name, subscribeBtnText)
					optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t _%s_", feed.Name, feed.Creator.Name), false, false)
					optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
					blocks = append(blocks, optionSection)
				}
			}

			blocks = append(blocks, divSection)

			for _, feed := range visibleFeeds {
				if !feed.IsSubscribed {
					subscribeBtnText := slack.NewTextBlockObject("plain_text", UNSUBSCRIBE_BUTTON_TEXT, false, false)
					subscribeBtnEle := slack.NewButtonBlockElement(feed.Id, feed.Name, subscribeBtnText)
					optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t _%s_", feed.Name, feed.Creator.Name), false, false)
					optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
					blocks = append(blocks, optionSection)
				}
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
