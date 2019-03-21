package main

import (
	"sync"

	"github.com/blang/semver"
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
