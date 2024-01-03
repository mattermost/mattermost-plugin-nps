package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOnActivate(t *testing.T) {
	botUserID := model.NewId()
	now := toDate(2019, time.March, 1)
	serverVersion := "5.10.0"

	makeAPIMock := func() *plugintest.API {
		api := &plugintest.API{}
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogInfo", mock.Anything).Maybe()
		return api
	}

	t.Run("should set up Plugin correctly", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("GetUserByUsername", "feedbackbot").Return(&model.User{Id: botUserID}, nil)
		api.On("GetBot", botUserID, true).Return(&model.Bot{UserId: botUserID}, nil)
		api.On("GetServerVersion").Return(serverVersion)
		api.On("KVList", 0, 100).Return([]string{}, nil)
		api.On("KVGet", fmt.Sprintf(ServerUpgradeKey, serverVersion)).Return(mustMarshalJSON(&serverUpgrade{}), nil)
		// Pretend it's in the future to avoid having to mock this whole process - the code is tested in welcome_test.go
		api.On("KVGet", WelcomeFeedbackMigrationKey).Return(mustMarshalJSON(&welcomeFeedbackMigration{CreateAt: time.Now().AddDate(1, 0, 0)}), nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			now: func() time.Time {
				return now
			},
		}
		p.SetAPI(api)

		err := p.OnActivate()

		assert.Nil(t, err)

		assert.Equal(t, botUserID, p.botUserID)
		assert.Equal(t, serverVersion, p.serverVersion)
		assert.NotNil(t, p.telemetryClient)
	})

	t.Run("should return an error if unable to check for an upgrade", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("GetUserByUsername", "feedbackbot").Return(&model.User{Id: botUserID}, nil)
		api.On("GetBot", botUserID, true).Return(&model.Bot{UserId: botUserID}, nil)
		api.On("GetServerVersion").Return(serverVersion)
		api.On("KVList", 0, 100).Return([]string{}, nil)
		api.On("KVGet", fmt.Sprintf(ServerUpgradeKey, serverVersion)).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := &Plugin{
			now: func() time.Time {
				return now
			},
		}
		p.SetAPI(api)

		err := p.OnActivate()

		assert.NotNil(t, err)

		assert.Equal(t, botUserID, p.botUserID)
		assert.Equal(t, serverVersion, p.serverVersion)
		assert.NotNil(t, p.telemetryClient)
	})

	t.Run("should return an error when unable to get Feedbackbot", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("CreateBot", mock.Anything).Return(nil, &model.AppError{})
		api.On("GetUserByUsername", "feedbackbot").Return(nil, &model.AppError{})
		api.On("GetUserByUsername", "surveybot").Return(nil, &model.AppError{})
		api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.SetAPI(api)

		err := p.OnActivate()

		assert.NotNil(t, err)
	})

	t.Run("should return an error when diagnostics are not enabled", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(false),
			},
		})
		api.On("LogError", mock.Anything)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.SetAPI(api)

		err := p.OnActivate()

		assert.NotNil(t, err)
	})
}

func TestEnsureBotExists(t *testing.T) {
	setupAPI := func() *plugintest.API {
		api := &plugintest.API{}
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogInfo", mock.Anything).Maybe()
		return api
	}

	t.Run("if feedbackbot already exists", func(t *testing.T) {
		t.Run("should find and return the existing bot ID", func(t *testing.T) {
			expectedBotID := model.NewId()

			api := setupAPI()
			api.On("GetUserByUsername", "feedbackbot").Return(&model.User{
				Id: expectedBotID,
			}, nil)
			api.On("GetBot", expectedBotID, true).Return(&model.Bot{UserId: expectedBotID}, nil)
			defer api.AssertExpectations(t)

			p := &Plugin{}
			p.API = api

			botID, err := p.ensureBotExists()

			assert.Equal(t, expectedBotID, botID)
			assert.Nil(t, err)
		})

		t.Run("should rename surveybot to feedbackbot when found", func(t *testing.T) {
			expectedBotID := model.NewId()

			api := setupAPI()
			api.On("GetUserByUsername", "feedbackbot").Return(nil, &model.AppError{})
			api.On("GetUserByUsername", "surveybot").Return(&model.User{
				Id: expectedBotID,
			}, nil)
			api.On("PatchBot", expectedBotID, &model.BotPatch{
				DisplayName: model.NewString("Feedbackbot"),
				Username:    model.NewString("feedbackbot"),
			}).Return(&model.Bot{
				UserId:      expectedBotID,
				Username:    "feedbackbot",
				DisplayName: "Feedbackbot",
			}, nil)
			defer api.AssertExpectations(t)

			p := &Plugin{}
			p.API = api

			botID, err := p.ensureBotExists()

			assert.Equal(t, expectedBotID, botID)
			assert.Nil(t, err)
		})

		t.Run("should return an error if unable to get user and create the bot", func(t *testing.T) {
			api := setupAPI()
			api.On("GetUserByUsername", "feedbackbot").Return(nil, &model.AppError{})
			api.On("GetUserByUsername", "surveybot").Return(nil, &model.AppError{})
			api.On("CreateBot", mock.Anything).Return(nil, &model.AppError{})
			api.On("LogError", mock.Anything, "err", mock.Anything)
			defer api.AssertExpectations(t)

			p := &Plugin{}
			p.API = api

			botID, err := p.ensureBotExists()

			assert.Equal(t, "", botID)
			assert.NotNil(t, err)
		})
	})

	t.Run("if feedbackbot doesn't exist", func(t *testing.T) {
		t.Run("should create the bot and return the ID", func(t *testing.T) {
			expectedBotID := model.NewId()
			profileImageBytes := []byte("profileImage")

			api := setupAPI()
			api.On("GetUserByUsername", "feedbackbot").Return(nil, &model.AppError{})
			api.On("GetUserByUsername", "surveybot").Return(nil, &model.AppError{})
			api.On("CreateBot", mock.Anything).Return(&model.Bot{
				UserId: expectedBotID,
			}, nil)
			api.On("GetBundlePath").Return("", nil)
			api.On("SetProfileImage", expectedBotID, profileImageBytes).Return(nil)
			defer api.AssertExpectations(t)

			p := &Plugin{
				readFile: func(path string) ([]byte, error) {
					return profileImageBytes, nil
				},
			}
			p.API = api

			botID, err := p.ensureBotExists()

			assert.Equal(t, expectedBotID, botID)
			assert.Nil(t, err)
		})

		t.Run("should log a warning if unable to set the profile picture, but still return the bot", func(t *testing.T) {
			expectedBotID := model.NewId()

			api := setupAPI()
			api.On("GetUserByUsername", "feedbackbot").Return(nil, &model.AppError{})
			api.On("GetUserByUsername", "surveybot").Return(nil, &model.AppError{})
			api.On("CreateBot", mock.Anything).Return(&model.Bot{
				UserId: expectedBotID,
			}, nil)
			api.On("GetBundlePath").Return("", &model.AppError{})
			api.On("LogWarn", mock.Anything, "err", mock.Anything)
			defer api.AssertExpectations(t)

			p := &Plugin{}
			p.API = api

			botID, err := p.ensureBotExists()

			assert.Equal(t, expectedBotID, botID)
			assert.Nil(t, err)
		})
	})
}

func TestSetBotProfileImage(t *testing.T) {
	t.Run("should set profile image correctly", func(t *testing.T) {
		botUserID := model.NewId()
		profileImageBytes := []byte("profile image")

		api := &plugintest.API{}
		api.On("GetBundlePath").Return("/foo/bar", nil)
		api.On("SetProfileImage", botUserID, profileImageBytes).Return(nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			readFile: func(path string) ([]byte, error) {
				assert.Equal(t, "/foo/bar/assets/icon-happy-bot-square@1x.png", path)

				return profileImageBytes, nil
			},
		}
		p.API = api

		assert.Nil(t, p.setBotProfileImage(botUserID))
	})

	t.Run("should return an error when SetProfileImage fails", func(t *testing.T) {
		botUserID := model.NewId()
		profileImageBytes := []byte("profile image")

		api := &plugintest.API{}
		api.On("GetBundlePath").Return("/foo/bar", nil)
		api.On("SetProfileImage", botUserID, profileImageBytes).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		p := &Plugin{
			readFile: func(path string) ([]byte, error) {
				assert.Equal(t, "/foo/bar/assets/icon-happy-bot-square@1x.png", path)

				return profileImageBytes, nil
			},
		}
		p.API = api

		assert.NotNil(t, p.setBotProfileImage(botUserID))
	})

	t.Run("should return an error when readFile fails", func(t *testing.T) {
		botUserID := model.NewId()

		api := &plugintest.API{}
		api.On("GetBundlePath").Return("/foo/bar", nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			readFile: func(path string) ([]byte, error) {
				return nil, &model.AppError{}
			},
		}
		p.API = api

		assert.NotNil(t, p.setBotProfileImage(botUserID))
	})

	t.Run("should return an error when GetBundlePath fails", func(t *testing.T) {
		botUserID := model.NewId()

		api := &plugintest.API{}
		api.On("GetBundlePath").Return("", &model.AppError{})
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.API = api

		assert.NotNil(t, p.setBotProfileImage(botUserID))
	})
}
