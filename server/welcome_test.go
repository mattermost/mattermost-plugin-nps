// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSetWelcomeFeedbackMigration(t *testing.T) {
	p := &Plugin{}
	assert := require.New(t)

	testNow := time.Date(2022, time.November, 15, 10, 45, 0, 0, time.UTC)
	testGetAlreadySet := time.Date(2021, time.November, 15, 10, 45, 0, 0, time.UTC)

	makeAPIMock := func() *plugintest.API {
		api := &plugintest.API{}
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogInfo", mock.Anything).Maybe()
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything).Maybe()
		return api
	}

	t.Run("when the feedback migration key is found, the value must be set based on the stored value", func(t *testing.T) {
		apiMock := makeAPIMock()
		apiMock.On("KVGet", WelcomeFeedbackMigrationKey).Return(mustMarshalJSON(welcomeFeedbackMigration{CreateAt: testGetAlreadySet}), nil)
		p.API = apiMock

		p.setWelcomeFeedbackMigration(testNow)

		assert.Equal(testGetAlreadySet.Add(-TimeUntilWelcomeFeedback), p.welcomeFeedbackAfter)
	})

	t.Run("When the feedback migration key is not found, the value must be stored and set on based the argument", func(t *testing.T) {
		apiMock := makeAPIMock()
		apiMock.On("KVGet", WelcomeFeedbackMigrationKey).Return(nil, nil)
		apiMock.On("KVSet", WelcomeFeedbackMigrationKey, mustMarshalJSON(welcomeFeedbackMigration{CreateAt: testNow})).Return(nil)
		p.API = apiMock

		p.setWelcomeFeedbackMigration(testNow)

		assert.Equal(testNow.Add(-TimeUntilWelcomeFeedback), p.welcomeFeedbackAfter)
	})
}

func TestCheckForWelcomeFeedback(t *testing.T) {
	p := &Plugin{
		configuration: &configuration{},
		botUserID:     "botUserID",
	}
	assert := require.New(t)

	testNow := time.Date(2022, time.November, 15, 10, 45, 0, 0, time.UTC)

	makeAPIMock := func() *plugintest.API {
		api := &plugintest.API{}
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogInfo", mock.Anything).Maybe()
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything).Maybe()
		return api
	}

	testCases := []struct {
		Name          string
		EnableSurvey  bool
		UserID        string
		UserCreatedAt time.Time
		FeedbackAfter time.Time
		SetupMock     func(*plugintest.API)
		AlreadySent   bool
		MessageSent   bool
	}{
		{
			Name:         "When the survey is disabled, no message should be sent",
			EnableSurvey: false,
			MessageSent:  false,
		},
		{
			Name:          "When the user was created before the feedback time, no message should be sent",
			EnableSurvey:  true,
			UserCreatedAt: testNow,
			FeedbackAfter: testNow.Add(time.Second),
			MessageSent:   false,
		},
		{
			Name:          "When the user was created after the feedback time but less than the minimum time required, no message should be sent",
			EnableSurvey:  true,
			UserCreatedAt: testNow.Add(TimeUntilWelcomeFeedback - time.Minute),
			FeedbackAfter: testNow,
			MessageSent:   false,
		},
		{
			Name:          "When the message has already been sent, don't send it again",
			EnableSurvey:  true,
			UserID:        "testAlreadySent",
			UserCreatedAt: testNow.Add(-TimeUntilWelcomeFeedback - time.Minute),
			FeedbackAfter: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			SetupMock: func(api *plugintest.API) {
				api.On("KVGet", "UserWelcomeFeedback-testAlreadySent").Return(mustMarshalJSON(true), nil)
			},
			MessageSent: false,
		},
		{
			Name:          "When the message has not already been sent, send it!",
			EnableSurvey:  true,
			UserID:        "testNotAlreadySent",
			UserCreatedAt: testNow.Add(-TimeUntilWelcomeFeedback - time.Minute),
			FeedbackAfter: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			SetupMock: func(api *plugintest.API) {
				// Check if the message has already been sent
				api.On("KVGet", "UserWelcomeFeedback-testNotAlreadySent").Return(mustMarshalJSON(false), nil)

				// Getting the DM channel
				api.On("GetDirectChannel", "testNotAlreadySent", p.botUserID).Return(&model.Channel{Id: "testChannelID"}, nil)
				// Create the post
				api.On("CreatePost", mock.AnythingOfType("*model.Post")).Return(&model.Post{}, nil)
				// Store that the message has been sent
				api.On("KVSet", "UserWelcomeFeedback-testNotAlreadySent", mustMarshalJSON(true)).Return(nil)
			},
			MessageSent: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			p.welcomeFeedbackAfter = tc.FeedbackAfter
			p.configuration.EnableSurvey = tc.EnableSurvey

			apiMock := makeAPIMock()
			if tc.SetupMock != nil {
				tc.SetupMock(apiMock)
			}
			p.API = apiMock

			sent, _ := p.checkForWelcomeFeedback(&model.User{
				Id:       tc.UserID,
				CreateAt: tc.UserCreatedAt.UnixMilli(),
			}, testNow)

			assert.Equal(tc.MessageSent, sent)
		})
	}
}
