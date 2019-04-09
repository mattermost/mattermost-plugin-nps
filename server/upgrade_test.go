package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/assert"
)

func TestCheckForServerUpgrade(t *testing.T) {
	now := time.Unix(1552599223, 0).UTC()
	version := "5.10.0"

	t.Run("should return true when an upgrade has occurred", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetServerVersion").Return(version)
		api.On("KVGet", fmt.Sprintf(SERVER_UPGRADE_KEY, version)).Return(nil, nil)
		api.On("KVSet", fmt.Sprintf(SERVER_UPGRADE_KEY, version), mustMarshalJSON(&serverUpgrade{
			Version:   semver.MustParse(version),
			UpgradeAt: now,
		})).Return(nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		upgraded, err := p.checkForServerUpgrade(now)

		assert.True(t, upgraded)
		assert.Nil(t, err)
	})

	t.Run("should return false when an upgrade has not occurred", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetServerVersion").Return(version)
		api.On("KVGet", fmt.Sprintf(SERVER_UPGRADE_KEY, version)).Return(mustMarshalJSON(&serverUpgrade{
			Version:   semver.MustParse(version),
			UpgradeAt: now,
		}), nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		upgraded, err := p.checkForServerUpgrade(now)

		assert.False(t, upgraded)
		assert.Nil(t, err)
	})

	t.Run("should return an error if unable to get the current server version", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetServerVersion").Return("garbage")
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		upgraded, err := p.checkForServerUpgrade(now)

		assert.False(t, upgraded)
		assert.NotNil(t, err)
	})

	t.Run("should return an error if unable to get the stored server version", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetServerVersion").Return(version)
		api.On("KVGet", fmt.Sprintf(SERVER_UPGRADE_KEY, version)).Return(nil, &model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		upgraded, err := p.checkForServerUpgrade(now)

		assert.False(t, upgraded)
		assert.NotNil(t, err)
	})

	t.Run("should return an error if unable to store the new server version", func(t *testing.T) {
		api := makeAPIMock()
		api.On("GetServerVersion").Return(version)
		api.On("KVGet", fmt.Sprintf(SERVER_UPGRADE_KEY, version)).Return(nil, nil)
		api.On("KVSet", fmt.Sprintf(SERVER_UPGRADE_KEY, version), mustMarshalJSON(&serverUpgrade{
			Version:   semver.MustParse(version),
			UpgradeAt: now,
		})).Return(&model.AppError{})
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		upgraded, err := p.checkForServerUpgrade(now)

		assert.False(t, upgraded)
		assert.NotNil(t, err)
	})
}
