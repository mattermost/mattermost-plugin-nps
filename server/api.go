package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

type apiHandler func(w http.ResponseWriter, r *http.Request)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	routes := []struct {
		Path    string
		Method  string
		Handler apiHandler
	}{
		{
			Path:    "/api/v1/connected",
			Method:  http.MethodPost,
			Handler: requiresUserId(p.userConnected),
		},
		{
			Path:    "/api/v1/score",
			Method:  http.MethodPost,
			Handler: requiresUserId(p.submitScore),
		},
		{
			Path:    "/api/v1/disable_for_user",
			Method:  http.MethodPost,
			Handler: requiresUserId(p.disableForUser),
		},
	}

	routeFound := false

	for _, route := range routes {
		if r.URL.Path == route.Path && r.Method == route.Method {
			route.Handler(w, r)
			routeFound = true

			break
		}
	}

	if !routeFound {
		http.NotFound(w, r)
	}
}

func (p *Plugin) disableForUser(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")

	var userSurvey *userSurveyState
	if err := p.KVGet(fmt.Sprintf(USER_SURVEY_KEY, userID), &userSurvey); err != nil {
		p.API.LogError("Failed to get survey state", "user_id", userID, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userSurvey.Disabled = true

	if err := p.KVSet(fmt.Sprintf(USER_SURVEY_KEY, userID), userSurvey); err != nil {
		p.API.LogError("Failed to set disabled survey state", "user_id", userID, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}

func (p *Plugin) userConnected(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")

	err := p.checkForDMs(userID)
	if err != nil {
		p.API.LogError("Failed to check for user notifications", "user_id", userID, "err", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) checkForDMs(userID string) *model.AppError {
	if !p.canSendDiagnostics() {
		return nil
	}

	now := p.now().UTC()
	userLockKey := fmt.Sprintf(USER_LOCK_KEY, userID)

	locked, err := p.tryLock(userLockKey, now)
	if !locked || err != nil {
		// Either an error occurred or there's already another thread checking for DMs
		return err
	}
	defer p.unlock(userLockKey)

	user, err := p.API.GetUser(userID)
	if err != nil {
		return err
	}

	if _, err := p.checkForAdminNoticeDM(user); err != nil {
		p.API.LogError("Failed to check for notice of scheduled survey for user", "err", err, "user_id", userID)
	}

	if _, err := p.checkForSurveyDM(user, now); err != nil {
		p.API.LogError("Failed to check for survey for user", "err", err, "user_id", userID)
	}

	return nil
}

func (p *Plugin) submitScore(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")

	var surveyResponse *model.PostActionIntegrationRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 2048)).Decode(&surveyResponse); err != nil {
		p.API.LogError("Failed to decode survey score response", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if surveyResponse.Context == nil {
		p.API.LogError("Score response is missing Context")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		p.API.LogError("Failed to get user", "user_id", userID, "err", appErr)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var score int
	if i, err := getScore(surveyResponse.Context["selected_option"].(string)); err != nil {
		p.API.LogError("Score response contains invalid score")
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		score = int(i)
	}

	p.API.LogDebug(fmt.Sprintf("Received score of %d from %s", score, r.Header.Get("Mattermost-User-ID")))

	now := p.now().UTC()

	p.sendScore(score, userID, now.UnixNano()/int64(time.Millisecond))

	isFirstResponse, appErr := p.markSurveyAnswered(userID, now)
	if appErr != nil {
		p.API.LogWarn("Failed to mark survey as answered", "err", appErr)
	}

	// Thank the user for their feedback when they first answer the survey
	if isFirstResponse {
		p.CreateBotDMPost(userID, p.buildFeedbackRequestPost())
	}

	// Send response to update score post
	response := model.PostActionIntegrationResponse{
		Update: p.buildAnsweredSurveyPost(user, score),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response.ToJson())
}

func getScore(selectedOption string) (int64, error) {
	score, err := strconv.ParseInt(selectedOption, 10, 0)
	if err != nil {
		return 0, err
	}

	if score < 0 || score > 10 {
		return 0, errors.New("score out of range")
	}

	return score, nil
}

func requiresUserId(handler apiHandler) apiHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		if userID := r.Header.Get("Mattermost-User-ID"); userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}
