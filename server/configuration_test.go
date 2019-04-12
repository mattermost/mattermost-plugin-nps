package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
