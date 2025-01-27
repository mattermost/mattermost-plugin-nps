// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// getServerVersion returns the current server version with only the major and minor version set. For example, both
// 5.10.0 and 5.10.1 will be returned as "5.10.0" by this method.
func getServerVersion(serverVersion string) string {
	return regexp.MustCompile(`\.[1-9]\d*$`).ReplaceAllString(serverVersion, ".0")
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
	if channel.Type != model.ChannelTypeDirect {
		return false
	}

	if !strings.HasPrefix(channel.Name, p.botUserID+"__") && !strings.HasSuffix(channel.Name, "__"+p.botUserID) {
		return false
	}

	return true
}
