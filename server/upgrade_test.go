package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestCheckForServerUpgrade(t *testing.T) {
	now := time.Unix(1552599223, 0).UTC()
	serverVersion := "5.10.0"

	makePlugin := func(api *plugintest.API) *Plugin {
		p := &Plugin{
			serverVersion: serverVersion,
		}
		p.SetAPI(api)

		return p
	}

	t.Run("should return true when an upgrade has occurred", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ServerUpgradeKey, serverVersion)).Return(nil, nil)
		api.On("KVSet", fmt.Sprintf(ServerUpgradeKey, serverVersion), mustMarshalJSON(&serverUpgrade{
			ServerVersion: serverVersion,
			UpgradeAt:     now,
		})).Return(nil)
		defer api.AssertExpectations(t)

		p := makePlugin(api)
		upgraded, err := p.checkForServerUpgrade(now)

		assert.True(t, upgraded)
		assert.Nil(t, err)
	})

	t.Run("should return false when an upgrade has not occurred", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ServerUpgradeKey, serverVersion)).Return(mustMarshalJSON(&serverUpgrade{
			ServerVersion: serverVersion,
			UpgradeAt:     now,
		}), nil)
		defer api.AssertExpectations(t)

		p := makePlugin(api)
		upgraded, err := p.checkForServerUpgrade(now)

		assert.False(t, upgraded)
		assert.Nil(t, err)
	})

	t.Run("should return an error if unable to get the stored server version", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ServerUpgradeKey, serverVersion)).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := makePlugin(api)
		upgraded, err := p.checkForServerUpgrade(now)

		assert.False(t, upgraded)
		assert.NotNil(t, err)
	})

	t.Run("should return an error if unable to store the new server version", func(t *testing.T) {
		api := makeAPIMock()
		api.On("KVGet", fmt.Sprintf(ServerUpgradeKey, serverVersion)).Return(nil, nil)
		api.On("KVSet", fmt.Sprintf(ServerUpgradeKey, serverVersion), mustMarshalJSON(&serverUpgrade{
			ServerVersion: serverVersion,
			UpgradeAt:     now,
		})).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		p := makePlugin(api)
		upgraded, err := p.checkForServerUpgrade(now)

		assert.False(t, upgraded)
		assert.NotNil(t, err)
	})
}
