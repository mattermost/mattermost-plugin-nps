package main

import (
	"path/filepath"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

func (p *Plugin) OnActivate() error {
	p.API.LogDebug("Activating plugin")

	if !p.canSendDiagnostics() {
		errMsg := "Not activating plugin because diagnostics are disabled"
		p.API.LogError(errMsg)
		return errors.New(errMsg)
	}

	botUserID, appErr := p.ensureBotExists()
	if appErr != nil {
		return errors.Wrap(appErr, "Failed to ensure bot user exists")
	}
	p.botUserID = botUserID

	p.serverVersion = getServerVersion(p.API.GetServerVersion())

	if err := p.initializeClient(); err != nil {
		p.API.LogError("Failed to initialize Rudder client", "err", err.Error())
		return err
	}

	now := p.now().UTC()

	if err := p.clearStaleLocks(now); err != nil {
		return err
	}

	p.API.LogDebug("Plugin activated")

	p.setActivated(true)

	if upgraded, appErr := p.checkForServerUpgrade(now); appErr != nil {
		return appErr
	} else if upgraded {
		p.API.LogInfo("Upgrade detected. Checking if a survey should be scheduled.")

		go p.checkForNextSurvey(now)
	}

	return nil
}

func (p *Plugin) OnDeactivate() error {
	if p.client != nil {
		err := p.client.Close()
		if err != nil {
			p.API.LogWarn("OnDeactivate: Failed to close telemetryClient", "error", err.Error())
		}
	}
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

	user, err := p.API.GetUserByUsername("surveybot")
	if err != nil || user == nil {
		p.API.LogDebug("Failed to find the bot, maybe does not exist, wuill try to create it", "err", err)

		bot, createErr := p.API.CreateBot(&model.Bot{
			Username:    "surveybot",
			DisplayName: "Surveybot",
			Description: SurveybotDescription,
		})
		if createErr != nil {
			p.API.LogError("Failed to create the bot", "err", createErr)
			return "", err
		}

		if profileErr := p.setBotProfileImage(bot.UserId); profileErr != nil {
			p.API.LogWarn("Failed to set profile image for bot", "err", profileErr)
		}

		p.API.LogInfo("Surveybot created")
		return bot.UserId, nil
	}

	bot, err := p.API.GetBot(user.Id, true)
	if err != nil {
		p.API.LogError("Failed to find Surveybot", "err", err)
		return "", err
	}

	p.API.LogDebug("Found Surveybot")
	return bot.UserId, nil
}

func (p *Plugin) setBotProfileImage(botUserID string) *model.AppError {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return &model.AppError{Message: err.Error()}
	}

	profileImage, err := p.readFile(filepath.Join(bundlePath, "assets", "icon-happy-bot-square@1x.png"))
	if err != nil {
		return &model.AppError{Message: err.Error()}
	}

	return p.API.SetProfileImage(botUserID, profileImage)
}
