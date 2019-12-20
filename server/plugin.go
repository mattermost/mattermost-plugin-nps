package main

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/v5/plugin"
	analytics "github.com/segmentio/analytics-go/v2"
)

const (
	// ADMIN_DM_NOTICE_KEY is used to store whether or not a DM notifying an admin about a scheduled survey has been
	// sent. It should contain the user's ID and server version like "AdminDM-abc123-5.10.0".
	ADMIN_DM_NOTICE_KEY = "AdminDM-%s-%s"

	// LAST_ADMIN_NOTICE_KEY is used to store the last time.Time that notifications were sent to admins to inform them
	// of an upcoming NPS survey.
	LAST_ADMIN_NOTICE_KEY = "LastAdminNotice"

	// SERVER_UPGRADE_KEY is used to store a serverUpgrade object containing when an upgrade to a given version first
	// occurred. It should contain the server version like "ServerUpgrade-5.10.0".
	SERVER_UPGRADE_KEY = "ServerUpgrade-%s"

	// SURVEY_KEY is used to store the surveyState containing when an NPS survey starts and ends on a given version
	// of Mattermost. It should contain the server version like "Survey-5.10.0".
	SURVEY_KEY = "Survey-%s"

	// USER_SURVEY_KEY is used to store the userSurveyState tracking a user's progress through an NPS survey on the
	// given version of Mattermost. It should contain the user's ID like "UserSurvey-abc123".
	USER_SURVEY_KEY = "UserSurvey-%s"

	SURVEYBOT_DESCRIPTION = "Surveybot collects user feedback to improve Mattermost. [Learn more](https://mattermost.com/pl/default-nps)."
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

	botUserID string

	client *analytics.Client

	// blockSegmentEvents prevents the plugin from sending events to Segment during testing.
	blockSegmentEvents bool

	// now provides access to time.Now in a way that is mockable for unit testing.
	now func() time.Time

	// readFile provides access to ioutil.ReadFile in a way that is mockable for unit testing.
	readFile func(path string) ([]byte, error)
}

func NewPlugin() *Plugin {
	return &Plugin{
		now:      time.Now,
		readFile: ioutil.ReadFile,
	}
}
