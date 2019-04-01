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

	p.registerCommands()

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
	// Attempt to find an existing bot
	var botUserId string
	if appErr := p.KVGet(BOT_USER_KEY, &botUserId); appErr != nil {
		return "", appErr
	}

	if botUserId != "" {
		// Bot already exists
		return botUserId, nil
	}

	// Create a bot since one doesn't exist
	p.API.LogInfo("Creating bot for NPS plugin")

	bot, appErr := p.API.CreateBot(&model.Bot{
		Username:    "surveybot",
		DisplayName: "Surveybot",
		Description: SURVEYBOT_DESCRIPTION,
	})
	if appErr != nil {
		// Unable to create the bot, so it may already exist and need to be reclaimed
		p.API.LogDebug("Failed to create bot for NPS plugin. Attempting to reclaim existing bot.")

		user, err := p.API.GetUserByUsername("surveybot")
		if err != nil || user == nil {
			return "", err
		}

		// A surveybot user exists, so try to reclaim the existing bot account
		p.API.LogDebug("Found surveybot user. Attempting to find matching bot account.")

		bot, err = p.API.GetBot(user.Id, true)
		if err != nil {
			return "", err
		}

		p.API.LogInfo("Found existing bot account")
	} else {
		// Give the newly created bot a profile picture
		bundlePath, err := p.API.GetBundlePath()
		if err != nil {
			return "", &model.AppError{Message: "Failed to get bundle path"}
		}

		profileImage, err := p.readFile(filepath.Join(bundlePath, "assets", "icon-happy-bot-square@1x.png"))
		if err != nil {
			return "", &model.AppError{Message: "Failed to read profile image"}
		}

		// Give it a profile picture
		if appErr = p.API.SetProfileImage(bot.UserId, profileImage); appErr != nil {
			p.API.LogWarn("Failed to set profile image for bot", "err", appErr)
		}

		p.API.LogInfo("Bot created for NPS plugin")
	}

	// Save the bot ID
	if appErr := p.KVSet(BOT_USER_KEY, bot.UserId); appErr != nil {
		return "", appErr
	}

	return bot.UserId, nil
}
