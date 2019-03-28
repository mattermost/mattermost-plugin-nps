package main

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func (p *Plugin) ChannelHasBeenCreated(c *plugin.Context, channel *model.Channel) {
	// Set the description for any DM channels opened between Surveybot and a user
	if !p.isBotDMChannel(channel) {
		return
	}

	channel.Header = SURVEYBOT_DESCRIPTION

	if _, err := p.API.UpdateChannel(channel); err != nil {
		p.API.LogWarn("Failed to set channel header for Surveybot", "err", err)
	}
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	if !p.canSendDiagnostics() {
		return
	}

	if post.UserId == p.botUserId {
		return
	}

	// Respond to any written feedback that the bot receives
	channel, err := p.API.GetChannel(post.ChannelId)
	if err != nil {
		p.API.LogWarn("Unable to get channel for post to send Surveybot response", "err", err)
		return
	}

	if !p.isBotDMChannel(channel) {
		return
	}

	// Send the feedback to Segment
	p.sendFeedback(post.Message, post.UserId, post.CreateAt)

	// Respond to the feedback
	_, err = p.CreateBotDMPost(post.UserId, &model.Post{
		Message: feedbackResponseBody,
		Type:    "custom_nps_thanks",
	})
	if err != nil {
		p.API.LogError("Failed to respond to Surveybot feedback")
	}
}

func (p *Plugin) UserHasLoggedIn(c *plugin.Context, user *model.User) {
	if err := p.checkForDMs(user.Id); err != nil {
		p.API.LogError("Failed to check for user notifications on login", "user_id", user.Id, "err", err)
	}
}
