package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	routes := []struct {
		Path    string
		Method  string
		Handler func(http.ResponseWriter, *http.Request)
	}{
		{
			Path:    "/api/v1/connected",
			Method:  http.MethodPost,
			Handler: p.userConnected,
		},
		{
			Path:    "/api/v1/score",
			Method:  http.MethodPost,
			Handler: p.submitScore,
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

func (p *Plugin) userConnected(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err := p.checkForDMs(userID)
	if err != nil {
		p.API.LogError("Failed to check for user notifications", "user_id", userID, "err", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) checkForDMs(userID string) *model.AppError {
	if p.canSendDiagnostics() {
		user, err := p.API.GetUser(userID)
		if err != nil {
			return err
		}

		now := time.Now()

		go func() {
			// Add a random delay to mitigate against the fact that the user may have multiple sessions hitting this
			// API at the same time across different servers.
			p.sleepUpTo(p.userSurveyMaxDelay)

			p.connectedLock.Lock()
			defer p.connectedLock.Unlock()

			if notice := p.checkForAdminNoticeDM(user); notice != nil {
				p.sendAdminNoticeDM(user, notice)
			}

			if p.shouldSendSurveyDM(user, p.getServerVersion(), now) {
				p.sendSurveyDM(user, now)
			}
		}()
	}

	return nil
}

func (p *Plugin) submitScore(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

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

	if _, ok := surveyResponse.Context["selected_option"].(string); !ok {
		p.API.LogError("Score response contains invalid score")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := p.API.GetUser(userID)
	if err != nil {
		p.API.LogError("Failed to get user", "user_id", userID, "err", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var score int
	if i, err := strconv.ParseInt(surveyResponse.Context["selected_option"].(string), 10, 0); err != nil {
		p.API.LogError("Score response contains invalid score")
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if i < 0 || i > 10 {
		p.API.LogError("Score response contains invalid score")
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		score = int(i)
	}

	p.API.LogDebug(fmt.Sprintf("Received score of %d from %s", score, r.Header.Get("Mattermost-User-ID")))

	now := time.Now().UTC()

	p.sendScore(score, userID, now.UnixNano()/int64(time.Millisecond))

	if err := p.markSurveyAnswered(userID, now); err != nil {
		p.API.LogWarn("Failed to mark survey as answered", "err", err)
	}

	p.CreateBotDMPost(userID, p.buildFeedbackRequestPost())

	// Send response to update score post
	response := model.PostActionIntegrationResponse{
		Update: p.buildAnsweredSurveyPost(user, score),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response.ToJson())
}
