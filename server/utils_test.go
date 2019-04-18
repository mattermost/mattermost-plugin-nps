package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
)

func TestGetServerVersion(t *testing.T) {
	t.Run("should set the patch number to 0", func(t *testing.T) {
		assert.Equal(t, "5.11.0", getServerVersion("5.11.1"))
	})
}

func TestKVSet(t *testing.T) {
	t.Run("should save a json encoded object in the KV store", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVSet", "key", []byte(`"value"`)).Return(nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		err := p.KVSet("key", "value")

		assert.Nil(t, err)
	})

	t.Run("should return an error if saving the value in the KV store fails", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVSet", "key", mock.Anything).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		err := p.KVSet("key", "value")

		assert.NotNil(t, err)
	})
}

func TestKVGet(t *testing.T) {
	t.Run("should save a json encoded object in the KV store", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", "key").Return([]byte(`"value"`), nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		var value string
		err := p.KVGet("key", &value)

		assert.Nil(t, err)
		assert.Equal(t, "value", value)
	})

	t.Run("should not modify value and return nil when an object doesn't exist in the KV store", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", "key").Return(nil, nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		var value string
		err := p.KVGet("key", &value)

		assert.Nil(t, err)
		assert.Equal(t, "", value)
	})

	t.Run("should return an error if getting the value from the KV store fails", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", "key").Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		var value string
		err := p.KVGet("key", &value)

		assert.NotNil(t, err)
		assert.Equal(t, "", value)
	})

	t.Run("should return an error if decoding the value fails", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", "key").Return([]byte(`"value`), nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		var value string
		err := p.KVGet("key", &value)

		assert.NotNil(t, err)
		assert.Equal(t, "", value)
	})
}

func TestCreateBotDMPost(t *testing.T) {
	t.Run("should send bot DM correctly", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetDirectChannel", "userID", "botUserID").Return(&model.Channel{Id: "channelID"}, nil)
		api.On("CreatePost", &model.Post{
			ChannelId: "channelID",
			Message:   "test",
			UserId:    "botUserID",
		}).Return(&model.Post{
			Id:        "postID",
			ChannelId: "channelID",
			Message:   "test",
			UserId:    "botUserID",
		}, nil)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: "botUserID",
		}
		p.SetAPI(api)

		post, err := p.CreateBotDMPost("userID", &model.Post{
			Message: "test",
		})

		assert.Nil(t, err)
		assert.Equal(t, "botUserID", post.UserId)
		assert.Equal(t, "channelID", post.ChannelId)
	})

	t.Run("should return an error if unable to get the DM channel", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetDirectChannel", "userID", "botUserID").Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: "botUserID",
		}
		p.SetAPI(api)

		_, err := p.CreateBotDMPost("userID", &model.Post{
			Message: "test",
		})

		assert.NotNil(t, err)
	})

	t.Run("should return an error if unable to create the post", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetDirectChannel", "userID", "botUserID").Return(&model.Channel{Id: "channelID"}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: "botUserID",
		}
		p.SetAPI(api)

		_, err := p.CreateBotDMPost("userID", &model.Post{
			Message: "test",
		})

		assert.NotNil(t, err)
	})
}

func TestIsBotChannel(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Channel  *model.Channel
		Expected bool
	}{
		{
			Name:     "not a direct channel",
			Channel:  &model.Channel{Type: model.CHANNEL_OPEN},
			Expected: false,
		},
		{
			Name: "a direct channel with another user",
			Channel: &model.Channel{
				Name: "user1__user2",
				Type: model.CHANNEL_DIRECT,
			},
			Expected: false,
		},
		{
			Name: "a direct channel with the name containing the bot's ID first",
			Channel: &model.Channel{
				Name: "botUserID__user2",
				Type: model.CHANNEL_DIRECT,
			},
			Expected: true,
		},
		{
			Name: "a direct channel with the name containing the bot's ID second",
			Channel: &model.Channel{
				Name: "user1__botUserID",
				Type: model.CHANNEL_DIRECT,
			},
			Expected: true,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := Plugin{
				botUserID: "botUserID",
			}

			assert.Equal(t, test.Expected, p.IsBotDMChannel(test.Channel))
		})
	}
}

// Test helper functions

func makeAPIMock() *plugintest.API {
	api := &plugintest.API{}

	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	return api
}

func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return data
}

func mustUnmarshalJSON(data []byte, v interface{}) interface{} {
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}

	return v
}

func toDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
