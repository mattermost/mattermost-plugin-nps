// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

type serverUpgrade struct {
	ServerVersion string    `json:"server_version"`
	UpgradeAt     time.Time `json:"upgrade_at"`
}

// checkForServerUpgrade checks to see if the plugin has been ran with this server version before. If the server
// version has changed, it stores the time of upgrade and returns true. Otherwise, it returns false.
func (p *Plugin) checkForServerUpgrade(now time.Time) (bool, *model.AppError) {
	var storedUpgrade *serverUpgrade
	if err := p.KVGet(fmt.Sprintf(ServerUpgradeKey, p.serverVersion), &storedUpgrade); err != nil {
		// Failed to get stored version
		return false, err
	}

	if storedUpgrade != nil {
		// We've already seen this version before, so no upgrade has occurred
		return false, nil
	}

	// Note that this will see any major or minor version change as an upgrade, even if a downgrade has occurred

	if err := p.KVSet(fmt.Sprintf(ServerUpgradeKey, p.serverVersion), &serverUpgrade{
		ServerVersion: p.serverVersion,
		UpgradeAt:     now,
	}); err != nil {
		// Return false if we're unable to save the server version to prevent an upgrade from being seen multiple times
		return false, err
	}

	return true, nil
}
