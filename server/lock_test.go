package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTryLock(t *testing.T) {
	now := toDate(2019, time.February, 18)

	api := &plugintest.API{}
	api.On("KVCompareAndSet", LOCK_KEY, []byte(nil), mustMarshalJSON(now)).Return(true, nil)
	defer api.AssertExpectations(t)

	p := &Plugin{}
	p.SetAPI(api)

	locked, err := p.tryLock(LOCK_KEY, now)

	assert.Equal(t, true, locked)
	assert.Nil(t, err)
}

func TestUnlock(t *testing.T) {
	api := &plugintest.API{}
	api.On("KVDelete", LOCK_KEY).Return(nil)
	defer api.AssertExpectations(t)

	p := &Plugin{}
	p.SetAPI(api)

	err := p.unlock(LOCK_KEY)

	assert.Nil(t, err)
}

func TestClearStaleLocks(t *testing.T) {
	now := toDate(2019, time.February, 18)
	serverVersion := "5.10.0"
	userID := model.NewId()

	userLockKey := fmt.Sprintf(USER_LOCK_KEY, userID)

	t.Run("shouldn't affect KV store entries that aren't locks", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVList", 0, 100).Return([]string{
			fmt.Sprintf(ADMIN_DM_NOTICE_KEY, userID, serverVersion),
			LAST_ADMIN_NOTICE_KEY,
			fmt.Sprintf(SERVER_UPGRADE_KEY, serverVersion),
			fmt.Sprintf(SURVEY_KEY, serverVersion),
			fmt.Sprintf(USER_SURVEY_KEY, userID),
			"something else",
		}, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.SetAPI(api)

		err := p.clearStaleLocks(now)

		assert.Nil(t, err)
	})

	t.Run("shouldn't affect locks that were acquired recently", func(t *testing.T) {
		lockValue := mustMarshalJSON(now.Add(-1 * time.Minute))
		userLockValue := mustMarshalJSON(now.Add(-5 * time.Minute))

		api := &plugintest.API{}
		api.On("KVList", 0, 100).Return([]string{
			LOCK_KEY,
			userLockKey,
		}, nil)
		api.On("KVGet", LOCK_KEY).Return(lockValue, nil)
		api.On("KVGet", userLockKey).Return(userLockValue, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.SetAPI(api)

		err := p.clearStaleLocks(now)

		assert.Nil(t, err)
	})

	t.Run("should clear locks that were acquired too long ago", func(t *testing.T) {
		lockValue := mustMarshalJSON(now.Add(-1 * time.Hour))
		userLockValue := mustMarshalJSON(now.Add(-5 * time.Hour))

		api := &plugintest.API{}
		api.On("KVList", 0, 100).Return([]string{
			LOCK_KEY,
			userLockKey,
		}, nil)
		api.On("KVGet", LOCK_KEY).Return(lockValue, nil)
		api.On("KVCompareAndDelete", LOCK_KEY, lockValue).Return(true, nil)
		api.On("KVGet", userLockKey).Return(userLockValue, nil)
		api.On("KVCompareAndDelete", userLockKey, userLockValue).Return(true, nil)
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.SetAPI(api)

		err := p.clearStaleLocks(now)

		assert.Nil(t, err)
	})

	t.Run("should clear locks that weren't properly freed before", func(t *testing.T) {
		lockValue := []byte("releasing")

		api := &plugintest.API{}
		api.On("KVList", 0, 100).Return([]string{
			LOCK_KEY,
		}, nil)
		api.On("KVGet", LOCK_KEY).Return(lockValue, nil)
		api.On("KVCompareAndDelete", LOCK_KEY, lockValue).Return(true, nil)
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.SetAPI(api)

		err := p.clearStaleLocks(now)

		assert.Nil(t, err)
	})

	t.Run("should not try to delete an old lock that was modified by another thread", func(t *testing.T) {
		lockValue := mustMarshalJSON(now.Add(-1 * time.Hour))

		api := &plugintest.API{}
		api.On("KVList", 0, 100).Return([]string{
			LOCK_KEY,
		}, nil)
		api.On("KVGet", LOCK_KEY).Return(lockValue, nil)
		api.On("KVCompareAndDelete", LOCK_KEY, lockValue).Return(false, nil)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.SetAPI(api)

		err := p.clearStaleLocks(now)

		assert.Nil(t, err)
	})

	t.Run("should check multiple pages of keys", func(t *testing.T) {
		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			keys[i] = fmt.Sprintf("key%d", i)
		}

		api := &plugintest.API{}
		api.On("KVList", 0, 100).Return(keys, nil)
		api.On("KVList", 1, 100).Return(keys, nil)
		api.On("KVList", 2, 100).Return(keys[:40], nil)
		defer api.AssertExpectations(t)

		p := &Plugin{}
		p.SetAPI(api)

		err := p.clearStaleLocks(now)

		assert.Nil(t, err)
	})
}
