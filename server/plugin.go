// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"os"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/telemetry"
)

const (
	// AdminDmNoticeKey is used to store whether or not a DM notifying an admin about a scheduled survey has been
	// sent. It should contain the user's ID and server version like "AdminDM-abc123-5.10.0".
	AdminDmNoticeKey = "AdminDM-%s-%s"

	// LastAdminNoticeKey is used to store the last time.Time that notifications were sent to admins to inform them
	// of an upcoming NPS survey.
	LastAdminNoticeKey = "LastAdminNotice"

	// WelcomeFeedbackMigrationKey is used to store when WelcomeFeedback have been actived for the first time on this server
	// to ensure we don't send the message to older users.
	WelcomeFeedbackMigrationKey = "WelcomeFeedbackMigration"

	// ServerUpgradeKey is used to store a serverUpgrade object containing when an upgrade to a given version first
	// occurred. It should contain the server version like "ServerUpgrade-5.10.0".
	ServerUpgradeKey = "ServerUpgrade-%s"

	// SurveyKey is used to store the surveyState containing when an NPS survey starts and ends on a given version
	// of Mattermost. It should contain the server version like "Survey-5.10.0".
	SurveyKey = "Survey-%s"

	// UserSurveyKey is used to store the userSurveyState tracking a user's progress through an NPS survey on the
	// given version of Mattermost. It should contain the user's ID like "UserSurvey-abc123".
	UserSurveyKey = "UserSurvey-%s"

	// UserSurveyStateKey is used to know if we sent the welcome feedback post to new users.
	// Format is 'UserWelcomeFeedback-{user_id}'
	UserWelcomeFeedbackKey = "UserWelcomeFeedback-%s"

	FeedbackbotDescription = "Feedbackbot collects user feedback to improve Mattermost. [Learn more](https://mattermost.com/pl/default-nps)."
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// serverVersion is the current major/minor server version without the patch version included.
	serverVersion string

	// activated is used to track whether or not OnActivate has initialized the plugin state.
	activated bool

	// welcomeFeedbackAfter is the date after which new users can get the welcome feedback post.
	welcomeFeedbackAfter time.Time

	botUserID string

	telemetryClient telemetry.Client
	tracker         telemetry.Tracker

	// now provides access to time.Now in a way that is mockable for unit testing.
	now func() time.Time

	// readFile provides access to os.ReadFile in a way that is mockable for unit testing.
	readFile func(path string) ([]byte, error)
}

func NewPlugin() *Plugin {
	return &Plugin{
		now:      time.Now,
		readFile: os.ReadFile,
	}
}
