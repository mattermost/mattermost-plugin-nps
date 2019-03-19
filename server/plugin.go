package main

import (
	"sync"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	analytics "github.com/segmentio/analytics-go"
)

const (
	ADMIN_DM_NOTICE_KEY = "AdminDM-"
	BOT_USER_KEY        = "Bot"
	SERVER_UPGRADE_KEY  = "ServerUpgrade"
	USER_SURVEY_KEY     = "UserSurvey-"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	serverVersion semver.Version

	botUserId string

	client *analytics.Client

	connectedLock sync.Mutex
}

func (p *Plugin) CreateBotDMPost(userID, message, postType string, props map[string]interface{}) (*model.Post, *model.AppError) {
	channel, err := p.API.GetDirectChannel(userID, p.botUserId)
	if err != nil {
		p.API.LogError("Couldn't get bot's DM channel", "user_id", userID, "err", err)
		return nil, err
	}

	if props == nil {
		props = make(map[string]interface{})
	}
	props["from_webhook"] = true

	post, err := p.API.CreatePost(&model.Post{
		UserId:    p.botUserId,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props:     props,
	})
	if err != nil {
		p.API.LogError("Couldn't send bot DM", "user_id", userID, "err", err)
		return nil, err
	}

	return post, nil
}