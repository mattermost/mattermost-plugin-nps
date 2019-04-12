package main

import (
	"strings"

	"github.com/mattermost/mattermost-server/model"
	analytics "github.com/segmentio/analytics-go"
)

const (
	NPS_FEEDBACK = "nps_feedback"
	NPS_SCORE    = "nps_score"

	SEGMENT_KEY = "5xaDYWpjOoCKmJNNKK6fg1DacwZ7ZVZc"
)

func (p *Plugin) initializeClient() {
	client := analytics.New(SEGMENT_KEY)

	if !p.blockSegmentEvents {
		client.Identify(&analytics.Identify{
			UserId: p.API.GetDiagnosticId(),
		})
	}

	p.client = client
}

func (p *Plugin) sendScore(score int, userID string, timestamp int64) {
	p.sendToSegment(NPS_SCORE, userID, timestamp, map[string]interface{}{
		"score": score,
	})
}

func (p *Plugin) sendFeedback(feedback string, userID string, timestamp int64) {
	p.sendToSegment(NPS_FEEDBACK, userID, timestamp, map[string]interface{}{
		"feedback": feedback,
	})
}

func (p *Plugin) sendToSegment(event string, userID string, timestamp int64, properties map[string]interface{}) {
	if !p.canSendDiagnostics() || p.blockSegmentEvents {
		return
	}

	track := &analytics.Track{
		Event:      event,
		UserId:     p.API.GetDiagnosticId(),
		Properties: p.getEventProperties(userID, timestamp, properties),
	}

	p.client.Track(track)
}

func (p *Plugin) getEventProperties(userID string, timestamp int64, other map[string]interface{}) map[string]interface{} {
	properties := map[string]interface{}{
		"user_id":        userID,
		"timestamp":      timestamp,
		"server_version": p.API.GetServerVersion(), // Note that this calls the API directly, so it gets the full version (including patch version)
		"server_id":      p.API.GetDiagnosticId(),
	}

	if systemInstallDate, err := p.API.GetSystemInstallDate(); err != nil {
		properties["server_install_date"] = int64(0)
	} else {
		properties["server_install_date"] = systemInstallDate
	}

	if user, err := p.API.GetUser(userID); err != nil {
		properties["user_role"] = ""
		properties["user_create_at"] = int64(0)
	} else {
		properties["user_role"] = p.getUserRole(user)
		properties["user_create_at"] = user.CreateAt
	}

	if license := p.API.GetLicense(); license == nil {
		properties["license_id"] = ""
		properties["license_sku"] = ""
	} else {
		properties["license_id"] = license.Id
		properties["license_sku"] = license.SkuShortName
	}

	for key, value := range other {
		properties[key] = value
	}

	return properties
}

func (p *Plugin) getUserRole(user *model.User) string {
	if isSystemAdmin(user) {
		return "system_admin"
	} else if p.isUserTeamAdmin(user) {
		return "team_admin"
	} else {
		return "user"
	}
}

func (p *Plugin) isUserTeamAdmin(user *model.User) bool {
	page := 0
	perPage := 50

	for {
		teamMembers, err := p.API.GetTeamMembersForUser(user.Id, page, perPage)
		if err != nil {
			p.API.LogWarn("Failed to get role for user when sending NPS results")
			return false
		}

		for _, teamMember := range teamMembers {
			for _, role := range strings.Fields(teamMember.Roles) {
				if role == model.TEAM_ADMIN_ROLE_ID {
					return true
				}
			}
		}

		if len(teamMembers) != perPage {
			break
		}
	}

	return false
}
