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

	SURVEYBOT_DESCRIPTION = "Surveybot collects user feedback to improve Mattermost. [Learn more](https://mattermost.com/pl/default-nps)."
)

type Plugin struct {
	plugin.MattermostPlugin

	configurationLock sync.RWMutex
	configuration     *configuration

	activatedLock sync.RWMutex
	activated     bool

	surveyLock sync.Mutex

	connectedLock sync.Mutex

	serverVersion semver.Version

	botUserId string

	client *analytics.Client
}
