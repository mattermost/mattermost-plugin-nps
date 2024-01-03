package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHasSurveyBeenEnabled(t *testing.T) {
	for _, test := range []struct {
		Name            string
		Activated       bool
		NewEnableSurvey bool
		OldEnableSurvey bool
		Expected        bool
	}{
		{
			Name:            "survey has been enabled",
			Activated:       true,
			NewEnableSurvey: true,
			OldEnableSurvey: false,
			Expected:        true,
		},
		{
			Name:            "survey has been enabled, but plugin has not been activated yet",
			Activated:       false,
			NewEnableSurvey: true,
			OldEnableSurvey: false,
			Expected:        false,
		},
		{
			Name:            "survey was already enabled",
			Activated:       true,
			NewEnableSurvey: true,
			OldEnableSurvey: true,
			Expected:        false,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := &Plugin{
				activated: test.Activated,
			}

			new := &configuration{EnableSurvey: test.NewEnableSurvey}
			old := &configuration{EnableSurvey: test.OldEnableSurvey}

			assert.Equal(t, test.Expected, p.hasSurveyBeenEnabled(new, old))
		})
	}
}

func TestOnConfigurationChanged(t *testing.T) {
	api := makeAPIMock()
	api.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Run(func(args mock.Arguments) {
		*args.Get(0).(*configuration) = configuration{
			EnableSurvey: false,
		}
	}).Return(nil)
	api.On("GetConfig").Return(&model.Config{})
	api.On("GetDiagnosticId").Return("diagnosticID")
	api.On("GetServerVersion").Return("v7.6") // aka the lost version

	p := &Plugin{
		configuration: &configuration{
			EnableSurvey: true,
		},
		MattermostPlugin: plugin.MattermostPlugin{
			API: api,
		},
	}

	err := p.OnConfigurationChange()
	require.NoError(t, err)
	require.False(t, p.configuration.EnableSurvey)
}
