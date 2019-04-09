package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) GetServerVersion() (semver.Version, *model.AppError) {
	versionString := p.API.GetServerVersion()

	version, err := semver.Parse(versionString)
	if err != nil {
		return version, &model.AppError{Message: err.Error()}
	}

	// Remove the parts of the version that the NPS plugin doesn't care about
	version.Patch = 0
	version.Pre = nil
	version.Build = nil

	return version, nil
}

func (p *Plugin) KVGet(key string, v interface{}) *model.AppError {
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		return appErr
	}

	if data == nil {
		return nil
	}

	if err := json.Unmarshal(data, v); err != nil {
		return &model.AppError{Message: fmt.Sprintf("Unable to deserialize value %s for key %s, err=%s", data, key, err)}
	}

	return nil
}

func (p *Plugin) KVSet(key string, v interface{}) *model.AppError {
	data, err := json.Marshal(v)
	if err != nil {
		return &model.AppError{Message: err.Error()}
	}

	return p.API.KVSet(key, data)
}

func (p *Plugin) CreateBotDMPost(userID string, post *model.Post) (*model.Post, *model.AppError) {
	channel, err := p.API.GetDirectChannel(userID, p.botUserID)
	if err != nil {
		p.API.LogError("Couldn't get bot's DM channel", "user_id", userID, "err", err)
		return nil, err
	}

	post.UserId = p.botUserID
	post.ChannelId = channel.Id

	created, err := p.API.CreatePost(post)
	if err != nil {
		p.API.LogError("Couldn't send bot DM", "user_id", userID, "err", err)
		return nil, err
	}

	return created, nil
}

func (p *Plugin) IsBotDMChannel(channel *model.Channel) bool {
	if channel.Type != model.CHANNEL_DIRECT {
		return false
	}

	if !strings.HasPrefix(channel.Name, p.botUserID+"__") && !strings.HasSuffix(channel.Name, "__"+p.botUserID) {
		return false
	}

	return true
}

func (p *Plugin) sleepUpTo(maxDelay time.Duration) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	delay := time.Duration(r.Int63n(int64(maxDelay)))

	time.Sleep(delay)
}
