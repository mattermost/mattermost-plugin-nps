package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) CreateBotDMPost(userID string, post *model.Post) (*model.Post, *model.AppError) {
	channel, err := p.API.GetDirectChannel(userID, p.botUserId)
	if err != nil {
		p.API.LogError("Couldn't get bot's DM channel", "user_id", userID, "err", err)
		return nil, err
	}

	post.UserId = p.botUserId
	post.ChannelId = channel.Id

	if post.Props == nil {
		post.Props = make(map[string]interface{})
	}
	post.Props["from_webhook"] = true

	created, err := p.API.CreatePost(post)
	if err != nil {
		p.API.LogError("Couldn't send bot DM", "user_id", userID, "err", err)
		return nil, err
	}

	return created, nil
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
		return &model.AppError{Message: err.Error()}
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

func (p *Plugin) isBotDMChannel(channel *model.Channel) bool {
	if channel.Type != model.CHANNEL_DIRECT {
		return false
	}

	if !strings.HasPrefix(channel.Name, p.botUserId+"__") && !strings.HasSuffix(channel.Name, "__"+p.botUserId) {
		return false
	}

	return true
}

// Test helper functions

func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return data
}

func mustUnmarshalJSON(data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}

func toDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
