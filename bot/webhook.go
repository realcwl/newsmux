package bot

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Luismorlan/newsmux/model"
	Logger "github.com/Luismorlan/newsmux/utils/log"
	"github.com/slack-go/slack"
)

func buildPostLink(post model.Post) string {
	return fmt.Sprintf("https://rnr.capital/shared-posts/%s", post.Id)
}

func buildSubsourceBlock(post model.Post) slack.Block {
	return slack.NewContextBlock("",
		slack.NewImageBlockElement(post.SubSource.AvatarUrl, post.SubSource.Name),
		slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("<%s|%s>", buildPostLink(post), post.SubSource.Name), false, false))
}

func buildRetweetBlock(post model.Post, postLink string) slack.MixedElement {
	return slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("><%s|%s> %s", postLink, post.SubSource.Name, buildContentWithShowMore(post, postLink)), false, false)
}

// buildImageElements should be used only when we have 2+ images
func buildImageElements(post model.Post) []slack.MixedElement {
	elements := []slack.MixedElement{}
	for _, imageUrl := range post.ImageUrls {
		elements = append(elements, slack.NewImageBlockElement(imageUrl, "post image"))
	}
	return elements
}

func buildFileObject(post model.Post) *slack.TextBlockObject {
	fileBlockText := "```"
	for i, url := range post.FileUrls {
		if i > 0 {
			fileBlockText += "\n"
		}
		fileBlockText += fmt.Sprintf("<%s|📄 %s>", url, url[strings.LastIndex(url, "/")+1:])
	}
	fileBlockText += "```"
	return slack.NewTextBlockObject("mrkdwn", fileBlockText, false, false)
}

func buildContentWithShowMore(post model.Post, postLink string) string {
	contentRunes := []rune(post.Content)
	if len(contentRunes) > 400 {
		return fmt.Sprintf("%s...<%s|[查看全文]>", string(contentRunes[:400]), postLink)
	}
	return post.Content
}

func TimeBoundedPushPost(ctx context.Context, channelId string, postId string) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		r, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s?channel_id=%s&post_id=%s", os.Getenv("BOT_SHARE_POST_URL"), channelId, postId), nil)
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		client := &http.Client{}
		_, err := client.Do(r)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			Logger.Log.Error("failed to push post to channel", err)
		}
		return
	case <-ctx.Done():
		Logger.Log.Errorf("push post via webhook timed out. post: %s, channel: %s", postId, channelId)
		return
	}
}

// PushPostViaWebhook is an async call to push a post to a channel
func PushPostViaWebhook(post model.Post, webhookUrl string) error {
	blocks := []slack.Block{}
	blocks = append(blocks, buildSubsourceBlock(post))
	// build subsource and post body blocks
	if post.SharedFromPost != nil {
		sharedFromPost := post.SharedFromPost
		if post.Content != "" {
			sharedFromContext := slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", buildContentWithShowMore(post, buildPostLink(post)), false, false))
			blocks = append(blocks, sharedFromContext)
		}

		sharedFromContextElements := []slack.MixedElement{buildRetweetBlock(*sharedFromPost, buildPostLink(post))}
		if len(sharedFromPost.ImageUrls) > 1 {
			sharedFromContextElements = append(sharedFromContextElements, buildImageElements(*sharedFromPost)...)
		}
		blocks = append(blocks, slack.NewContextBlock("", sharedFromContextElements...))
	} else {
		bodyBlockElements := []slack.MixedElement{}
		if post.Title != "" {
			bodyBlockElements = append(bodyBlockElements, slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*", post.Title), false, false))
		}
		bodyBlockElements = append(bodyBlockElements, slack.NewTextBlockObject("mrkdwn", buildContentWithShowMore(post, buildPostLink(post)), false, false))
		blocks = append(blocks, slack.NewContextBlock("", bodyBlockElements...))
		if len(post.ImageUrls) > 1 {
			blocks = append(blocks, slack.NewContextBlock("", buildImageElements(post)...))
		}
		if len(post.FileUrls) > 0 {
			blocks = append(blocks, slack.NewContextBlock("", buildFileObject(post)))
		}
	}

	if len(post.ImageUrls) == 1 {
		blocks = append(blocks, slack.NewImageBlock(post.ImageUrls[0], "post image", "", nil))
	}

	webhookMsg := &slack.WebhookMessage{
		Text:   fmt.Sprintf("%s: %s...", post.SubSource.Name, string([]rune(post.Content)[:30])),
		Blocks: &slack.Blocks{BlockSet: blocks},
	}

	err := slack.PostWebhook(webhookUrl, webhookMsg)
	if err != nil {
		Logger.Log.Error(err)
		return err
	}

	return nil
}
