package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) ChannelHasBeenCreated(c *plugin.Context, channel *model.Channel) {
	// Set the description for any DM channels opened between Feedbackbot and a user
	if !p.IsBotDMChannel(channel) {
		return
	}

	channel.Header = FeedbackbotDescription

	if _, err := p.API.UpdateChannel(channel); err != nil {
		p.API.LogWarn("Failed to set channel header for Feedbackbot", "err", err)
	}
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	if !p.canSendDiagnostics() {
		return
	}

	// Make sure that Feedbackbot doesn't respond to itself
	if post.UserId == p.botUserID {
		return
	}

	// Or to system messages
	if post.IsSystemMessage() {
		return
	}

	// Make sure this is a post sent directly to Feedbackbot
	channel, appErr := p.API.GetChannel(post.ChannelId)
	if appErr != nil {
		p.API.LogError("Unable to get channel for Feedbackbot feedback", "err", appErr)
		return
	}

	if !p.IsBotDMChannel(channel) {
		return
	}

	// Make sure this is not a post sent by another bot
	user, appErr := p.API.GetUser(post.UserId)
	if appErr != nil {
		p.API.LogError("Unable to get sender for Feedbackbot feedback", "err", appErr)
		return
	}

	if user.IsBot {
		return
	}

	emailStr := ""
	email := post.GetProp("feedback_email")
	if email != nil {
		if emailVal, ok := email.(string); ok && emailVal != "" {
			emailStr = emailVal
		}
	}
	// Send the feedback to Segment
	p.sendFeedback(post.Message, emailStr, post.UserId, post.CreateAt)

	rootID := post.RootId
	// if it is a new post in the channel, update response RootId
	if rootID == "" {
		rootID = post.Id
	}

	// Respond to the feedback which is a previous comment
	_, appErr = p.CreateBotDMPost(post.UserId, &model.Post{
		Message: feedbackResponseBody,
		Type:    "custom_nps_thanks",
		RootId:  rootID,
	})
	if appErr != nil {
		p.API.LogError("Failed to respond to Feedbackbot feedback")
	}
}

func (p *Plugin) UserHasLoggedIn(c *plugin.Context, user *model.User) {
	if err := p.checkForDMs(user.Id); err != nil {
		p.API.LogError("Failed to check for user notifications on login", "user_id", user.Id, "err", err)
	}
}
