package main

import (
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestGetLastServerUpgrade(t *testing.T) {
	t.Run("nothing stored", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", SERVER_UPGRADE_KEY).Return(nil, nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		upgrade, err := p.getLastServerUpgrade()
		assert.Nil(t, upgrade)
		assert.Nil(t, err)
	})

	t.Run("something stored", func(t *testing.T) {
		timestamp := time.Unix(1552599223, 0).UTC()

		api := &plugintest.API{}
		api.On("KVGet", SERVER_UPGRADE_KEY).Return(mustMarshalJSON(&serverUpgrade{
			Version:   semver.MustParse("5.10.0"),
			Timestamp: timestamp,
		}), nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		upgrade, err := p.getLastServerUpgrade()
		assert.Equal(t, &serverUpgrade{
			Version:   semver.MustParse("5.10.0"),
			Timestamp: timestamp,
		}, upgrade)
		assert.Nil(t, err)
	})
}
