package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

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
			Path:    "/api/v1/score/9tq3aohzpfg5prbxzyqrhjc7ih",
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
			p.sendSurveyDM(user, now)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (p *Plugin) submitScore(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	p.API.LogDebug(fmt.Sprintf("Received score of %s from %s", body, r.Header.Get("Mattermost-User-ID")))
	w.Write([]byte("{}"))
}
