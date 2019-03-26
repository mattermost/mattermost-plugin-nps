package main

import (
	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

var minimumServerVersion = semver.MustParse("5.9.0") // TODO change this to 5.10.0

func checkMinimumVersion(serverVersion semver.Version) error {
	if serverVersion.LT(minimumServerVersion) {
		return errors.New("NPS plugin can only be ran on Mattermost 5.10.0 or higher")
	}

	return nil
}

func (p *Plugin) OnActivate() error {
	p.API.LogDebug("Activating NPS plugin")

	if !p.canSendDiagnostics() {
		p.API.LogDebug("Not activating NPS plugin because diagnostics are disabled")
		return nil
	}

	if serverVersion, err := semver.Parse(p.API.GetServerVersion()); err != nil {
		return errors.Wrap(err, "failed to parse server version")
	} else {
		p.serverVersion = serverVersion
	}

	if err := checkMinimumVersion(p.serverVersion); err != nil {
		return errors.Wrap(err, "failed to check minimum server version")
	}

	if err := p.ensureBotExists(); err != nil {
		return errors.Wrap(err, "failed to ensure bot user exists")
	}

	p.initializeClient()

	p.API.LogDebug("NPS plugin activated")

	p.checkForNextSurvey(p.serverVersion)

	return nil
}

func (p *Plugin) canSendDiagnostics() bool {
	enableDiagnostics := p.API.GetConfig().LogSettings.EnableDiagnostics
	return enableDiagnostics != nil && *enableDiagnostics
}

func (p *Plugin) ensureBotExists() error {
	// Attempt to find an existing bot
	botUserIdBytes, err := p.API.KVGet(BOT_USER_KEY)
	if err != nil {
		return err
	}

	if botUserIdBytes == nil {
		// Create a bot since one doesn't exist
		p.API.LogDebug("Creating bot for NPS plugin")

		bot, err := p.API.CreateBot(&model.Bot{
			Username:    "surveybot",
			DisplayName: "Surveybot",
			Description: SURVEYBOT_DESCRIPTION,
		})
		if err != nil {
			return err
		}

		// Give it a profile picture
		err = p.API.SetProfileImage(bot.UserId, profileImage)
		if err != nil {
			p.API.LogError("Failed to set profile image for bot", "err", err)
		}

		p.API.LogDebug("Bot created for NPS plugin")

		// Save the bot ID
		err = p.API.KVSet(BOT_USER_KEY, []byte(bot.UserId))
		if err != nil {
			return err
		}

		p.botUserId = bot.UserId
	} else {
		p.botUserId = string(botUserIdBytes)
	}

	return nil
}
