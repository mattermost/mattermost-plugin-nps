package main

import (
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	analytics "github.com/segmentio/analytics-go"
)

const (
	ADMIN_DM_NOTICE_KEY = "AdminDM-"
	BOT_USER_KEY        = "Bot"
	SERVER_UPGRADE_KEY  = "ServerUpgrade"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	botUserId string

	client *analytics.Client
}

func (p *Plugin) CreateBotDMPost(userID, message, postType string) *model.AppError {
	channel, err := p.API.GetDirectChannel(userID, p.botUserId)
	if err != nil {
		p.API.LogError("Couldn't get bot's DM channel", "user_id", userID, "err", err)
		return err
	}

	post := &model.Post{
		UserId:    p.botUserId,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props: map[string]interface{}{
			"from_webhook":      "true",
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogError("Couldn't send bot DM", "user_id", userID, "err", err)
		return err
	}

	return nil
}