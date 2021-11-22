package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckForDMs(t *testing.T) {
	now := toDate(2019, time.May, 10)
	userID := model.NewId()

	userLockKey := fmt.Sprintf(UserLockKey, userID)

	t.Run("should do nothing with diagnostics disabled", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(false),
			},
		})
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		err := p.checkForDMs(userID)

		assert.Nil(t, err)
	})

	t.Run("should not try to check for DMs if user is already locked", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("KVCompareAndSet", userLockKey, []byte(nil), mustMarshalJSON(now)).Return(false, nil)
		defer api.AssertExpectations(t)

		p := Plugin{
			now: func() time.Time {
				return now
			},
		}
		p.SetAPI(api)

		err := p.checkForDMs(userID)

		assert.Nil(t, err)
	})

	t.Run("should return error if unable to get user", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(true),
			},
		})
		api.On("KVCompareAndSet", userLockKey, []byte(nil), mustMarshalJSON(now)).Return(true, nil)
		api.On("GetUser", userID).Return(nil, &model.AppError{})
		api.On("KVDelete", userLockKey).Return(nil)
		defer api.AssertExpectations(t)

		p := Plugin{
			now: func() time.Time {
				return now
			},
		}
		p.SetAPI(api)

		err := p.checkForDMs(userID)

		assert.NotNil(t, err)
	})

	// The rest of this functionality is tested by TestCheckForAdminNoticeDM and TestCheckForSurveyDM
}

func TestSubmitScore(t *testing.T) {
	botUserID := model.NewId()
	userID := model.NewId()
	userSurveyKey := fmt.Sprintf(UserSurveyKey, userID)
	systemInstallDate := int64(1497898133094)
	teamMembers := []*model.TeamMember{
		{
			Roles: model.TeamUserRoleId,
		},
	}
	licenseID := model.NewId()
	skuShortName := model.NewId()

	now := toDate(2018, time.April, 1)

	makeAPIMock := func() *plugintest.API {
		api := &plugintest.API{}
		api.On("LogDebug", mock.Anything).Maybe()

		// Disabling diagnostics allows the handler to run, but prevents data from being sent to Segment
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(false),
			},
		}).Maybe()

		return api
	}

	t.Run("should send score to segment, respond for additional feedback, and update the score post", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetUser", userID).Return(&model.User{
			Id: userID,
		}, nil)
		api.On("KVGet", userSurveyKey).Return(mustMarshalJSON(&userSurveyState{}), nil)
		api.On("KVSet", userSurveyKey, mustMarshalJSON(&userSurveyState{
			AnsweredAt: now,
		})).Return(nil)
		api.On("GetDirectChannel", userID, botUserID).Return(&model.Channel{}, nil)
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
		api.On("GetSystemInstallDate").Return(systemInstallDate, nil)
		api.On("GetTeamMembersForUser", userID, 0, 50).Return(teamMembers, nil)
		api.On("GetLicense").Return(&model.License{
			Id:           licenseID,
			SkuShortName: skuShortName,
		})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			now: func() time.Time {
				return now
			},
			tracker: telemetry.NewTracker(nil, "", "", "", "", "", false),
		}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(mustMarshalJSON(&model.PostActionIntegrationRequest{
			Context: map[string]interface{}{
				"selected_option": "10",
			},
		})))
		request.Header.Set("Mattermost-User-ID", userID)

		p.submitScore(recorder, request)

		result := recorder.Result()
		body, _ := ioutil.ReadAll(result.Body)

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.IsType(t, &model.PostActionIntegrationResponse{}, mustUnmarshalJSON(body, &model.PostActionIntegrationResponse{}))
	})

	t.Run("should not respond for feedback if the user changes their score", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetUser", userID).Return(&model.User{
			Id: userID,
		}, nil)
		api.On("KVGet", userSurveyKey).Return(mustMarshalJSON(&userSurveyState{
			AnsweredAt: now.Add(-time.Minute),
		}), nil)
		api.On("GetSystemInstallDate").Return(systemInstallDate, nil)
		api.On("GetTeamMembersForUser", userID, 0, 50).Return(teamMembers, nil)
		api.On("GetLicense").Return(&model.License{
			Id:           licenseID,
			SkuShortName: skuShortName,
		})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			now: func() time.Time {
				return now
			},
			tracker: telemetry.NewTracker(nil, "", "", "", "", "", false),
		}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(mustMarshalJSON(&model.PostActionIntegrationRequest{
			Context: map[string]interface{}{
				"selected_option": "10",
			},
		})))
		request.Header.Set("Mattermost-User-ID", userID)

		p.submitScore(recorder, request)

		result := recorder.Result()
		body, _ := ioutil.ReadAll(result.Body)

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.IsType(t, &model.PostActionIntegrationResponse{}, mustUnmarshalJSON(body, &model.PostActionIntegrationResponse{}))
	})

	t.Run("should only log warning if unable to mark survey answered", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetUser", userID).Return(&model.User{
			Id: userID,
		}, nil)
		api.On("KVGet", userSurveyKey).Return(nil, &model.AppError{})
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything)
		api.On("GetSystemInstallDate").Return(systemInstallDate, nil)
		api.On("GetTeamMembersForUser", userID, 0, 50).Return(teamMembers, nil)
		api.On("GetLicense").Return(&model.License{
			Id:           licenseID,
			SkuShortName: skuShortName,
		})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			now: func() time.Time {
				return now
			},
			tracker: telemetry.NewTracker(nil, "", "", "", "", "", false),
		}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(mustMarshalJSON(&model.PostActionIntegrationRequest{
			Context: map[string]interface{}{
				"selected_option": "10",
			},
		})))
		request.Header.Set("Mattermost-User-ID", userID)

		p.submitScore(recorder, request)

		result := recorder.Result()
		body, _ := ioutil.ReadAll(result.Body)

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.IsType(t, &model.PostActionIntegrationResponse{}, mustUnmarshalJSON(body, &model.PostActionIntegrationResponse{}))
	})

	t.Run("should return bad request if score is missing or invalid", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetUser", userID).Return(&model.User{
			Id: userID,
		}, nil)
		api.On("LogError", mock.Anything)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(mustMarshalJSON(&model.PostActionIntegrationRequest{
			Context: map[string]interface{}{
				"selected_option": "hmm",
			},
		})))
		request.Header.Set("Mattermost-User-ID", userID)

		p.submitScore(recorder, request)

		result := recorder.Result()

		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	})

	t.Run("should return error if unable to get user", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetUser", userID).Return(nil, &model.AppError{})
		api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(mustMarshalJSON(&model.PostActionIntegrationRequest{
			Context: map[string]interface{}{
				"selected_option": "10",
			},
		})))
		request.Header.Set("Mattermost-User-ID", userID)

		p.submitScore(recorder, request)

		result := recorder.Result()

		assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
	})

	t.Run("should return bad request if request context is missing", func(t *testing.T) {
		api := makeAPIMock()
		api.On("LogError", mock.Anything)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(mustMarshalJSON(&model.PostActionIntegrationRequest{})))
		request.Header.Set("Mattermost-User-ID", userID)

		p.submitScore(recorder, request)

		result := recorder.Result()

		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	})

	t.Run("should return bad request if request body is invalid", func(t *testing.T) {
		api := makeAPIMock()
		api.On("LogError", mock.Anything, mock.Anything, mock.Anything)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader([]byte("garbage")))
		request.Header.Set("Mattermost-User-ID", userID)

		p.submitScore(recorder, request)

		result := recorder.Result()

		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	})

	t.Run("should return bad request if request body is empty", func(t *testing.T) {
		api := makeAPIMock()
		api.On("LogError", mock.Anything, mock.Anything, mock.Anything)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/score", nil)
		request.Header.Set("Mattermost-User-ID", userID)

		p.submitScore(recorder, request)

		result := recorder.Result()

		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	})
}

func TestGetScore(t *testing.T) {
	for _, test := range []struct {
		Name           string
		SelectedOption string
		ExpectedScore  int64
		ExpectError    bool
	}{
		{
			Name:           "a number",
			SelectedOption: "7",
			ExpectedScore:  7,
		},
		{
			Name:           "zero",
			SelectedOption: "0",
			ExpectedScore:  0,
		},
		{
			Name:           "ten",
			SelectedOption: "10",
			ExpectedScore:  10,
		},
		{
			Name:           "too low",
			SelectedOption: "-400",
			ExpectError:    true,
		},
		{
			Name:           "too high",
			SelectedOption: "1000000",
			ExpectError:    true,
		},
		{
			Name:           "garbage",
			SelectedOption: "garbage",
			ExpectError:    true,
		},
		{
			Name:           "empty",
			SelectedOption: "",
			ExpectError:    true,
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			score, err := getScore(test.SelectedOption)

			assert.Equal(t, test.ExpectedScore, score)
			if test.ExpectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestRequiresUserId(t *testing.T) {
	t.Run("should call handler when user ID is present", func(t *testing.T) {
		called := false
		handler := func(w http.ResponseWriter, r *http.Request) {
			called = true
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Mattermost-User-ID", "1234ab")

		requiresUserID(handler)(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		assert.True(t, called)
	})

	t.Run("should return HTTP 401 when user ID is missing", func(t *testing.T) {
		called := false
		handler := func(w http.ResponseWriter, r *http.Request) {
			called = true
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		requiresUserID(handler)(recorder, request)

		assert.Equal(t, http.StatusUnauthorized, recorder.Result().StatusCode)
		assert.False(t, called)
	})
}

func TestDisableForUser(t *testing.T) {
	botUserID := model.NewId()
	userID := model.NewId()
	userSurveyKey := fmt.Sprintf(UserSurveyKey, userID)
	systemInstallDate := int64(1497898133094)

	licenseID := model.NewId()
	skuShortName := model.NewId()

	now := toDate(2018, time.April, 1)

	makeAPIMock := func() *plugintest.API {
		api := &plugintest.API{}
		api.On("LogDebug", mock.Anything).Maybe()

		// Disabling diagnostics allows the handler to run, but prevents data from being sent to Segment
		api.On("GetConfig").Return(&model.Config{
			LogSettings: model.LogSettings{
				EnableDiagnostics: model.NewBool(false),
			},
		}).Maybe()

		return api
	}

	t.Run("should disable sending for user", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", userSurveyKey).Return(mustMarshalJSON(&userSurveyState{}), nil)
		api.On("KVSet", userSurveyKey, mustMarshalJSON(&userSurveyState{
			Disabled: true,
		})).Return(nil)
		api.On("GetSystemInstallDate").Return(systemInstallDate, nil)
		api.On("GetUser", userID).Return(nil, &model.AppError{})
		api.On("GetLicense").Return(&model.License{
			Id:           licenseID,
			SkuShortName: skuShortName,
		})
		defer api.AssertExpectations(t)

		p := Plugin{
			botUserID: botUserID,
			now: func() time.Time {
				return now
			},
			tracker: telemetry.NewTracker(nil, "", "", "", "", "", false),
		}
		p.SetAPI(api)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/disable_for_user", bytes.NewReader(mustMarshalJSON(&model.PostActionIntegrationRequest{
			Context: map[string]interface{}{},
		})))
		request.Header.Set("Mattermost-User-ID", userID)

		p.disableForUser(recorder, request)

		result := recorder.Result()
		body, _ := ioutil.ReadAll(result.Body)

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.IsType(t, &model.PostActionIntegrationResponse{}, mustUnmarshalJSON(body, &model.PostActionIntegrationResponse{}))
	})
}
