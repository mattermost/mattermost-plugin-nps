package main

import (
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetLastServerUpgrade(t *testing.T) {
	t.Run("nothing stored", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", SERVER_UPGRADE_KEY).Return(nil, nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		assert.Nil(t, p.getLastServerUpgrade())
	})

	t.Run("something stored", func(t *testing.T) {
		timestamp := time.Unix(1552599223, 0)

		api := &plugintest.API{}
		api.On("KVGet", SERVER_UPGRADE_KEY).Return([]byte(`{"Version": "5.10.0", "Timestamp": "2019-03-14T17:33:43-04:00"}`), nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		assert.Equal(t, &serverUpgrade{
			Version:   semver.MustParse("5.10.0"),
			Timestamp: timestamp,
		}, p.getLastServerUpgrade())
	})
}

func TestShouldScheduleSurvey(t *testing.T) {
	for _, test := range []struct {
		Name           string
		CurrentVersion semver.Version
		LastUpgrade    *serverUpgrade
		Expected       bool
	}{
		{
			Name:           "No previous version stored",
			CurrentVersion: semver.MustParse("5.10.0"),
			LastUpgrade:    nil,
			Expected:       true,
		},
		{
			Name:           "Stored version is different by a major version",
			CurrentVersion: semver.MustParse("5.10.0"),
			LastUpgrade:    &serverUpgrade{Version: semver.MustParse("4.10.0")},
			Expected:       true,
		},
		{
			Name:           "Stored version is different by a minor version",
			CurrentVersion: semver.MustParse("5.10.0"),
			LastUpgrade:    &serverUpgrade{Version: semver.MustParse("5.9.0")},
			Expected:       true,
		},
		{
			Name:           "Stored version is different by a patch version",
			CurrentVersion: semver.MustParse("5.10.1"),
			LastUpgrade:    &serverUpgrade{Version: semver.MustParse("5.10.0")},
			Expected:       false,
		},
		{
			Name:           "Stored version is the same",
			CurrentVersion: semver.MustParse("5.10.0"),
			LastUpgrade:    &serverUpgrade{Version: semver.MustParse("5.10.0")},
			Expected:       false,
		},
		{
			Name:           "Stored version is newer",
			CurrentVersion: semver.MustParse("4.10.0"),
			LastUpgrade:    &serverUpgrade{Version: semver.MustParse("5.10.0")},
			Expected:       false,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, shouldScheduleSurvey(test.CurrentVersion, test.LastUpgrade))
		})
	}
}

func TestShouldSendSurveyScheduledEmail(t *testing.T) {
	for _, test := range []struct {
		Name        string
		Now         time.Time
		LastUpgrade *serverUpgrade
		Expected    bool
	}{
		{
			Name:        "No previous time stored",
			Now:         time.Date(2009, time.November, 10, 23, 30, 0, 0, time.UTC),
			LastUpgrade: nil,
			Expected:    true,
		},
		{
			Name:        "Last survey sent on the same day",
			Now:         time.Date(2009, time.November, 10, 23, 30, 0, 0, time.UTC),
			LastUpgrade: &serverUpgrade{Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
			Expected:    false,
		},
		{
			Name:        "Last survey sent under one week ago",
			Now:         time.Date(2009, time.November, 17, 22, 59, 59, 999999, time.UTC),
			LastUpgrade: &serverUpgrade{Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
			Expected:    false,
		},
		{
			Name:        "Last survey sent one week ago",
			Now:         time.Date(2009, time.November, 17, 23, 0, 0, 0, time.UTC),
			LastUpgrade: &serverUpgrade{Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
			Expected:    true,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, shouldSendSurveyScheduledEmail(test.Now, test.LastUpgrade))
		})
	}
}

func TestSendSurveyScheduledEmail(t *testing.T) {
	email1 := "admin1@example.com"
	email2 := "admin2@example.com"
	inactiveEmail := "inactive@example.com"

	api := &plugintest.API{}
	api.On("GetUsers", &model.UserGetOptions{Page: 0, PerPage: 100, Role: "system_admin"}).Return([]*model.User{
		{
			Email: email1,
		},
		{
			Email: email2,
		},
		{
			Email:    inactiveEmail,
			DeleteAt: 1234,
		},
	}, nil)
	api.On("GetConfig").Return(&model.Config{
		ServiceSettings: model.ServiceSettings{
			SiteURL: model.NewString("https://mattermost.example.com"),
		},
		TeamSettings: model.TeamSettings{
			SiteName: model.NewString("SiteName"),
		},
		EmailSettings: model.EmailSettings{
			FeedbackOrganization: model.NewString(""),
		},
	})
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything)
	api.On("SendMail", email1, mock.Anything, mock.Anything).Return(nil)
	api.On("SendMail", email2, mock.Anything, mock.Anything).Return(nil)
	// No mock for inactiveEmail since it shouldn't be called
	defer api.AssertExpectations(t)

	p := Plugin{}
	p.SetAPI(api)

	p.sendSurveyScheduledEmail()
}

func TestGetAdminUsers(t *testing.T) {
	perPage := 3

	t.Run("less than 1 page of admin users", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetUsers", &model.UserGetOptions{Page: 0, PerPage: perPage, Role: "system_admin"}).Return([]*model.User{
			{
				Email: model.NewId(),
			},
			{
				Email: model.NewId(),
			},
		}, nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		received, err := p.getAdminUsers(perPage)

		assert.Nil(t, err)
		assert.Len(t, received, 2)
	})

	t.Run("more than 1 page of admin users", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetUsers", &model.UserGetOptions{Page: 0, PerPage: perPage, Role: "system_admin"}).Return([]*model.User{
			{
				Email: model.NewId(),
			},
			{
				Email: model.NewId(),
			},
			{
				Email: model.NewId(),
			},
		}, nil)
		api.On("GetUsers", &model.UserGetOptions{Page: 1, PerPage: perPage, Role: "system_admin"}).Return([]*model.User{
			{
				Email: model.NewId(),
			},
			{
				Email: model.NewId(),
			},
			{
				Email: model.NewId(),
			},
		}, nil)
		api.On("GetUsers", &model.UserGetOptions{Page: 2, PerPage: perPage, Role: "system_admin"}).Return([]*model.User{
			{
				Email: model.NewId(),
			},
			{
				Email: model.NewId(),
			},
		}, nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		received, err := p.getAdminUsers(perPage)

		assert.Nil(t, err)
		assert.Len(t, received, 8)
	})
}
