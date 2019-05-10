package main

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/mattermost/mattermost-server/model"
)

const (
	// LOCK_KEY is used to prevent multiple instances of the plugin from scheduling surveys in parallel.
	LOCK_KEY = "Lock"

	// USER_LOCK_KEY is used to prevent multiple instances of the plugin from responding to a single user's requests
	// in parallel.
	USER_LOCK_KEY = "UserLock-%s"

	// LOCK_EXPIRATION is how long a lock can be held before it will be automatically released the next time that an
	// instance of the plugin is started up.
	LOCK_EXPIRATION = time.Hour
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
			if key != LOCK_KEY && !userLockPattern.MatchString(key) {
				continue
			}

			value, err := p.API.KVGet(key)
			if err != nil {
				return err
			}

			// Ignore any unmarshaling error in case the lock has gotten stuck in a really bad state
			var t time.Time
			_ = json.Unmarshal(value, &t)

			if now.Sub(t) >= LOCK_EXPIRATION {
				// There's no KVCompareAndDelete, so make sure we're the only ones modifying this lock
				if set, err := p.API.KVCompareAndSet(key, value, []byte("releasing")); err != nil {
					return err
				} else if !set {
					// Someone else has released or cleared the lock
					continue
				}

				p.API.LogInfo("Freeing expired NPS lock", "key", key)

				if err := p.API.KVDelete(key); err != nil {
					return err
				}
			}
		}

		if len(keys) < perPage {
			break
		}

		page += 1
	}

	return nil
}
