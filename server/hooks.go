package main

import (
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func (p *Plugin) ChannelHasBeenCreated(c *plugin.Context, channel *model.Channel) {
	// Set the description for any DM channels opened between Surveybot and a user
	if channel.Type != model.CHANNEL_DIRECT {
		return
	}

	if !strings.HasPrefix(channel.Name, p.botUserId+"__") && !strings.HasSuffix(channel.Name, "__"+p.botUserId) {
		return
	}

	channel.Header = "Surveybot collects user feedback to improve Mattermost. [Learn more](https://mattermost.com/pl/default-nps)."

	if _, err := p.API.UpdateChannel(channel); err != nil {
		p.API.LogWarn("Failed to set channel header for Surveybot", "err", err)
	}
}
