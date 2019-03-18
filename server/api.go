package main

import (
	"net/http"
	"time"

	"github.com/mattermost/mattermost-server/plugin"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	routes := []struct{
		Path    string
		Method  string
		Handler func(http.ResponseWriter, *http.Request)
	}{
		{
			Path:    "/api/v1/connected",
			Method:  http.MethodPost,
			Handler: p.userConnected,
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

	p.connectedLock.Lock()
	defer p.connectedLock.Unlock()

	if p.canSendDiagnostics() {
		user, err := p.API.GetUser(userID)
		if err != nil {
			p.API.LogError("Failed to get user", "user_id", userID, "err", err)

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		now := time.Now()

		if notice := p.checkForAdminNoticeDM(user); notice != nil {
			p.sendAdminNoticeDM(user, notice)
		}

		if p.shouldSendSurveyDM(user, now) {
			p.sendSurveyDM(user)
		}
	}

	w.WriteHeader(http.StatusOK)
}