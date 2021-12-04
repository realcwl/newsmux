package bot

// This handler is for slack slash commands
// https://api.slack.com/interactivity/slash-commands

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/Luismorlan/newsmux/model"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const SIMILARITY_THRESHOLD = 37
const SIMILARITY_WINDOW_HOURS = 1

var PostsSent map[string][]postMeta

func init() {
	PostsSent = make(map[string][]postMeta)
}

type postMeta struct {
	Id           string `json:"id"`
	SemanticHash string `json:"semantic_hash"`
	PostTime     time.Time
}

func isHashingSemanticallyIdentical(h1 string, h2 string) bool {
	// If the hashing is invalid, or not of same length, they cannot be considered
	// as the semantically identical.
	if h1 == "" || h2 == "" || len(h1) != len(h2) {
		return false
	}

	// Calculate hamming distance by counting how many different bits in total.
	count := 0
	for idx := 0; idx < len(h1); idx++ {
		if h1[idx] != h2[idx] {
			count++
		}
	}

	return count <= SIMILARITY_THRESHOLD
}

func isPostDuplicated(
	lhs model.Post,
	channelId string,
) bool {
	for k, v := range PostsSent[channelId] {
		// the collector has some interval(up to 12 hours for zsxq) to collect the data
		// we will keep the cache for one day
		if math.Abs(time.Since(v.PostTime).Hours()) > 24 {
			PostsSent[channelId] = append(PostsSent[channelId][:k], PostsSent[channelId][k+1:]...)
		}

		if lhs.SemanticHashing == "" ||
			v.SemanticHash == "" {
			return false
		}

		if (math.Abs(lhs.ContentGeneratedAt.Sub(v.PostTime).Hours())) < SIMILARITY_WINDOW_HOURS {
			return isHashingSemanticallyIdentical(lhs.SemanticHashing, v.SemanticHash)
		}
	}
	return false
}

func PostShareHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelId, ok := c.GetQuery("channel_id")
		if !ok {
			Logger.Log.Error("got an post share request without channel")
			c.JSON(http.StatusBadRequest, gin.H{"error": "channel id is required"})
			return
		}

		var channel model.Channel
		res := db.Where("id = ?", channelId).First(&channel)
		if res.RowsAffected == 0 {
			Logger.Log.Errorf("invalid channel id: %s", channelId)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
			return
		}
		fmt.Println("c", channel)

		postId, ok := c.GetQuery("post_id")
		if !ok {
			Logger.Log.Error("got an post share request without post")
			c.JSON(http.StatusBadRequest, gin.H{"error": "post id is required"})
			return
		}

		var post *model.Post
		res = db.Preload("SubSource").Preload("SharedFromPost").Preload("SharedFromPost.SubSource").Where("id=?", postId).First(&post)
		if res.RowsAffected == 0 {
			Logger.Log.Errorf("invalid post id: %s", postId)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
			return
		}

		if isPostDuplicated(*post, channelId) {
			c.Data(200, "application/json; charset=utf-8", []byte("Post duplicated"))
			return
		}

		if err := PushPostViaWebhook(*post, channel.WebhookUrl); err != nil {
			Logger.Log.Error("Fail to post via webhook", err)
		}
		if _, ok := PostsSent[channelId]; ok {
			PostsSent[channelId] = append(PostsSent[channelId],
				postMeta{
					Id:           post.Id,
					SemanticHash: post.SemanticHashing,
					PostTime:     post.ContentGeneratedAt,
				})
		} else {
			PostsSent[channelId] = []postMeta{
				{
					Id:           post.Id,
					SemanticHash: post.SemanticHashing,
					PostTime:     post.ContentGeneratedAt,
				},
			}
		}

		c.Data(200, "application/json; charset=utf-8", []byte("Post sent"))
	}
}
