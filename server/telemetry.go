// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/logger"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/telemetry"
)

const (
	NpsFeedback = "nps_feedback"
	NpsScore    = "nps_score"
	NpsDisable  = "nps_disable"
)

func (p *Plugin) initializeTelemetryClient() error {
	client, err := telemetry.NewRudderClient()

	p.telemetryClient = client
	return err
}

func (p *Plugin) initTracker() {
	p.tracker = telemetry.NewTracker(p.telemetryClient, p.API.GetDiagnosticId(), p.API.GetServerVersion(), manifest.Id, manifest.Version, "nps", telemetry.NewTrackerConfig(p.API.GetConfig()), logger.New(p.API))
}

func (p *Plugin) sendScore(score int, userID string, timestamp int64) {
	_ = p.tracker.TrackUserEvent(NpsScore, userID, p.getEventProperties(userID, timestamp, map[string]interface{}{
		"score": score,
	}))
}

func (p *Plugin) sendFeedback(feedback string, email string, userID string, timestamp int64) {
	_ = p.tracker.TrackUserEvent(NpsFeedback, userID, p.getEventProperties(userID, timestamp, map[string]interface{}{
		"feedback": feedback,
		"email":    email,
	}))
}

func (p *Plugin) sendUserDisabledEvent(userID string, timestamp int64) {
	_ = p.tracker.TrackUserEvent(NpsDisable, userID, p.getEventProperties(userID, timestamp, map[string]interface{}{}))
}

func (p *Plugin) getEventProperties(userID string, timestamp int64, other map[string]interface{}) map[string]interface{} {
	properties := map[string]interface{}{
		"timestamp": timestamp,
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
	switch {
	case isSystemAdmin(user):
		return "system_admin"
	case p.isUserTeamAdmin(user):
		return "team_admin"
	default:
		return "user"
	}
}

func (p *Plugin) isUserTeamAdmin(user *model.User) bool {
	page := 0
	perPage := 50

	for {
		teamMembers, err := p.API.GetTeamMembersForUser(user.Id, page, perPage)
		if err != nil {
			p.API.LogWarn("Failed to get role for user when sending survey results")
			return false
		}

		for _, teamMember := range teamMembers {
			for _, role := range strings.Fields(teamMember.Roles) {
				if role == model.TeamAdminRoleId {
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
