package main

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	// LockKey is used to prevent multiple instances of the plugin from scheduling surveys in parallel.
	LockKey = "Lock"

	// UserLockKey is used to prevent multiple instances of the plugin from responding to a single user's requests
	// in parallel.
	UserLockKey = "UserLock-%s"

	// LockExpiration is how long a lock can be held before it will be automatically released the next time that an
	// instance of the plugin is started up.
	LockExpiration = time.Hour
)

var userLockPattern = regexp.MustCompile("^UserLock-.{26}$")

func (p *Plugin) tryLock(key string, now time.Time) (bool, *model.AppError) {
	b, err := json.Marshal(now)
	if err != nil {
		return false, &model.AppError{Message: err.Error()}
	}

	return p.API.KVCompareAndSet(key, nil, b)
}

func (p *Plugin) unlock(key string) *model.AppError {
	return p.API.KVDelete(key)
}

// clearStaleLocks deletes any lock entries that have been held for a long time since that likely means that the routine
// that held them died without properly releasing them.
func (p *Plugin) clearStaleLocks(now time.Time) *model.AppError {
	page := 0
	perPage := 100

	for {
		keys, err := p.API.KVList(page, perPage)
		if err != nil {
			return err
		}

		for _, key := range keys {
			if key != LockKey && !userLockPattern.MatchString(key) {
				continue
			}

			value, err := p.API.KVGet(key)
			if err != nil {
				return err
			}

			// Ignore any unmarshaling error in case the lock has gotten stuck in a really bad state
			var t time.Time
			_ = json.Unmarshal(value, &t)

			if now.Sub(t) >= LockExpiration {
				deleted, err := p.API.KVCompareAndDelete(key, value)
				if err != nil {
					return err
				}
				if deleted {
					p.API.LogInfo("Freed expired lock", "key", key)
				}
			}
		}

		if len(keys) < perPage {
			break
		}

		page++
	}

	return nil
}
