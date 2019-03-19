package main

import (
	"encoding/json"
	"time"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) CreateBotDMPost(userID, message, postType string, props map[string]interface{}) (*model.Post, *model.AppError) {
	channel, err := p.API.GetDirectChannel(userID, p.botUserId)
	if err != nil {
		p.API.LogError("Couldn't get bot's DM channel", "user_id", userID, "err", err)
		return nil, err
	}

	if props == nil {
		props = make(map[string]interface{})
	}
	props["from_webhook"] = true

	post, err := p.API.CreatePost(&model.Post{
		UserId:    p.botUserId,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props:     props,
	})
	if err != nil {
		p.API.LogError("Couldn't send bot DM", "user_id", userID, "err", err)
		return nil, err
	}

	return post, nil
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
