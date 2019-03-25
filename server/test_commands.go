package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	// ENABLE_TEST_COMMANDS can be set to true at compile time to enable some additional testing commands.
	ENABLE_TEST_COMMANDS = false

	COMMAND_TEST         = "nps-test"
	COMMAND_TEST_RESET   = "reset"
	COMMAND_TEST_VERSION = "version"
)

func (p *Plugin) registerCommands() {
	if ENABLE_TEST_COMMANDS {
		p.API.RegisterCommand(&model.Command{
			Trigger: COMMAND_TEST,
		})
	}
}

func parseCommandArgs(commandArgs *model.CommandArgs) (string, string, []string) {
	split := strings.Fields(commandArgs.Command)

	command := ""
	if len(split) > 0 {
		command = split[0]

		// Trim the leading slash
		command = command[1:]
	}

	action := ""
	if len(split) > 1 {
		action = split[1]
	}

	var args []string
	if len(split) > 2 {
		args = split[2:]
	}

	return command, action, args
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	command, action, args := parseCommandArgs(commandArgs)

	if ENABLE_TEST_COMMANDS && command == COMMAND_TEST {
		return p.executeTestCommand(action, args)
	}

	return nil, nil
}

func (p *Plugin) executeTestCommand(action string, args []string) (*model.CommandResponse, *model.AppError) {
	if action == "" {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "No action specified.",
		}, nil
	}

	commands := map[string]func([]string) (*model.CommandResponse, *model.AppError){
		COMMAND_TEST_RESET:   p.executeTestResetCommand,
		COMMAND_TEST_VERSION: p.executeTestVersionCommand,
	}

	command, ok := commands[action]
	if !ok {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Incorrect action specified.",
		}, nil
	}

	return command(args)
}

func (p *Plugin) executeTestResetCommand(args []string) (*model.CommandResponse, *model.AppError) {
	if err := p.API.KVDeleteAll(); err != nil {
		p.API.LogError("Failed to reset plugin state", "err", err)
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Failed to reset plugin state. See log for more details.",
		}, nil
	}

	if err := p.API.KVSet(BOT_USER_KEY, []byte(p.botUserId)); err != nil {
		p.API.LogError("Failed to restore bot user ID after resetting plugin state", "err", err)
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Failed to re-add bot user ID after resetting plugin state. This will likely render the plugin inoperable. See log for more details.",
		}, nil
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         "NPS plugin reset. Please re-enable it from the system console to continue testing.",
	}, nil
}

func (p *Plugin) executeTestVersionCommand(args []string) (*model.CommandResponse, *model.AppError) {
	if len(args) == 0 || len(args) > 2 {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Wrong number of params specified. Expected `" + COMMAND_TEST + " " + COMMAND_TEST_RESET + " <version> [<unix timestamp of upgrade>]`.",
		}, nil
	}

	version := semver.MustParse(args[0])

	var timestamp time.Time
	if len(args) == 2 {
		t, _ := strconv.ParseInt(args[1], 10, 64)
		timestamp = time.Unix(t, 0).UTC()
	} else {
		timestamp = time.Now().UTC()
	}

	if err := p.storeServerUpgrade(&serverUpgrade{
		Version:   version,
		Timestamp: timestamp,
	}); err != nil {
		p.API.LogError("Failed to store server version", "err", err)
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         "Failed to store server version. See log for more details.",
		}, nil
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("Stored server version set to %s as if the server was upgraded at %s. Disable and re-enable the plugin to simulate upgrade or refresh to attempt to trigger a survey.", version, timestamp),
	}, nil
}
