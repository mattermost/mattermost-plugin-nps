package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEnsureBotExists(t *testing.T) {
	setupAPI := func() *plugintest.API {
		api := &plugintest.API{}
		api.On("LogDebug", mock.Anything).Maybe()
		api.On("LogInfo", mock.Anything).Maybe()
		return api
	}

	t.Run("should return error if unable to read KV store", func(t *testing.T) {
		expectedErr := &model.AppError{Message: "Something went wrong"}

		api := setupAPI()
		api.On("KVGet", BOT_USER_KEY).Return(nil, expectedErr)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.API = api

		botId, err := p.ensureBotExists()

		assert.Equal(t, "", botId)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("should return the stored bot ID", func(t *testing.T) {
		expectedBotId := model.NewId()

		api := setupAPI()
		api.On("KVGet", BOT_USER_KEY).Return(mustMarshalJSON(expectedBotId), nil)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.API = api

		botId, err := p.ensureBotExists()

		assert.Equal(t, expectedBotId, botId)
		assert.Nil(t, err)
	})

	t.Run("if a bot already exists, but is not in the KV store", func(t *testing.T) {
		t.Run("should return an error if unable to get surveybot user", func(t *testing.T) {
			expectedError := &model.AppError{
				Message: "Unable to get surveybot user",
			}

			api := setupAPI()
			api.On("KVGet", BOT_USER_KEY).Return(nil, nil)
			api.On("CreateBot", mock.Anything).Return(nil, &model.AppError{})
			api.On("GetUserByUsername", "surveybot").Return(nil, expectedError)
			defer api.AssertExpectations(t)

			p := &Plugin{}
			p.API = api

			botId, err := p.ensureBotExists()

			assert.Equal(t, "", botId)
			assert.Equal(t, expectedError, err)
		})

		t.Run("should return an error if unable to get bot", func(t *testing.T) {
			expectedBotId := model.NewId()
			expectedError := &model.AppError{
				Message: "Unable to get bot",
			}

			api := setupAPI()
			api.On("KVGet", BOT_USER_KEY).Return(nil, nil)
			api.On("CreateBot", mock.Anything).Return(nil, &model.AppError{})
			api.On("GetUserByUsername", "surveybot").Return(&model.User{
				Id: expectedBotId,
			}, nil)
			api.On("GetBot", expectedBotId, true).Return(nil, expectedError)
			defer api.AssertExpectations(t)

			p := &Plugin{}
			p.API = api

			botId, err := p.ensureBotExists()

			assert.Equal(t, "", botId)
			assert.Equal(t, expectedError, err)
		})

		t.Run("should find and return the existing bot ID", func(t *testing.T) {
			expectedBotId := model.NewId()

			api := setupAPI()
			api.On("KVGet", BOT_USER_KEY).Return(nil, nil)
			api.On("CreateBot", mock.Anything).Return(nil, &model.AppError{})
			api.On("GetUserByUsername", "surveybot").Return(&model.User{
				Id: expectedBotId,
			}, nil)
			api.On("GetBot", expectedBotId, true).Return(&model.Bot{
				UserId: expectedBotId,
			}, nil)
			api.On("KVSet", BOT_USER_KEY, mustMarshalJSON(expectedBotId)).Return(nil)
			defer api.AssertExpectations(t)

			p := &Plugin{}
			p.API = api

			botId, err := p.ensureBotExists()

			assert.Equal(t, expectedBotId, botId)
			assert.Nil(t, err)
		})
	})

	t.Run("if a bot doesn't already exist", func(t *testing.T) {
		t.Run("should return an error if unable to get the bundle path", func(t *testing.T) {
			api := setupAPI()
			api.On("KVGet", BOT_USER_KEY).Return(nil, nil)
			api.On("CreateBot", mock.Anything).Return(&model.Bot{
				UserId: model.NewId(),
			}, nil)
			api.On("GetBundlePath").Return("", &model.AppError{})
			defer api.AssertExpectations(t)

			p := &Plugin{}
			p.API = api

			botId, err := p.ensureBotExists()

			assert.Equal(t, "", botId)
			assert.NotNil(t, err)
		})

		t.Run("should return an error if unable to read the profile picture", func(t *testing.T) {
			api := setupAPI()
			api.On("KVGet", BOT_USER_KEY).Return(nil, nil)
			api.On("CreateBot", mock.Anything).Return(&model.Bot{
				UserId: model.NewId(),
			}, nil)
			api.On("GetBundlePath").Return("", nil)
			defer api.AssertExpectations(t)

			p := &Plugin{
				readFile: func(path string) ([]byte, error) {
					return nil, &model.AppError{}
				},
			}
			p.API = api

			botId, err := p.ensureBotExists()

			assert.Equal(t, "", botId)
			assert.NotNil(t, err)
		})

		t.Run("should log a warning if unable to set the profile picture, but still return the bot", func(t *testing.T) {
			expectedBotId := model.NewId()
			profileImageBytes := []byte("profileImage")
			expectedError := &model.AppError{
				Message: "Unable to set profile picture",
			}

			api := setupAPI()
			api.On("KVGet", BOT_USER_KEY).Return(nil, nil)
			api.On("CreateBot", mock.Anything).Return(&model.Bot{
				UserId: expectedBotId,
			}, nil)
			api.On("GetBundlePath").Return("", nil)
			api.On("SetProfileImage", expectedBotId, profileImageBytes).Return(expectedError)
			api.On("LogWarn", mock.Anything, mock.Anything, expectedError)
			api.On("KVSet", BOT_USER_KEY, mustMarshalJSON(expectedBotId)).Return(nil)
			defer api.AssertExpectations(t)

			p := &Plugin{
				readFile: func(path string) ([]byte, error) {
					return profileImageBytes, nil
				},
			}
			p.API = api

			botId, err := p.ensureBotExists()

			assert.Equal(t, expectedBotId, botId)
			assert.Nil(t, err)
		})

		t.Run("should create the bot and return the ID", func(t *testing.T) {
			expectedBotId := model.NewId()
			profileImageBytes := []byte("profileImage")

			api := setupAPI()
			api.On("KVGet", BOT_USER_KEY).Return(nil, nil)
			api.On("CreateBot", mock.Anything).Return(&model.Bot{
				UserId: expectedBotId,
			}, nil)
			api.On("GetBundlePath").Return("", nil)
			api.On("SetProfileImage", expectedBotId, profileImageBytes).Return(nil)
			api.On("KVSet", BOT_USER_KEY, mustMarshalJSON(expectedBotId)).Return(nil)
			defer api.AssertExpectations(t)

			p := &Plugin{
				readFile: func(path string) ([]byte, error) {
					return profileImageBytes, nil
				},
			}
			p.API = api

			botId, err := p.ensureBotExists()

			assert.Equal(t, expectedBotId, botId)
			assert.Nil(t, err)
		})
	})
}
