package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func TestShouldSendAdminNotices(t *testing.T) {
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
			assert.Equal(t, test.Expected, shouldSendAdminNotices(test.Now, test.LastUpgrade))
		})
	}
}

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

	nextSurvey := time.Date(2009, time.November, 17, 23, 0, 0, 0, time.UTC)

	api := &plugintest.API{}
	api.On("KVSet", ADMIN_DM_NOTICE_KEY+admins[0].Id, []byte(`{"Sent":false,"NextSurvey":"2009-11-17T23:00:00Z"}`)).Return(nil)
	api.On("KVSet", ADMIN_DM_NOTICE_KEY+admins[1].Id, []byte(`{"Sent":false,"NextSurvey":"2009-11-17T23:00:00Z"}`)).Return(nil)
	defer api.AssertExpectations(t)

	p := Plugin{}
	p.SetAPI(api)

	p.sendAdminNoticeDMs(admins, nextSurvey)
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
	t.Run("not a system admin", func(t *testing.T) {
		user := &model.User{
			Roles: model.SYSTEM_USER_ROLE_ID,
		}

		p := Plugin{}

		notice := p.checkForAdminNoticeDM(user)

		assert.Nil(t, notice)
	})

	t.Run("should log error when KVGet fails", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		appErr := &model.AppError{}

		api := &plugintest.API{}
		api.On("KVGet", ADMIN_DM_NOTICE_KEY+user.Id).Return(nil, appErr)
		api.On("LogError", mock.Anything, "err", appErr)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		notice := p.checkForAdminNoticeDM(user)

		assert.Nil(t, notice)
	})

	t.Run("shouldn't error when no notice is stored", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := &plugintest.API{}
		api.On("KVGet", ADMIN_DM_NOTICE_KEY+user.Id).Return(nil, nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		notice := p.checkForAdminNoticeDM(user)

		assert.Nil(t, notice)
	})

	t.Run("should log error when decoding fails", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := &plugintest.API{}
		api.On("KVGet", ADMIN_DM_NOTICE_KEY+user.Id).Return([]byte("garbage"), nil)
		api.On("LogError", mock.Anything, "err", mock.Anything)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		notice := p.checkForAdminNoticeDM(user)

		assert.Nil(t, notice)
	})

	t.Run("shouldn't return notice when already sent", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		api := &plugintest.API{}
		api.On("KVGet", ADMIN_DM_NOTICE_KEY+user.Id).Return([]byte(`{"Sent":true}`), nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		notice := p.checkForAdminNoticeDM(user)

		assert.Nil(t, notice)
	})

	t.Run("should return unsent notice", func(t *testing.T) {
		user := &model.User{
			Id:    model.NewId(),
			Roles: model.SYSTEM_USER_ROLE_ID + " " + model.SYSTEM_ADMIN_ROLE_ID,
		}

		nextSurvey := time.Date(2009, time.November, 17, 23, 0, 0, 0, time.UTC)

		api := &plugintest.API{}
		api.On("KVGet", ADMIN_DM_NOTICE_KEY+user.Id).Return([]byte(`{"Sent":false,"NextSurvey":"2009-11-17T23:00:00Z"}`), nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		notice := p.checkForAdminNoticeDM(user)

		assert.Equal(t, &adminNotice{Sent: false, NextSurvey: nextSurvey}, notice)
	})
}

func TestSendAdminNoticeDM(t *testing.T) {
	t.Run("should send DM and mark notice as sent", func(t *testing.T) {
		botUserId := model.NewId()
		user := &model.User{
			Id: model.NewId(),
		}
		notice := &adminNotice{
			NextSurvey: time.Date(2009, time.November, 17, 23, 0, 0, 0, time.UTC),
			Sent:       false,
		}

		api := &plugintest.API{}
		api.On("LogDebug", "Sending admin notice DM", "user_id", user.Id)
		api.On("GetDirectChannel", user.Id, botUserId).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, nil)
		api.On("KVSet", ADMIN_DM_NOTICE_KEY+user.Id, []byte(`{"Sent":true,"NextSurvey":"2009-11-17T23:00:00Z"}`)).Return(nil)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserId: botUserId,
		}
		p.SetAPI(api)

		p.sendAdminNoticeDM(user, notice)
	})

	t.Run("should log error from failed DM", func(t *testing.T) {
		botUserId := model.NewId()
		user := &model.User{
			Id: model.NewId(),
		}
		notice := &adminNotice{
			NextSurvey: time.Date(2009, time.November, 17, 23, 0, 0, 0, time.UTC),
			Sent:       false,
		}

		appErr := &model.AppError{}

		api := &plugintest.API{}
		api.On("LogDebug", "Sending admin notice DM", "user_id", user.Id)
		api.On("GetDirectChannel", user.Id, botUserId).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, appErr)
		api.On("LogError", mock.Anything, "user_id", user.Id, "err", appErr)
		api.On("LogError", mock.Anything, "err", appErr)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserId: botUserId,
		}
		p.SetAPI(api)

		p.sendAdminNoticeDM(user, notice)
	})

	t.Run("should log error when failing to store sent notice", func(t *testing.T) {
		botUserId := model.NewId()
		user := &model.User{
			Id: model.NewId(),
		}
		notice := &adminNotice{
			NextSurvey: time.Date(2009, time.November, 17, 23, 0, 0, 0, time.UTC),
			Sent:       false,
		}

		appErr := &model.AppError{}

		api := &plugintest.API{}
		api.On("LogDebug", "Sending admin notice DM", "user_id", user.Id)
		api.On("GetDirectChannel", user.Id, botUserId).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, nil)
		api.On("KVSet", ADMIN_DM_NOTICE_KEY+user.Id, []byte(`{"Sent":true,"NextSurvey":"2009-11-17T23:00:00Z"}`)).Return(appErr)
		api.On("LogError", mock.Anything, "err", appErr)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserId: botUserId,
		}
		p.SetAPI(api)

		p.sendAdminNoticeDM(user, notice)
	})
}

func TestShouldSendSurveyDM(t *testing.T) {
	for _, test := range []struct {
		Name          string
		User          *model.User
		Now           time.Time
		LastUpgrade   *serverUpgrade
		SurveyState   *surveyState
		ServerVersion semver.Version
		Expected      bool
	}{
		{
			Name: "should send survey",
			User: &model.User{
				Id:       model.NewId(),
				CreateAt: toDate(2019, time.March, 11).UnixNano() / int64(time.Millisecond),
			},
			Now: toDate(2019, time.April, 1),
			LastUpgrade: &serverUpgrade{
				Timestamp: toDate(2019, time.March, 11),
			},
			SurveyState: &surveyState{
				ServerVersion: semver.MustParse("5.9.0"),
				SentAt:        toDate(2019, time.January, 1),
				AnsweredAt:    toDate(2019, time.January, 1),
			},
			ServerVersion: semver.MustParse("5.10.0"),
			Expected:      true,
		},
		{
			Name: "unable to get last upgrade",
			User: &model.User{
				Id:       model.NewId(),
				CreateAt: toDate(2019, time.March, 11).UnixNano() / int64(time.Millisecond),
			},
			Now:         toDate(2019, time.April, 1),
			LastUpgrade: nil,
			SurveyState: &surveyState{
				ServerVersion: semver.MustParse("5.9.0"),
				SentAt:        toDate(2019, time.January, 1),
				AnsweredAt:    toDate(2019, time.January, 1),
			},
			ServerVersion: semver.MustParse("5.10.0"),
			Expected:      false,
		},
		{
			Name: "should not send too soon after upgrade",
			User: &model.User{
				Id:       model.NewId(),
				CreateAt: toDate(2019, time.March, 11).UnixNano() / int64(time.Millisecond),
			},
			Now: toDate(2019, time.April, 1),
			LastUpgrade: &serverUpgrade{
				Timestamp: toDate(2019, time.March, 12),
			},
			SurveyState: &surveyState{
				ServerVersion: semver.MustParse("5.9.0"),
				SentAt:        toDate(2019, time.January, 1),
				AnsweredAt:    toDate(2019, time.January, 1),
			},
			ServerVersion: semver.MustParse("5.10.0"),
			Expected:      false,
		},
		{
			Name: "should not send for recently created user",
			User: &model.User{
				Id:       model.NewId(),
				CreateAt: toDate(2019, time.March, 12).UnixNano() / int64(time.Millisecond),
			},
			Now: toDate(2019, time.April, 1),
			LastUpgrade: &serverUpgrade{
				Timestamp: toDate(2019, time.March, 11),
			},
			SurveyState: &surveyState{
				ServerVersion: semver.MustParse("5.9.0"),
				SentAt:        toDate(2019, time.January, 1),
				AnsweredAt:    toDate(2019, time.January, 1),
			},
			ServerVersion: semver.MustParse("5.10.0"),
			Expected:      false,
		},
		{
			Name: "should still send with no stored survey state",
			User: &model.User{
				Id:       model.NewId(),
				CreateAt: toDate(2019, time.March, 11).UnixNano() / int64(time.Millisecond),
			},
			Now: toDate(2019, time.April, 1),
			LastUpgrade: &serverUpgrade{
				Timestamp: toDate(2019, time.March, 11),
			},
			SurveyState:   nil,
			ServerVersion: semver.MustParse("5.10.0"),
			Expected:      true,
		},
		{
			Name: "should not send on same server version",
			User: &model.User{
				Id:       model.NewId(),
				CreateAt: toDate(2019, time.March, 11).UnixNano() / int64(time.Millisecond),
			},
			Now: toDate(2019, time.April, 1),
			LastUpgrade: &serverUpgrade{
				Timestamp: toDate(2019, time.March, 11),
			},
			SurveyState: &surveyState{
				ServerVersion: semver.MustParse("5.10.0"),
				SentAt:        toDate(2019, time.January, 1),
				AnsweredAt:    toDate(2019, time.January, 1),
			},
			ServerVersion: semver.MustParse("5.10.0"),
			Expected:      false,
		},
		{
			Name: "should not send too soon after sending last survey",
			User: &model.User{
				Id:       model.NewId(),
				CreateAt: toDate(2019, time.March, 11).UnixNano() / int64(time.Millisecond),
			},
			Now: toDate(2019, time.April, 1),
			LastUpgrade: &serverUpgrade{
				Timestamp: toDate(2019, time.March, 11),
			},
			SurveyState: &surveyState{
				ServerVersion: semver.MustParse("5.9.0"),
				SentAt:        toDate(2019, time.January, 2),
				AnsweredAt:    toDate(2019, time.January, 1),
			},
			ServerVersion: semver.MustParse("5.10.0"),
			Expected:      false,
		},
		{
			Name: "should send too soon after answering last survey",
			User: &model.User{
				Id:       model.NewId(),
				CreateAt: toDate(2019, time.March, 11).UnixNano() / int64(time.Millisecond),
			},
			Now: toDate(2019, time.April, 1),
			LastUpgrade: &serverUpgrade{
				Timestamp: toDate(2019, time.March, 11),
			},
			SurveyState: &surveyState{
				ServerVersion: semver.MustParse("5.9.0"),
				SentAt:        toDate(2019, time.January, 1),
				AnsweredAt:    toDate(2019, time.January, 2),
			},
			ServerVersion: semver.MustParse("5.10.0"),
			Expected:      false,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			var lastUpgradeBytes []byte
			if test.LastUpgrade != nil {
				lastUpgradeBytes, _ = json.Marshal(test.LastUpgrade)
			}
			var surveyStateBytes []byte
			if test.SurveyState != nil {
				surveyStateBytes, _ = json.Marshal(test.SurveyState)
			}

			api := &plugintest.API{}
			api.On("KVGet", SERVER_UPGRADE_KEY).Return(lastUpgradeBytes, nil).Maybe()
			api.On("KVGet", USER_SURVEY_KEY+test.User.Id).Return(surveyStateBytes, nil).Maybe()
			api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Maybe()
			defer api.AssertExpectations(t)

			p := Plugin{
				serverVersion: test.ServerVersion,
			}
			p.SetAPI(api)

			result := p.shouldSendSurveyDM(test.User, test.Now)

			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestSendSurveyDM(t *testing.T) {
	t.Run("should send DM and mark survey as sent", func(t *testing.T) {
		botUserId := model.NewId()
		serverVersion := semver.MustParse("5.10.0")
		user := &model.User{
			Id:       model.NewId(),
			Username: "testuser",
		}
		now := toDate(2019, time.March, 1)

		api := &plugintest.API{}
		api.On("LogDebug", "Sending survey DM", "user_id", user.Id)
		api.On("GetConfig").Return(&model.Config{
			ServiceSettings: model.ServiceSettings{
				SiteURL: model.NewString("https://mattermost.example.com"),
			},
		})
		api.On("GetDirectChannel", user.Id, botUserId).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, nil)
		api.On("KVSet", USER_SURVEY_KEY+user.Id, []byte(`{"ServerVersion":"5.10.0","SentAt":"2019-03-01T00:00:00Z","AnsweredAt":"0001-01-01T00:00:00Z"}`)).Return(nil)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserId:     botUserId,
			serverVersion: serverVersion,
		}
		p.SetAPI(api)

		p.sendSurveyDM(user, now)
	})

	t.Run("should log error from failed DM", func(t *testing.T) {
		botUserId := model.NewId()
		serverVersion := semver.MustParse("5.10.0")
		user := &model.User{
			Id:       model.NewId(),
			Username: "testuser",
		}
		now := toDate(2019, time.March, 1)

		appErr := &model.AppError{}

		api := &plugintest.API{}
		api.On("LogDebug", "Sending survey DM", "user_id", user.Id)
		api.On("GetConfig").Return(&model.Config{
			ServiceSettings: model.ServiceSettings{
				SiteURL: model.NewString("https://mattermost.example.com"),
			},
		})
		api.On("GetDirectChannel", user.Id, botUserId).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, appErr)
		api.On("LogError", mock.Anything, "user_id", user.Id, "err", appErr)
		api.On("LogError", mock.Anything, "err", appErr)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserId:     botUserId,
			serverVersion: serverVersion,
		}
		p.SetAPI(api)

		p.sendSurveyDM(user, now)
	})

	t.Run("should log error when failing to store sent survey", func(t *testing.T) {
		botUserId := model.NewId()
		serverVersion := semver.MustParse("5.10.0")
		user := &model.User{
			Id:       model.NewId(),
			Username: "testuser",
		}
		now := toDate(2019, time.March, 1)

		appErr := &model.AppError{}

		api := &plugintest.API{}
		api.On("LogDebug", "Sending survey DM", "user_id", user.Id)
		api.On("GetConfig").Return(&model.Config{
			ServiceSettings: model.ServiceSettings{
				SiteURL: model.NewString("https://mattermost.example.com"),
			},
		})
		api.On("GetDirectChannel", user.Id, botUserId).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(nil, nil)
		api.On("KVSet", USER_SURVEY_KEY+user.Id, []byte(`{"ServerVersion":"5.10.0","SentAt":"2019-03-01T00:00:00Z","AnsweredAt":"0001-01-01T00:00:00Z"}`)).Return(appErr)
		api.On("LogError", mock.Anything, "err", appErr)
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserId:     botUserId,
			serverVersion: serverVersion,
		}
		p.SetAPI(api)

		p.sendSurveyDM(user, now)
	})
}

func toDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
