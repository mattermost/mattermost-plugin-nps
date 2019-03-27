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

	client.Identify(&analytics.Identify{
		UserId: p.API.GetDiagnosticId(),
	})

	p.client = client
}

func (p *Plugin) sendScore(score int, userId string, timestamp int64) {
	p.sendToSegment(NPS_SCORE, p.getEventProperties(userId, timestamp, map[string]interface{}{
		"score": score,
	}))
}

func (p *Plugin) sendFeedback(feedback string, userId string, timestamp int64) {
	p.sendToSegment(NPS_FEEDBACK, p.getEventProperties(userId, timestamp, map[string]interface{}{
		"feedback": feedback,
	}))
}

func (p *Plugin) sendToSegment(event string, properties map[string]interface{}) {
	if !p.canSendDiagnostics() {
		return
	}

	track := &analytics.Track{
		Event:      event,
		UserId:     p.API.GetDiagnosticId(),
		Properties: properties,
	}

	p.client.Track(track)
}

func (p *Plugin) getEventProperties(userId string, timestamp int64, other map[string]interface{}) map[string]interface{} {
	properties := map[string]interface{}{
		"user_id":        userId,
		"timestamp":      timestamp,
		"server_version": p.API.GetServerVersion(),
		"server_id":      p.API.GetDiagnosticId(),
	}

	if systemInstallDate, err := p.API.GetSystemInstallDate(); err != nil {
		properties["server_install_date"] = int64(0)
	} else {
		properties["server_install_date"] = systemInstallDate
	}

	if user, err := p.API.GetUser(userId); err != nil {
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
	if p.isUserSystemAdmin(user) {
		return "system_admin"
	} else if p.isUserTeamAdmin(user) {
		return "team_admin"
	} else {
		return "user"
	}
}

func (p *Plugin) isUserSystemAdmin(user *model.User) bool {
	for _, role := range strings.Fields(user.Roles) {
		if role == model.SYSTEM_ADMIN_ROLE_ID {
			return true
		}
	}

	return false
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
