package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
)

const (
	// How often "survey scheduled" emails can be sent to prevent multiple emails from being sent if multiple server
	// upgrades occur within a short time
	MIN_DAYS_BETWEEN_SURVEY_EMAILS = 7

	// How long until a survey occurs after a server upgrade
	DAYS_UNTIL_SURVEY = 21

	// Get admin users up to 100 at a time when sending email notifications
	ADMIN_USERS_PER_PAGE = 100
)

type serverUpgrade struct {
	Version   semver.Version
	Timestamp time.Time
}

// checkForNextSurvey schedules a new NPS survey if a major or minor version change has occurred. Returns whether or
// not a survey was scheduled.
//
// Note that this only sends an email to admins to notify them that a survey has been scheduled. The web app plugin is
// in charge of checking and actually triggering the survey.
func (p *Plugin) checkForNextSurvey(currentVersion semver.Version) bool {
	lastUpgrade := p.getLastServerUpgrade()

	if !shouldScheduleSurvey(currentVersion, lastUpgrade) {
		// No version change
		p.API.LogDebug("No server version change detected. Not scheduling a new survey.")
		return false
	}

	now := time.Now()
	nextSurvey := now.Add(DAYS_UNTIL_SURVEY * 24 * time.Hour)

	if lastUpgrade == nil {
		p.API.LogInfo(fmt.Sprintf("NPS plugin installed. Scheduling NPS survey for %s", nextSurvey.Format("Jan 2, 2006")))
	} else {
		p.API.LogInfo(fmt.Sprintf("Version change detected from %s to %s. Scheduling NPS survey for %s", lastUpgrade.Version, currentVersion, nextSurvey.Format("Jan 2, 2006")))
	}

	if shouldSendSurveyScheduledEmail(now, lastUpgrade) {
		p.sendSurveyScheduledEmail()
	}

	if err := p.storeServerUpgrade(&serverUpgrade{
		Version:   currentVersion,
		Timestamp: now,
	}); err != nil {
		p.API.LogError("Failed to store time of server upgrade. The next NPS survey may not occur.", "err", err)
	}

	return true
}

func (p *Plugin) getLastServerUpgrade() *serverUpgrade {
	upgradeBytes, appErr := p.API.KVGet(SERVER_UPGRADE_KEY)
	if appErr != nil || upgradeBytes == nil {
		return nil
	}

	var upgrade serverUpgrade

	err := json.Unmarshal(upgradeBytes, &upgrade)
	if err != nil {
		return nil
	}

	return &upgrade
}

func (p *Plugin) storeServerUpgrade(upgrade *serverUpgrade) *model.AppError {
	upgradeBytes, err := json.Marshal(upgrade)
	if err != nil {
		return &model.AppError{Message: err.Error()}
	}

	return p.API.KVSet(SERVER_UPGRADE_KEY, upgradeBytes)
}

func shouldScheduleSurvey(currentVersion semver.Version, lastUpgrade *serverUpgrade) bool {
	return lastUpgrade == nil || currentVersion.Major > lastUpgrade.Version.Major || currentVersion.Minor > lastUpgrade.Version.Minor
}

func shouldSendSurveyScheduledEmail(now time.Time, lastUpgrade *serverUpgrade) bool {
	// Only send a "survey scheduled" email if it has been at least 7 days since the last time we've sent one to
	// prevent spamming admins when multiple upgrades are done within a short period.
	return lastUpgrade == nil || now.Sub(lastUpgrade.Timestamp) >= MIN_DAYS_BETWEEN_SURVEY_EMAILS*24*time.Hour
}

func (p *Plugin) sendSurveyScheduledEmail() {
	admins, err := p.getAdminUsers(ADMIN_USERS_PER_PAGE)
	if err != nil {
		p.API.LogError("Failed to get system admins to send NPS survey notification emails", "err", err)
		return
	}

	config := p.API.GetConfig()

	subject := fmt.Sprintf("[%s] Net Promoter Score survey scheduled in %d days", *config.TeamSettings.SiteName, DAYS_UNTIL_SURVEY)

	bodyProps := map[string]interface{}{
		"SiteURL":         *config.ServiceSettings.SiteURL,
		"DaysUntilSurvey": DAYS_UNTIL_SURVEY,
	}
	if config.EmailSettings.FeedbackOrganization != nil && *config.EmailSettings.FeedbackOrganization != "" {
		bodyProps["Organization"] = "Sent by " + *config.EmailSettings.FeedbackOrganization
	} else {
		bodyProps["Organization"] = ""
	}

	var buf bytes.Buffer
	if err := emailBodyTemplate.Execute(&buf, bodyProps); err != nil {
		p.API.LogError("Failed to prepare NPS survey notification email", "err", err)
		return
	}
	body := buf.String()

	for _, admin := range admins {
		if admin.DeleteAt != 0 {
			continue
		}

		p.API.LogDebug("Sending NPS survey notification email", "email", admin.Email)

		if err := p.API.SendMail(admin.Email, subject, body); err != nil {
			p.API.LogError("Failed to send NPS survey notification email", "email", admin.Email, "err", err)
		}
	}
}

func (p *Plugin) getAdminUsers(perPage int) ([]*model.User, *model.AppError) {
	var admins []*model.User

	page := 0

	for {
		adminsPage, err := p.API.GetUsers(&model.UserGetOptions{Page: page, PerPage: perPage, Role: "system_admin"})
		if err != nil {
			return nil, err
		}

		admins = append(admins, adminsPage...)

		if len(adminsPage) < perPage {
			break
		}

		page += 1
	}

	return admins, nil
}
