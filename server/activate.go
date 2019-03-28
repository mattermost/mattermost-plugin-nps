package main

import (
	"io/ioutil"
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

	if err := p.ensureBotExists(); err != nil {
		return errors.Wrap(err, "failed to ensure bot user exists")
	}

	p.initializeClient()

	p.registerCommands()

	p.setActivated(true)
	p.API.LogDebug("NPS plugin activated")

	go p.checkForNextSurvey(p.serverVersion)

	return nil
}

func (p *Plugin) setActivated(activated bool) {
	p.activatedLock.Lock()
	defer p.activatedLock.Unlock()

	p.activated = activated
}

func (p *Plugin) isActivated() bool {
	p.activatedLock.RLock()
	defer p.activatedLock.RUnlock()

	return p.activated
}

func (p *Plugin) canSendDiagnostics() bool {
	enableDiagnostics := p.API.GetConfig().LogSettings.EnableDiagnostics
	return enableDiagnostics != nil && *enableDiagnostics
}

func (p *Plugin) ensureBotExists() *model.AppError {
	// Attempt to find an existing bot
	var botUserId string
	appErr := p.KVGet(BOT_USER_KEY, &botUserId)
	if appErr != nil {
		return appErr
	}

	if botUserId != "" {
		// Bot already exists
		p.botUserId = botUserId
		return nil
	}

	var bot *model.Bot

	if user, err := p.API.GetUserByUsername("surveybot"); err == nil && user != nil {
		// A surveybot user exists, so try to reclaim the existing bot account
		p.API.LogDebug("Finding existing bot account")

		bot, err = p.API.GetBot(user.Id, true)
		if err != nil {
			return err
		}

		p.API.LogDebug("Found existing bot account")
	} else {
		// Create a bot since one doesn't exist
		p.API.LogDebug("Creating bot for NPS plugin")

		bundlePath, err := p.API.GetBundlePath()
		if err != nil {
			return &model.AppError{Message: "Failed to get bundle path"}
		}

		profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "icon-happy-bot-square@1x.png"))
		if err != nil {
			return &model.AppError{Message: "Failed to read profile image"}
		}

		bot, appErr = p.API.CreateBot(&model.Bot{
			Username:    "surveybot",
			DisplayName: "Surveybot",
			Description: SURVEYBOT_DESCRIPTION,
		})
		if appErr != nil {
			return appErr
		}

		// Give it a profile picture
		if appErr = p.API.SetProfileImage(bot.UserId, profileImage); appErr != nil {
			p.API.LogWarn("Failed to set profile image for bot", "error", appErr)
		}

		p.API.LogDebug("Bot created for NPS plugin")
	}

	// Save the bot ID
	if appErr := p.KVSet(BOT_USER_KEY, bot.UserId); appErr != nil {
		return appErr
	}

	p.botUserId = bot.UserId

	return nil
}
