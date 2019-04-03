package main

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/plugin"
	analytics "github.com/segmentio/analytics-go"
)

const (
	ADMIN_DM_NOTICE_KEY = "AdminDM-"
	SERVER_UPGRADE_KEY  = "ServerUpgrade"
	USER_SURVEY_KEY     = "UserSurvey-"

	SURVEYBOT_DESCRIPTION = "Surveybot collects user feedback to improve Mattermost. [Learn more](https://mattermost.com/pl/default-nps)."

	DEFAULT_UPGRADE_CHECK_MAX_DELAY = 2 * time.Minute
	DEAFULT_USER_SURVEY_MAX_DELAY   = 3 * time.Second
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// activated is used to track whether or not OnActivate has initialized the plugin state.
	activated bool

	// surveyLock is used to prevent multiple threads from accessing checkForNextSurvey at the same time.
	surveyLock sync.Mutex

	// connectedLock is used to prevent multiple connected requests from being handled at the same time in order to
	// prevent users from receiving duplicate Surveybot DMs.
	connectedLock sync.Mutex

	botUserId string

	client *analytics.Client

	// upgradeCheckMaxDelay adds a short delay to checkForNextSurvey calls to mitigate against race conditions caused
	// by multiple servers restarting at the same time after an upgrade.
	upgradeCheckMaxDelay time.Duration

	// userSurveyMaxDelay adds a short delay when checking whether the user needs to receive DMs notifying them of a new
	// NPS survey.
	userSurveyMaxDelay time.Duration

	// readFile provides access to ioutil.ReadFile in a way that is mockable for unit testing.
	readFile func(path string) ([]byte, error)
}

func NewPlugin() *Plugin {
	return &Plugin{
		upgradeCheckMaxDelay: DEFAULT_UPGRADE_CHECK_MAX_DELAY,
		userSurveyMaxDelay:   DEAFULT_USER_SURVEY_MAX_DELAY,

		readFile: ioutil.ReadFile,
	}
}
