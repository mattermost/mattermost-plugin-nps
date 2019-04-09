package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSendAdminNoticeEmails(t *testing.T) {
	admins := []*model.User{
		{
			Email: "admin1@example.com",
		},
		{
			Email: "admin2@example.com",
		},
	}

	api := &plugintest.API{}
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
	api.On("SendMail", admins[0].Email, mock.Anything, mock.Anything).Return(nil)
	api.On("SendMail", admins[1].Email, mock.Anything, mock.Anything).Return(nil)
	defer api.AssertExpectations(t)

	p := Plugin{}
	p.SetAPI(api)

	p.sendAdminNoticeEmails(admins)
}

func TestSendAdminNoticeDMs(t *testing.T) {
	admins := []*model.User{
		{
			Id: model.NewId(),
		},
		{
			Id: model.NewId(),
		},
	}
	survey := &surveyState{
		ServerVersion: semver.MustParse("5.10.0"),
		StartAt:       time.Date(2009, time.November, 17, 23, 0, 0, 0, time.UTC),
	}

	api := &plugintest.API{}
	api.On("KVSet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, admins[0].Id, survey.ServerVersion), mustMarshalJSON(&adminNotice{
		Sent:          false,
		ServerVersion: survey.ServerVersion,
		SurveyStartAt: survey.StartAt,
	})).Return(nil)
	api.On("KVSet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, admins[1].Id, survey.ServerVersion), mustMarshalJSON(&adminNotice{
		Sent:          false,
		ServerVersion: survey.ServerVersion,
		SurveyStartAt: survey.StartAt,
	})).Return(nil)
	defer api.AssertExpectations(t)

	p := Plugin{}
	p.SetAPI(api)

	p.sendAdminNoticeDMs(admins, survey)
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

	t.Run("shouldn't return deactivated users", func(t *testing.T) {
		activeEmail := model.NewId()

		api := &plugintest.API{}
		api.On("GetUsers", &model.UserGetOptions{Page: 0, PerPage: perPage, Role: "system_admin"}).Return([]*model.User{
			{
				Email: activeEmail,
			},
			{
				Email:    model.NewId(),
				DeleteAt: 1234,
			},
		}, nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		received, err := p.getAdminUsers(perPage)

		assert.Nil(t, err)
		assert.Len(t, received, 1)
		assert.Equal(t, activeEmail, received[0].Email)
	})
}

func TestCheckForAdminNoticeDM(t *testing.T) {
	botUserID := model.NewId()
	serverVersion := semver.MustParse("5.12.0")

	t.Run("should send notification DM", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, serverVersion)).Return(mustMarshalJSON(&adminNotice{
			Sent:          false,
			ServerVersion: serverVersion,
		}), nil)
		api.On("GetDirectChannel", user.Id, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
		api.On("KVSet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, serverVersion), mustMarshalJSON(&adminNotice{
			Sent:          true,
			ServerVersion: serverVersion,
		})).Return(nil)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForAdminNoticeDM(user, serverVersion)

		assert.True(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should return error if failing to save that the notice was sent", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, serverVersion)).Return(mustMarshalJSON(&adminNotice{
			Sent:          false,
			ServerVersion: serverVersion,
		}), nil)
		api.On("GetDirectChannel", user.Id, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
		api.On("KVSet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, serverVersion), mustMarshalJSON(&adminNotice{
			Sent:          true,
			ServerVersion: serverVersion,
		})).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForAdminNoticeDM(user, serverVersion)

		assert.True(t, sent)
		assert.NotNil(t, err)
	})

	t.Run("should return error if unable to send the DM", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, serverVersion)).Return(mustMarshalJSON(&adminNotice{
			Sent:          false,
			ServerVersion: serverVersion,
		}), nil)
		api.On("GetDirectChannel", user.Id, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForAdminNoticeDM(user, serverVersion)

		assert.True(t, sent)
		assert.NotNil(t, err)
	})

	t.Run("should not resend notification that was already sent", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, serverVersion)).Return(mustMarshalJSON(&adminNotice{
			Sent:          true,
			ServerVersion: serverVersion,
		}), nil)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForAdminNoticeDM(user, serverVersion)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should not resend notification if none are needed", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, serverVersion)).Return(nil, nil)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForAdminNoticeDM(user, serverVersion)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should return error if unable to get pending notice", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ADMIN_DM_NOTICE_KEY, user.Id, serverVersion)).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForAdminNoticeDM(user, serverVersion)

		assert.False(t, sent)
		assert.NotNil(t, err)
	})

	t.Run("should not send notification to non-admin", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID,
		}

		p := Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}

		sent, err := p.checkForAdminNoticeDM(user, serverVersion)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should not send notification when survey is disabled", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		p := Plugin{
			configuration: &configuration{
				EnableSurvey: false,
			},
		}

		sent, err := p.checkForAdminNoticeDM(user, serverVersion)

		assert.False(t, sent)
		assert.Nil(t, err)
	})
}

func TestCheckForSurveyDM(t *testing.T) {
	botUserID := model.NewId()
	now := toDate(2019, time.March, 1)
	postID := model.NewId()
	serverVersion := semver.MustParse("5.12.0")

	newSurveyStateBytes := mustMarshalJSON(&userSurveyState{
		ScorePostId:   postID,
		ServerVersion: serverVersion,
		SentAt:        now,
	})

	t.Run("should send first ever survey DM", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(nil, nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest")).Return(nil, nil)
		api.On("GetConfig").Return(&model.Config{ServiceSettings: model.ServiceSettings{SiteURL: model.NewString("https://mattermost.example.com")}})
		api.On("GetDirectChannel", user.Id, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(&model.Post{Id: postID}, nil)
		api.On("KVSet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion), newSurveyStateBytes).Return(nil)
		api.On("KVSet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest"), newSurveyStateBytes).Return(nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.True(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should return error if unable to save latest survey state", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(nil, nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest")).Return(nil, nil)
		api.On("GetConfig").Return(&model.Config{ServiceSettings: model.ServiceSettings{SiteURL: model.NewString("https://mattermost.example.com")}})
		api.On("GetDirectChannel", user.Id, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(&model.Post{Id: postID}, nil)
		api.On("KVSet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion), newSurveyStateBytes).Return(nil)
		api.On("KVSet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest"), newSurveyStateBytes).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.True(t, sent)
		assert.NotNil(t, err)
	})

	t.Run("should return error if unable to save survey state", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(nil, nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest")).Return(nil, nil)
		api.On("GetConfig").Return(&model.Config{ServiceSettings: model.ServiceSettings{SiteURL: model.NewString("https://mattermost.example.com")}})
		api.On("GetDirectChannel", user.Id, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(&model.Post{Id: postID}, nil)
		api.On("KVSet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion), newSurveyStateBytes).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.True(t, sent)
		assert.NotNil(t, err)
	})

	t.Run("should return error if unable to send DM", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(nil, nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest")).Return(nil, nil)
		api.On("GetConfig").Return(&model.Config{ServiceSettings: model.ServiceSettings{SiteURL: model.NewString("https://mattermost.example.com")}})
		api.On("GetDirectChannel", user.Id, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.True(t, sent)
		assert.NotNil(t, err)
	})

	t.Run("should send survey DM if it's been long enough since the last survey", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(nil, nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest")).Return(mustMarshalJSON(&userSurveyState{
			ServerVersion: semver.MustParse("5.11.0"),
			SentAt:        now.Add(-1 * MIN_TIME_BETWEEN_USER_SURVEYS),
			AnsweredAt:    now.Add(-1 * MIN_TIME_BETWEEN_USER_SURVEYS),
		}), nil)
		api.On("GetConfig").Return(&model.Config{ServiceSettings: model.ServiceSettings{SiteURL: model.NewString("https://mattermost.example.com")}})
		api.On("GetDirectChannel", user.Id, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(&model.Post{Id: postID}, nil)
		api.On("KVSet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion), newSurveyStateBytes).Return(nil)
		api.On("KVSet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest"), newSurveyStateBytes).Return(nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.True(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should not send survey or return error if last survey was answered too recently", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(nil, nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest")).Return(mustMarshalJSON(&userSurveyState{
			ServerVersion: semver.MustParse("5.11.0"),
			SentAt:        now.Add(-1 * MIN_TIME_BETWEEN_USER_SURVEYS),
			AnsweredAt:    now.Add(-1 * MIN_TIME_BETWEEN_USER_SURVEYS).Add(time.Millisecond),
		}), nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should not send survey or return error if last survey was sent too recently", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(nil, nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, "latest")).Return(mustMarshalJSON(&userSurveyState{
			ServerVersion: semver.MustParse("5.11.0"),
			SentAt:        now.Add(-1 * MIN_TIME_BETWEEN_USER_SURVEYS).Add(time.Millisecond),
		}), nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should not send survey or return error if survey was already sent", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(mustMarshalJSON(&userSurveyState{}), nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should return error if unable to get user survey state", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now,
		}), nil)
		api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, user.Id, serverVersion)).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.NotNil(t, err)
	})

	t.Run("should not send survey or return error if survey hasn't started yet", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(mustMarshalJSON(&surveyState{
			ServerVersion: serverVersion,
			StartAt:       now.Add(time.Millisecond),
		}), nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should not send survey or return error if there's no survey scheduled", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(nil, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should return error if unable to get the scheduled survey", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(SURVEY_KEY, serverVersion)).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.NotNil(t, err)
	})

	t.Run("should not send survey or return error if the user hasn't existed for long enough", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).Add(time.Minute).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: true,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.Nil(t, err)
	})

	t.Run("should not send survey or return error if surveys are disabled", func(t *testing.T) {
		user := &model.User{
			Id:       model.NewId(),
			CreateAt: now.Add(-1*TIME_UNTIL_SURVEY).UnixNano() / int64(time.Millisecond),
		}

		api := makeAPIMock()
		defer api.AssertExpectations(t)

		p := &Plugin{
			botUserID: botUserID,
			configuration: &configuration{
				EnableSurvey: false,
			},
		}
		p.SetAPI(api)

		sent, err := p.checkForSurveyDM(user, serverVersion, now)

		assert.False(t, sent)
		assert.Nil(t, err)
	})
}

func TestMarkSurveyAnswered(t *testing.T) {
	now := toDate(2019, 3, 2)
	serverVersion := semver.MustParse("5.8.0")
	userID := model.NewId()

	api := &plugintest.API{}
	api.On("KVGet", fmt.Sprintf(USER_SURVEY_KEY, userID, serverVersion)).Return(mustMarshalJSON(&userSurveyState{
		ServerVersion: serverVersion,
		SentAt:        toDate(2019, 3, 1),
	}), nil)
	api.On("KVSet", fmt.Sprintf(USER_SURVEY_KEY, userID, serverVersion), mustMarshalJSON(&userSurveyState{
		ServerVersion: serverVersion,
		SentAt:        toDate(2019, 3, 1),
		AnsweredAt:    now,
	})).Return(nil)
	defer api.AssertExpectations(t)

	p := Plugin{}
	p.SetAPI(api)

	err := p.markSurveyAnswered(userID, serverVersion, now)

	assert.Nil(t, err)
}
