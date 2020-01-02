package main

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/mock"
)

func TestChannelHasBeenCreated(t *testing.T) {
	botUserID := model.NewId()

	t.Run("should set channel header for a Surveybot DM channel", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("UpdateChannel", mock.MatchedBy(func(channel *model.Channel) bool {
			return channel.Header == SURVEYBOT_DESCRIPTION
		})).Return(nil, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
		}
		p.SetAPI(api)

		p.ChannelHasBeenCreated(nil, &model.Channel{
			Type: model.CHANNEL_DIRECT,
			Name: fmt.Sprintf("%s__%s", botUserID, model.NewId()),
		})
	})

	t.Run("should do nothing for other channels", func(t *testing.T) {
		p := &Plugin{}

		p.ChannelHasBeenCreated(nil, &model.Channel{
			Type: model.CHANNEL_OPEN,
		})
	})
}

func TestMessageHasBeenPosted(t *testing.T) {
	botChannelID := model.NewId()
	botUserID := model.NewId()
	userID := model.NewId()

	t.Run("should send feedback to segment and respond to user", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("GetChannel", botChannelID).Return(&model.Channel{
			Type: model.CHANNEL_DIRECT,
			Name: fmt.Sprintf("%s__%s", botUserID, userID),
		}, nil)
		api.On("GetUser", userID).Return(&model.User{}, nil)
		api.On("GetDirectChannel", userID, botUserID).Return(&model.Channel{
			Id: botChannelID,
		}, nil)
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			blockSegmentEvents: true,
			botUserID:          botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
		})
	})

	t.Run("should not respond to posts made by other bots", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("GetChannel", botChannelID).Return(&model.Channel{
			Type: model.CHANNEL_DIRECT,
			Name: fmt.Sprintf("%s__%s", botUserID, userID),
		}, nil)
		api.On("GetUser", userID).Return(&model.User{
			IsBot: true,
		}, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			blockSegmentEvents: true,
			botUserID:          botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
		})
	})

	t.Run("should not respond to a post outside of a Surveybot DM channel", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("GetChannel", botChannelID).Return(&model.Channel{
			Type: model.CHANNEL_OPEN,
		}, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			blockSegmentEvents: true,
			botUserID:          botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
		})
	})

	t.Run("should not respond to posts made by Surveybot", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		defer api.AssertExpectations(t)

		p := &Plugin{
			blockSegmentEvents: true,
			botUserID:          botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    botUserID,
		})
	})

	t.Run("should do nothing if diagnostics are disabled", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(false),
			},
		})
		defer api.AssertExpectations(t)

		p := &Plugin{
			blockSegmentEvents: true,
			botUserID:          botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
		})
	})

	t.Run("should not respond to autoresponder posts", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		defer api.AssertExpectations(t)

		p := &Plugin{
			blockSegmentEvents: true,
			botUserID:          botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
			Type:      model.POST_AUTO_RESPONDER,
		})
	})

	t.Run("should not respond to other system messages", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		defer api.AssertExpectations(t)

		p := &Plugin{
			blockSegmentEvents: true,
			botUserID:          botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
			Type:      model.POST_HEADER_CHANGE,
		})
	})
}

func TestUserHasLoggedIn(t *testing.T) {
	t.Run("should check for DMs when a user logs in", func(t *testing.T) {
		t.SkipNow()
	})
}
