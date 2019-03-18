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

		assert.Nil(t, p.getLastServerUpgrade())
	})

	t.Run("something stored", func(t *testing.T) {
		timestamp := time.Unix(1552599223, 0)

		api := &plugintest.API{}
		api.On("KVGet", SERVER_UPGRADE_KEY).Return([]byte(`{"Version": "5.10.0", "Timestamp": "2019-03-14T17:33:43-04:00"}`), nil)
		defer api.AssertExpectations(t)

		p := Plugin{}
		p.SetAPI(api)

		assert.Equal(t, &serverUpgrade{
			Version:   semver.MustParse("5.10.0"),
			Timestamp: timestamp,
		}, p.getLastServerUpgrade())
	})
}