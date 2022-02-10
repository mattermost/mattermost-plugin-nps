package main

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/mock"
)

func TestChannelHasBeenCreated(t *testing.T) {
	botUserID := model.NewId()

	t.Run("should set channel header for a Feedbackbot DM channel", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("UpdateChannel", mock.MatchedBy(func(channel *model.Channel) bool {
			return channel.Header == FeedbackbottDescription
		})).Return(nil, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
		}
		p.SetAPI(api)

		p.ChannelHasBeenCreated(nil, &model.Channel{
			Type: model.ChannelTypeDirect,
			Name: fmt.Sprintf("%s__%s", botUserID, model.NewId()),
		})
	})

	t.Run("should do nothing for other channels", func(t *testing.T) {
		p := &Plugin{}

		p.ChannelHasBeenCreated(nil, &model.Channel{
			Type: model.ChannelTypeOpen,
		})
	})
}

func TestMessageHasBeenPosted(t *testing.T) {
	botChannelID := model.NewId()
	botUserID := model.NewId()
	userID := model.NewId()
	rootID := model.NewId()

	systemInstallDate := int64(1497898133094)
	teamMembers := []*model.TeamMember{
		{
			Roles: model.TeamUserRoleId,
		},
	}
	licenseID := model.NewId()
	skuShortName := model.NewId()

	t.Run("should send feedback to segment and respond to user's post on existing post", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("GetChannel", botChannelID).Return(&model.Channel{
			Type: model.ChannelTypeDirect,
			Name: fmt.Sprintf("%s__%s", botUserID, userID),
		}, nil)
		api.On("GetUser", userID).Return(&model.User{Id: userID}, nil)
		api.On("GetDirectChannel", userID, botUserID).Return(&model.Channel{
			Id: botChannelID,
		}, nil)
		api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return post.RootId == rootID
		})).Return(nil, nil)
		api.On("GetSystemInstallDate").Return(systemInstallDate, nil)
		api.On("GetTeamMembersForUser", userID, 0, 50).Return(teamMembers, nil)
		api.On("GetLicense").Return(&model.License{
			Id:           licenseID,
			SkuShortName: skuShortName,
		})
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			tracker:   telemetry.NewTracker(nil, "", "", "", "", "", false),
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
			RootId:    rootID,
		})
	})

	postID := model.NewId()

	t.Run("should send feedback to segment and respond to user's new post", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("GetChannel", botChannelID).Return(&model.Channel{
			Type: model.ChannelTypeDirect,
			Name: fmt.Sprintf("%s__%s", botUserID, userID),
		}, nil)
		api.On("GetUser", userID).Return(&model.User{Id: userID}, nil)
		api.On("GetDirectChannel", userID, botUserID).Return(&model.Channel{
			Id: botChannelID,
		}, nil)
		api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return post.RootId == postID
		})).Return(nil, nil)
		api.On("GetSystemInstallDate").Return(systemInstallDate, nil)
		api.On("GetTeamMembersForUser", userID, 0, 50).Return(teamMembers, nil)
		api.On("GetLicense").Return(&model.License{
			Id:           licenseID,
			SkuShortName: skuShortName,
		})
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			tracker:   telemetry.NewTracker(nil, "", "", "", "", "", false),
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			Id:        postID,
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
			Type: model.ChannelTypeDirect,
			Name: fmt.Sprintf("%s__%s", botUserID, userID),
		}, nil)
		api.On("GetUser", userID).Return(&model.User{
			IsBot: true,
		}, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			tracker:   telemetry.NewTracker(nil, "", "", "", "", "", false),
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
		})
	})

	t.Run("should not respond to a post outside of a Feedbackbot DM channel", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("GetChannel", botChannelID).Return(&model.Channel{
			Type: model.ChannelTypeOpen,
		}, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
		})
	})

	t.Run("should not respond to posts made by Feedbackbot", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
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
			botUserID: botUserID,
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
			botUserID: botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
			Type:      model.PostTypeAutoResponder,
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
			botUserID: botUserID,
		}
		p.SetAPI(api)

		p.MessageHasBeenPosted(nil, &model.Post{
			ChannelId: botChannelID,
			UserId:    userID,
			Type:      model.PostTypeHeaderChange,
		})
	})
}

func TestUserHasLoggedIn(t *testing.T) {
	t.Run("should check for DMs when a user logs in", func(t *testing.T) {
		t.SkipNow()
	})
}
