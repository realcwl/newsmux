package bot

import (
	"fmt"
	"strings"

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
	return slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("><%s|%s> %s", postLink, post.SubSource.Name, buildContentWithShowMore(post)), false, false)
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

func buildContentWithShowMore(post model.Post) string {
	if len(post.Content) > 600 {
		return fmt.Sprintf("%s...<%s|[查看全文]>", post.Content[:600], buildPostLink(post))
	}
	return post.Content
}

// PushPostViaWebhook is an async call to push a post to a channel
func PushPostViaWebhook(post model.Post, webhookUrl string) {
	blocks := []slack.Block{}
	blocks = append(blocks, buildSubsourceBlock(post))
	// build subsource and post body blocks
	if post.SharedFromPost != nil {
		sharedFromPost := post.SharedFromPost
		if post.Content != "" {
			sharedFromContext := slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", buildContentWithShowMore(post), false, false))
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
		bodyBlockElements = append(bodyBlockElements, slack.NewTextBlockObject("mrkdwn", buildContentWithShowMore(post), false, false))
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
		Blocks: &slack.Blocks{BlockSet: blocks},
	}
	err := slack.PostWebhook(webhookUrl, webhookMsg)
	if err != nil {
		Logger.Log.Error(err)
	}
}
