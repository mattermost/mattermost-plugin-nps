package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

type welcomeFeedbackMigration struct {
	CreateAt time.Time
}

// setWelcomeFeedbackMigration in called on plugin activation to set the time when the welcome feedback
// has been enabled on this server, preventing us to send the welcome feedback to older users
func (p *Plugin) setWelcomeFeedbackMigration(now time.Time) {
	var migration welcomeFeedbackMigration
	// set the current date in the KV store if it does not exist.
	if err := p.KVGet(WelcomeFeedbackMigrationKey, &migration); err != nil {
		p.API.LogError("Failed to get welcome feedback migration key", "err", err)
		return
	}

	if migration.CreateAt.IsZero() {
		migration = welcomeFeedbackMigration{CreateAt: now}
		p.API.LogInfo("Setting welcome feedback migration date", "date", now.String())
		if err := p.KVSet(WelcomeFeedbackMigrationKey, migration); err != nil {
			p.API.LogError("Failed to set welcome feedback migration key", "err", err)
			return
		}
	}

	p.welcomeFeedbackAfter = migration.CreateAt.Add(-TimeUntilWelcomeFeedback)
	p.API.LogDebug(fmt.Sprintf("Will send welcome feedback to users who joined after %s", p.welcomeFeedbackAfter.String()))
}

func (p *Plugin) checkForWelcomeFeedback(user *model.User, now time.Time) (bool, *model.AppError) {
	if !p.getConfiguration().EnableSurvey {
		return false, nil
	}

	// There probably was an error during the initialization
	if p.welcomeFeedbackAfter.IsZero() {
		return false, nil
	}

	createdAt := time.UnixMilli(user.CreateAt)
	// User created before welcome feedback time - they should never get the message
	if p.welcomeFeedbackAfter.After(createdAt) {
		return false, nil
	}

	// User has now reached the required time to get the welcome feedback
	if now.Before(createdAt.Add(TimeUntilWelcomeFeedback)) {
		return false, nil
	}

	var alreadySent bool
	if err := p.KVGet(fmt.Sprintf(UserWelcomeFeedbackKey, user.Id), &alreadySent); err != nil {
		return false, err
	}

	// DM has already been sent
	if alreadySent {
		return false, nil
	}

	return true, p.sendWelcomeFeedbackDM(user, now)
}

func (p *Plugin) sendWelcomeFeedbackDM(user *model.User, now time.Time) *model.AppError {
	p.API.LogDebug("Sending welcome feedback DM", "user_id", user.Id)

	// Send the DM
	_, err := p.CreateBotDMPost(user.Id, &model.Post{
		Message: fmt.Sprintf(welcomeFeedbackRequestBody, user.Username),
		Type:    "custom_nps_feedback",
	})
	if err != nil {
		return err
	}

	// Store that the welcome survey has been sent
	err = p.KVSet(fmt.Sprintf(UserWelcomeFeedbackKey, user.Id), true)
	if err != nil {
		p.API.LogError("Failed to save sent survey state. Survey will be resent on next refresh.", "err", err)
		return err
	}

	return nil
}
