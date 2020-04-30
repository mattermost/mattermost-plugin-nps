package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "nps-test",
		DisplayName:      "NPS Test",
		Description:      "Command for testing the NPS plugin",
		AutoComplete:     true,
		AutoCompleteDesc: "Available actions: next-survey",
		AutoCompleteHint: "[action]",
	}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	command, action, _ := parseCommand(commandArgs.Command)

	if command != "/nps-test" {
		return nil, nil
	}

	switch action {
	case "next-survey":
		return p.executeNextSurveyCommand()
	default:
		return &model.CommandResponse{Text: fmt.Sprint("Invalid action specified", action)}, nil
	}
}

func parseCommand(command string) (string, string, []string) {
	split := strings.Fields(command)

	switch len(split) {
	case 0:
		return "", "", nil
	case 1:
		return split[0], "", nil
	case 2:
		return split[0], split[1], nil
	default:
		return split[0], split[1], split[2:]
	}
}

func (p *Plugin) executeNextSurveyCommand() (*model.CommandResponse, *model.AppError) {
	var nextSurvey *surveyState
	if err := p.KVGet(fmt.Sprintf(SURVEY_KEY, p.serverVersion), &nextSurvey); err != nil {
		return nil, err
	}

	responseText := ""

	if nextSurvey != nil {
		surveyEnabled := p.getConfiguration().EnableSurvey
		surveyStarted := p.now().After(nextSurvey.StartAt)

		startAt := nextSurvey.StartAt.Format("15:04:05 MST on Monday January 2, 2006 ")

		if surveyEnabled && surveyStarted {
			responseText = fmt.Sprintf("Survey started at %s", startAt)
		} else if surveyEnabled {
			responseText = fmt.Sprintf("Survey scheduled to start at %s", startAt)
		} else {
			responseText = fmt.Sprintf("Survey disabled, but it was scheduled to start at %s", startAt)
		}
	} else {
		responseText = "No survey scheduled"
	}

	return &model.CommandResponse{Text: responseText}, nil
}
