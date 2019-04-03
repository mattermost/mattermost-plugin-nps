package main

import (
	"path/filepath"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

func (p *Plugin) OnActivate() error {
	p.API.LogDebug("Activating NPS plugin")

	if !p.canSendDiagnostics() {
		errMsg := "Not activating NPS plugin because diagnostics are disabled"
		p.API.LogError(errMsg)
		return errors.New(errMsg)
	}

	if serverVersion, err := semver.Parse(p.API.GetServerVersion()); err != nil {
		return errors.Wrap(err, "failed to parse server version")
	} else {
		p.serverVersion = serverVersion
	}

	botUserId, err := p.ensureBotExists()
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot user exists")
	}
	p.botUserId = botUserId

	p.initializeClient()

	p.setActivated(true)
	p.API.LogDebug("NPS plugin activated")

	go p.checkForNextSurvey(p.serverVersion)

	return nil
}

func (p *Plugin) setActivated(activated bool) {
	p.activated = activated
}

func (p *Plugin) isActivated() bool {
	return p.activated
}

func (p *Plugin) canSendDiagnostics() bool {
	enableDiagnostics := p.API.GetConfig().LogSettings.EnableDiagnostics
	return enableDiagnostics != nil && *enableDiagnostics
}

func (p *Plugin) ensureBotExists() (string, *model.AppError) {
	p.API.LogInfo("Ensuring Surveybot exists")

	bot, createErr := p.API.CreateBot(&model.Bot{
		Username:    "surveybot",
		DisplayName: "Surveybot",
		Description: SURVEYBOT_DESCRIPTION,
	})
	if createErr != nil {
		p.API.LogDebug("Failed to create Surveybot. Attempting to find existing one.", "err", createErr)

		// Unable to create the bot, so it should already exist
		user, err := p.API.GetUserByUsername("surveybot")
		if err != nil || user == nil {
			p.API.LogError("Failed to find Surveybot user", "err", err)
			return "", err
		}

		bot, err = p.API.GetBot(user.Id, true)
		if err != nil {
			p.API.LogError("Failed to find Surveybot", "err", err)
			return "", err
		}

		p.API.LogDebug("Found Surveybot")
	} else {
		if err := p.setBotProfileImage(bot.UserId); err != nil {
			p.API.LogWarn("Failed to set profile image for bot", "err", err)
		}

		p.API.LogInfo("Surveybot created")
	}

	return bot.UserId, nil
}

func (p *Plugin) setBotProfileImage(botUserId string) *model.AppError {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return &model.AppError{Message: err.Error()}
	}

	profileImage, err := p.readFile(filepath.Join(bundlePath, "assets", "icon-happy-bot-square@1x.png"))
	if err != nil {
		return &model.AppError{Message: err.Error()}
	}

	return p.API.SetProfileImage(botUserId, profileImage)
}
