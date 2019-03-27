package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
)

const (
	// How often "survey scheduled" emails can be sent to prevent multiple emails from being sent if multiple server
	// upgrades occur within a short time
	MIN_TIME_BETWEEN_SURVEY_EMAILS = 7 * 24 * time.Hour

	// How long until a survey occurs after a server upgrade in days (for use in notifications)
	DAYS_UNTIL_SURVEY = 21

	// How long until a survey occurs after a server upgrade as a time.Duration
	TIME_UNTIL_SURVEY = 21 * 24 * time.Hour

	// Get admin users up to 100 at a time when sending email notifications
	ADMIN_USERS_PER_PAGE = 100

	// The minimum time before a user can be sent a survey after completing the previous one
	MIN_TIME_BETWEEN_USER_SURVEYS = 90 * 24 * time.Hour
)

type adminNotice struct {
	Sent       bool
	NextSurvey time.Time
}

// checkForNextSurvey schedules a new NPS survey if a major or minor version change has occurred. Returns whether or
// not a survey was scheduled.
//
// Note that this only sends an email to admins to notify them that a survey has been scheduled. The web app plugin is
// in charge of checking and actually triggering the survey.
func (p *Plugin) checkForNextSurvey(currentVersion semver.Version) bool {
	p.surveyLock.Lock()
	defer p.surveyLock.Unlock()

	if !p.getConfiguration().EnableSurvey {
		// Surveys are disabled, so return false without updating the stored version. If surveys are re-enabled, the
		// plugin will then detect an upgrade (if one occurred) and schedule the next survey.
		p.API.LogDebug("Not sending NPS survey because survey is disabled")
		return false
	}

	lastUpgrade, _ := p.getLastServerUpgrade()

	if !shouldScheduleSurvey(currentVersion, lastUpgrade) {
		// No version change
		p.API.LogDebug("No server version change detected. Not scheduling a new survey.")
		return false
	}

	now := time.Now().UTC()
	nextSurvey := now.Add(TIME_UNTIL_SURVEY)

	if lastUpgrade == nil {
		p.API.LogInfo(fmt.Sprintf("NPS plugin installed. Scheduling NPS survey for %s", nextSurvey.Format("Jan 2, 2006")))
	} else {
		p.API.LogInfo(fmt.Sprintf("Version change detected from %s to %s. Scheduling NPS survey for %s", lastUpgrade.Version, currentVersion, nextSurvey.Format("Jan 2, 2006")))
	}

	if shouldSendAdminNotices(now, lastUpgrade) {
		p.sendAdminNotices(nextSurvey)
	}

	if err := p.storeServerUpgrade(&serverUpgrade{
		Version:   currentVersion,
		Timestamp: now,
	}); err != nil {
		p.API.LogError("Failed to store time of server upgrade. The next NPS survey may not occur.", "err", err)
	}

	return true
}

func shouldScheduleSurvey(currentVersion semver.Version, lastUpgrade *serverUpgrade) bool {
	return lastUpgrade == nil || currentVersion.Major > lastUpgrade.Version.Major || currentVersion.Minor > lastUpgrade.Version.Minor
}

func shouldSendAdminNotices(now time.Time, lastUpgrade *serverUpgrade) bool {
	// Only send a "survey scheduled" email if it has been at least 7 days since the last time we've sent one to
	// prevent spamming admins when multiple upgrades are done within a short period.
	return lastUpgrade == nil || now.Sub(lastUpgrade.Timestamp) >= MIN_TIME_BETWEEN_SURVEY_EMAILS
}

func (p *Plugin) sendAdminNotices(nextSurvey time.Time) {
	admins, err := p.getAdminUsers(ADMIN_USERS_PER_PAGE)
	if err != nil {
		p.API.LogError("Failed to get system admins to send admin notices", "err", err)
		return
	}

	p.sendAdminNoticeEmails(admins)
	p.sendAdminNoticeDMs(admins, nextSurvey)
}

func (p *Plugin) sendAdminNoticeEmails(admins []*model.User) {
	config := p.API.GetConfig()

	subject := fmt.Sprintf(adminEmailSubject, *config.TeamSettings.SiteName, DAYS_UNTIL_SURVEY)

	bodyProps := map[string]interface{}{
		"PluginID":        manifest.Id,
		"SiteURL":         *config.ServiceSettings.SiteURL,
		"DaysUntilSurvey": DAYS_UNTIL_SURVEY,
	}
	if config.EmailSettings.FeedbackOrganization != nil && *config.EmailSettings.FeedbackOrganization != "" {
		bodyProps["Organization"] = "Sent by " + *config.EmailSettings.FeedbackOrganization
	} else {
		bodyProps["Organization"] = ""
	}

	var buf bytes.Buffer
	if err := adminEmailBodyTemplate.Execute(&buf, bodyProps); err != nil {
		p.API.LogError("Failed to prepare NPS survey notification email", "err", err)
		return
	}
	body := buf.String()

	for _, admin := range admins {
		p.API.LogDebug("Sending NPS survey notification email", "email", admin.Email)

		if err := p.API.SendMail(admin.Email, subject, body); err != nil {
			p.API.LogError("Failed to send NPS survey notification email", "email", admin.Email, "err", err)
		}
	}
}

func (p *Plugin) sendAdminNoticeDMs(admins []*model.User, nextSurvey time.Time) {
	// Actual DMs will be sent when the admins next log in, so just mark that they're scheduled to receive one
	for _, admin := range admins {
		err := p.KVSet(ADMIN_DM_NOTICE_KEY+admin.Id, &adminNotice{
			Sent:       false,
			NextSurvey: nextSurvey,
		})
		if err != nil {
			p.API.LogError("Failed to store scheduled admin notice", "err", err)
			continue
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

		for _, admin := range adminsPage {
			// Filter out deactivated users
			if admin.DeleteAt > 0 {
				continue
			}

			admins = append(admins, admin)
		}

		if len(adminsPage) < perPage {
			break
		}

		page += 1
	}

	return admins, nil
}

func (p *Plugin) checkForAdminNoticeDM(user *model.User) *adminNotice {
	if !p.getConfiguration().EnableSurvey {
		// Surveys are disabled
		return nil
	}

	if !isSystemAdmin(user) {
		return nil
	}

	var notice *adminNotice
	err := p.KVGet(ADMIN_DM_NOTICE_KEY+user.Id, &notice)

	if err != nil {
		p.API.LogError("Failed to get scheduled admin notice", "err", err)
		return nil
	}

	if notice == nil {
		// No notice stored for this user, likely because they were created after the survey was scheduled
		return nil
	}

	if notice.Sent {
		// Already sent
		return nil
	}

	return notice
}

func isSystemAdmin(user *model.User) bool {
	for _, role := range strings.Fields(user.Roles) {
		if role == model.SYSTEM_ADMIN_ROLE_ID {
			return true
		}
	}

	return false
}

func (p *Plugin) sendAdminNoticeDM(user *model.User, notice *adminNotice) {
	p.API.LogDebug("Sending admin notice DM", "user_id", user.Id)

	// Send the DM
	if _, err := p.CreateBotDMPost(user.Id, p.buildAdminNoticePost(notice.NextSurvey)); err != nil {
		p.API.LogError("Failed to send admin notice", "err", err)
		return
	}

	// Store that the DM has been sent
	notice.Sent = true

	if err := p.KVSet(ADMIN_DM_NOTICE_KEY+user.Id, notice); err != nil {
		p.API.LogError("Failed to save sent admin notice. Admin notice will be resent on next refresh.", "err", err)
	}
}

func (p *Plugin) buildAdminNoticePost(nextSurvey time.Time) *model.Post {
	return &model.Post{
		Message: fmt.Sprintf(adminDMBody, nextSurvey.Format("January 2, 2006"), manifest.Id),
		Type:    "custom_nps_admin_notice",
	}
}

type surveyState struct {
	ServerVersion semver.Version
	SentAt        time.Time
	AnsweredAt    time.Time
	ScorePostId   string
}

func (p *Plugin) shouldSendSurveyDM(user *model.User, now time.Time) bool {
	if !p.getConfiguration().EnableSurvey {
		// Surveys are disabled
		return false
	}

	// Only send the survey once it has been 21 days since the last upgrade
	lastUpgrade, err := p.getLastServerUpgrade()
	if lastUpgrade == nil || err != nil {
		p.API.LogError("Failed to get date of last upgrade")
		return false
	}

	if now.Sub(lastUpgrade.Timestamp) < TIME_UNTIL_SURVEY {
		return false
	}

	// And the user has existed for at least as long
	if now.Sub(time.Unix(user.CreateAt/1000, 0)) < TIME_UNTIL_SURVEY {
		return false
	}

	// And that it has been long enough since the survey last occurred
	var state *surveyState
	if err := p.KVGet(USER_SURVEY_KEY+user.Id, &state); err != nil {
		p.API.LogError("Failed to get user survey state", "err", err)
		return false
	}

	if state == nil {
		// The user hasn't answered a survey before
		return true
	}

	if state.ServerVersion.Major == p.serverVersion.Major && state.ServerVersion.Minor == p.serverVersion.Minor {
		// Last survey occurred on the current version
		return false
	}

	if now.Sub(state.SentAt) < MIN_TIME_BETWEEN_USER_SURVEYS {
		// Not enough time since last survey was sent
		return false
	}

	if now.Sub(state.AnsweredAt) < MIN_TIME_BETWEEN_USER_SURVEYS {
		// Not enough time since last survey was completed
		return false
	}

	return true
}

func (p *Plugin) sendSurveyDM(user *model.User, now time.Time) {
	p.API.LogDebug("Sending survey DM", "user_id", user.Id)

	// Send the DM
	post, err := p.CreateBotDMPost(user.Id, p.buildSurveyPost(user))
	if err != nil {
		p.API.LogError("Failed to send survey", "err", err)
		return
	}

	// Store that the survey has been sent
	err = p.KVSet(USER_SURVEY_KEY+user.Id, &surveyState{
		ServerVersion: p.serverVersion,
		SentAt:        now,
		ScorePostId:   post.Id,
	})
	if err != nil {
		p.API.LogError("Failed to save sent survey state. Survey will be resent on next refresh.", "err", err)
	}
}

func (p *Plugin) buildSurveyPost(user *model.User) *model.Post {
	var options []*model.PostActionOptions
	for i := 10; i >= 0; i-- {
		text := strconv.Itoa(i)
		if i == 0 {
			text = "0 (Not Likely)"
		} else if i == 10 {
			text = "10 (Very Likely)"
		}

		options = append(options, &model.PostActionOptions{
			Text:  text,
			Value: strconv.Itoa(i),
		})
	}

	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	action := &model.PostAction{
		Name:    "Select an option...",
		Type:    model.POST_ACTION_TYPE_SELECT,
		Options: options,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/v1/score", siteURL, manifest.Id),
		},
	}

	return &model.Post{
		Message: fmt.Sprintf(surveyBody, user.Username),
		Type:    "custom_nps_survey",
		Props: map[string]interface{}{
			"attachments": []*model.SlackAttachment{
				{
					Title:   surveyDropdownTitle,
					Actions: []*model.PostAction{action},
				},
			},
		},
	}
}

func (p *Plugin) buildAnsweredSurveyPost(user *model.User, score int) *model.Post {
	return &model.Post{
		Type:    "custom_nps_survey",
		Message: fmt.Sprintf(surveyBody, user.Username),
		Props: map[string]interface{}{
			"attachments": []*model.SlackAttachment{
				{
					Title: surveyDropdownTitle,
					Text:  fmt.Sprintf(surveyAnsweredBody, score),
				},
			},
			"from_webhook": "true", // Needs to be manually specified since this doesn't go through CreateBotDMPost
		},
	}
}

func (p *Plugin) buildFeedbackRequestPost() *model.Post {
	return &model.Post{
		Type:    "custom_nps_feedback",
		Message: feedbackRequestBody,
	}
}

func (p *Plugin) markSurveyAnswered(userID string, now time.Time) *model.AppError {
	var state *surveyState
	if err := p.KVGet(USER_SURVEY_KEY+userID, &state); err != nil {
		return err
	}

	state.AnsweredAt = now

	return p.KVSet(USER_SURVEY_KEY+userID, &state)
}
