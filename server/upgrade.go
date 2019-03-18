package main

import (
	"encoding/json"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
)

type serverUpgrade struct {
	Version   semver.Version
	Timestamp time.Time
}

func (p *Plugin) getLastServerUpgrade() *serverUpgrade {
	upgradeBytes, appErr := p.API.KVGet(SERVER_UPGRADE_KEY)
	if appErr != nil || upgradeBytes == nil {
		return nil
	}

	var upgrade serverUpgrade

	err := json.Unmarshal(upgradeBytes, &upgrade)
	if err != nil {
		return nil
	}

	return &upgrade
}

func (p *Plugin) storeServerUpgrade(upgrade *serverUpgrade) *model.AppError {
	upgradeBytes, err := json.Marshal(upgrade)
	if err != nil {
		return &model.AppError{Message: err.Error()}
	}

	return p.API.KVSet(SERVER_UPGRADE_KEY, upgradeBytes)
}
