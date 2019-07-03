package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
)

const (
	// How often "survey scheduled" emails can be sent to prevent multiple emails from being sent if multiple server
	// upgrades occur within a short time
	MIN_TIME_BETWEEN_SURVEY_EMAILS = 7 * time.Minute

	// How long until a survey occurs after a server upgrade in days (for use in notifications)
	DAYS_UNTIL_SURVEY = 21

	// How long until a survey occurs after a server upgrade as a time.Duration
	TIME_UNTIL_SURVEY = 21 * time.Minute

	// Get admin users up to 100 at a time when sending email notifications
	ADMIN_USERS_PER_PAGE = 100

	// The minimum time before a user can be sent a survey after completing the previous one
	MIN_TIME_BETWEEN_USER_SURVEYS = 30 * time.Minute
)

type adminNotice struct {
	Sent          bool      `json:"sent"`
	ServerVersion string    `json:"server_version"`
	SurveyStartAt time.Time `json:"survey_start_at"`
}

type surveyState struct {
	ServerVersion string    `json:"server_version"`
	CreateAt      time.Time `json:"create_at"`
	StartAt       time.Time `json:"start_at"`
}

type userSurveyState struct {
	ServerVersion string    `json:"server_version"`
	SentAt        time.Time `json:"sent_at"`
	AnsweredAt    time.Time `json:"answered_at"`
	ScorePostId   string    `json:"score_post_id"`
}

// checkForNextSurvey schedules a new NPS survey if a major or minor version change has occurred. Returns whether or
// not a survey was scheduled.
//
// Note that this only sends an email to admins to notify them that a survey has been scheduled. The web app plugin is
// in charge of checking and actually triggering the survey.
func (p *Plugin) checkForNextSurvey(now time.Time) bool {
	if !p.getConfiguration().EnableSurvey {
		// Surveys are disabled, so return false without updating the stored version. If surveys are re-enabled, the
		// plugin will then detect an upgrade (if one occurred) and schedule the next survey.
		p.API.LogInfo("Not sending NPS survey because survey is disabled")
		return false
	}

	locked, err := p.tryLock(LOCK_KEY, now)
	if !locked || err != nil {
		// Either an error occurred or there's already another thread checking for surveys
		return false
	}
	defer p.unlock(LOCK_KEY)

	var nextSurvey *surveyState
	if err := p.KVGet(fmt.Sprintf(SURVEY_KEY, p.serverVersion), &nextSurvey); err != nil {
		p.API.LogError("Failed to get survey state", "err", err)
		return false
	}

	if nextSurvey != nil {
		p.API.LogInfo(fmt.Sprintf("Survey already scheduled for %s", nextSurvey.StartAt.Format("Jan 2, 2006")))
		return false
	}

	nextSurvey = &surveyState{
		ServerVersion: p.serverVersion,
		CreateAt:      now,
		StartAt:       now.Add(TIME_UNTIL_SURVEY),
	}

	p.API.LogInfo(fmt.Sprintf("Scheduling next survey for %s", nextSurvey.StartAt.Format("Jan 2, 2006")))

	if err := p.KVSet(fmt.Sprintf(SURVEY_KEY, p.serverVersion), nextSurvey); err != nil {
		p.API.LogError("Failed to schedule next survey", "err", err)
		return false
	}

	if sent, err := p.sendAdminNotices(now, nextSurvey); err != nil {
		p.API.LogError("Failed to send notification of next survey to admins", "err", err)
		return false
	} else if !sent {
		p.API.LogInfo("Not sending notification of next survey to admins since they already received one recently")
	} else {
		p.API.LogInfo("Sent notification of next survey to admins")
	}

	return true
}

func (p *Plugin) sendAdminNotices(now time.Time, nextSurvey *surveyState) (bool, error) {
	var lastSentAt *time.Time
	if err := p.KVGet(LAST_ADMIN_NOTICE_KEY, &lastSentAt); err != nil {
		return false, err
	}

	if lastSentAt != nil && now.Sub(*lastSentAt) < MIN_TIME_BETWEEN_SURVEY_EMAILS {
		// Not enough time has passed since the last survey notification, so don't send a new one
		return false, nil
	}

	admins, err := p.getAdminUsers(ADMIN_USERS_PER_PAGE)
	if err != nil {
		return false, err
	}

	p.sendAdminNoticeEmails(admins)
	p.sendAdminNoticeDMs(admins, nextSurvey)

	if err := p.KVSet(LAST_ADMIN_NOTICE_KEY, now); err != nil {
		return false, err
	}

	return true, nil
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

func (p *Plugin) sendAdminNoticeDMs(admins []*model.User, nextSurvey *surveyState) {
	// Actual DMs will be sent when the admins next log in, so just mark that they're scheduled to receive one
	for _, admin := range admins {
		err := p.KVSet(fmt.Sprintf(ADMIN_DM_NOTICE_KEY, admin.Id, nextSurvey.ServerVersion), &adminNotice{
			Sent:          false,
			ServerVersion: nextSurvey.ServerVersion,
			SurveyStartAt: nextSurvey.StartAt,
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

func (p *Plugin) checkForAdminNoticeDM(user *model.User) (bool, *model.AppError) {
	if !p.getConfiguration().EnableSurvey {
		// Surveys are disabled
		return false, nil
	}

	if !isSystemAdmin(user) {
		return false, nil
	}

	var notice *adminNotice
	if err := p.KVGet(fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, p.serverVersion), &notice); err != nil {
		return false, err
	}

	if notice == nil {
		// No notice stored for this user, likely because they were created after the survey was scheduled
		return false, nil
	}

	if notice.Sent {
		// Already sent
		return false, nil
	}

	return true, p.sendAdminNoticeDM(user, notice)
}

func isSystemAdmin(user *model.User) bool {
	for _, role := range strings.Fields(user.Roles) {
		if role == model.SYSTEM_ADMIN_ROLE_ID {
			return true
		}
	}

	return false
}

func (p *Plugin) sendAdminNoticeDM(user *model.User, notice *adminNotice) *model.AppError {
	p.API.LogDebug("Sending admin notice DM", "user_id", user.Id)

	// Send the DM
	if _, err := p.CreateBotDMPost(user.Id, p.buildAdminNoticePost(notice.SurveyStartAt)); err != nil {
		return err
	}

	// Store that the DM has been sent
	notice.Sent = true

	if err := p.KVSet(fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, notice.ServerVersion), notice); err != nil {
		p.API.LogError("Failed to save sent admin notice. Admin notice will be resent on next refresh.", "err", err)
		return err
	}

	return nil
}

func (p *Plugin) buildAdminNoticePost(surveyStartAt time.Time) *model.Post {
	return &model.Post{
		Message: fmt.Sprintf(adminDMBody, surveyStartAt.Format("January 2, 2006"), manifest.Id),
		Type:    "custom_nps_admin_notice",
	}
}

func (p *Plugin) checkForSurveyDM(user *model.User, now time.Time) (bool, *model.AppError) {
	if !p.getConfiguration().EnableSurvey {
		// Surveys are disabled
		return false, nil
	}

	if now.Sub(time.Unix(user.CreateAt/1000, 0)) < TIME_UNTIL_SURVEY {
		// The user hasn't existed for long enough to receive a survey
		return false, nil
	}

	var survey *surveyState
	if err := p.KVGet(fmt.Sprintf(SURVEY_KEY, p.serverVersion), &survey); err != nil {
		return false, err
	}

	if survey == nil {
		// No survey scheduled
		return false, nil
	}

	if now.Before(survey.StartAt) {
		// Survey hasn't started yet
		return false, nil
	}

	// And that it has been long enough since the survey last occurred
	var userSurvey *userSurveyState
	if err := p.KVGet(fmt.Sprintf(USER_SURVEY_KEY, user.Id), &userSurvey); err != nil {
		return false, err
	}

	if userSurvey != nil {
		if userSurvey.ServerVersion == p.serverVersion {
			// The user has already received this survey
			return false, nil
		}

		if now.Sub(userSurvey.SentAt) < MIN_TIME_BETWEEN_USER_SURVEYS {
			// Not enough time has passed since the user was last sent a survey
			return false, nil
		}

		if now.Sub(userSurvey.AnsweredAt) < MIN_TIME_BETWEEN_USER_SURVEYS {
			// Not enough time has passed since the user last completed a survey
			return false, nil
		}
	}

	return true, p.sendSurveyDM(user, now)
}

func (p *Plugin) sendSurveyDM(user *model.User, now time.Time) *model.AppError {
	p.API.LogDebug("Sending survey DM", "user_id", user.Id)

	// Send the DM
	post, err := p.CreateBotDMPost(user.Id, p.buildSurveyPost(user))
	if err != nil {
		return err
	}

	userSurveyState := &userSurveyState{
		ServerVersion: p.serverVersion,
		SentAt:        now,
		ScorePostId:   post.Id,
	}

	// Store that the survey has been sent
	err = p.KVSet(fmt.Sprintf(USER_SURVEY_KEY, user.Id), userSurveyState)
	if err != nil {
		p.API.LogError("Failed to save sent survey state. Survey will be resent on next refresh.", "err", err)
		return err
	}

	return nil
}

func (p *Plugin) buildSurveyPost(user *model.User) *model.Post {
	return &model.Post{
		Message: fmt.Sprintf(surveyBody, user.Username),
		Type:    "custom_nps_survey",
		Props: map[string]interface{}{
			"attachments": []*model.SlackAttachment{
				{
					Title:   surveyDropdownTitle,
					Actions: []*model.PostAction{p.buildSurveyPostAction()},
				},
			},
		},
	}
}

func (p *Plugin) buildSurveyPostAction() *model.PostAction {
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

	return &model.PostAction{
		Name:    "Select an option...",
		Type:    model.POST_ACTION_TYPE_SELECT,
		Options: options,
		Integration: &model.PostActionIntegration{
			URL: fmt.Sprintf("%s/plugins/%s/api/v1/score", siteURL, manifest.Id),
		},
	}
}

func (p *Plugin) buildAnsweredSurveyPost(user *model.User, score int) *model.Post {
	action := p.buildSurveyPostAction()
	action.DefaultOption = strconv.Itoa(score)

	return &model.Post{
		Type:    "custom_nps_survey",
		Message: fmt.Sprintf(surveyBody, user.Username),
		Props: map[string]interface{}{
			"attachments": []*model.SlackAttachment{
				{
					Title:   surveyDropdownTitle,
					Text:    fmt.Sprintf(surveyAnsweredBody, score),
					Actions: []*model.PostAction{action},
				},
			},
		},
	}
}

func (p *Plugin) buildFeedbackRequestPost() *model.Post {
	return &model.Post{
		Type:    "custom_nps_feedback",
		Message: feedbackRequestBody,
	}
}

func (p *Plugin) markSurveyAnswered(userID string, now time.Time) (bool, *model.AppError) {
	var userSurvey *userSurveyState
	if err := p.KVGet(fmt.Sprintf(USER_SURVEY_KEY, userID), &userSurvey); err != nil {
		return false, err
	}

	if !userSurvey.AnsweredAt.IsZero() {
		// Survey was already answered
		return false, nil
	}

	userSurvey.AnsweredAt = now

	if err := p.KVSet(fmt.Sprintf(USER_SURVEY_KEY, userID), userSurvey); err != nil {
		return false, err
	}

	return true, nil
}
