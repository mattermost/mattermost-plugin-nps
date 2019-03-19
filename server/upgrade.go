package main

import (
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
)

type serverUpgrade struct {
	Version   semver.Version
	Timestamp time.Time
}

func (p *Plugin) getLastServerUpgrade() (*serverUpgrade, *model.AppError) {
	var upgrade *serverUpgrade
	err := p.KVGet(SERVER_UPGRADE_KEY, &upgrade)

	return upgrade, err
}

func (p *Plugin) storeServerUpgrade(upgrade *serverUpgrade) *model.AppError {
	return p.KVSet(SERVER_UPGRADE_KEY, upgrade)
}
