// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"
)

// PreferenceName is where Settings → Reaction Notifications stores the choice.
const PreferenceName = "notify"

// preferenceCategory matches the web app's getPluginPreferenceKey: "pp_" + plugin ID, cut to
// the 32 characters a preference category may have ("pp_dev.patika.reaction-notificat").
func preferenceCategory(pluginID string) string {
	category := "pp_" + pluginID
	if len(category) > 32 {
		category = category[:32]
	}
	return category
}

// mattermostAdapter implements reactions.Mattermost on top of the plugin API.
type mattermostAdapter struct {
	p *Plugin
}

func (a mattermostAdapter) GetPost(postID string) (*model.Post, error) {
	return a.p.client.Post.GetPost(postID)
}

func (a mattermostAdapter) GetUser(userID string) (*model.User, error) {
	return a.p.client.User.Get(userID)
}

func (a mattermostAdapter) GetChannel(channelID string) (*model.Channel, error) {
	return a.p.client.Channel.Get(channelID)
}

func (a mattermostAdapter) GetChannelMember(channelID, userID string) (*model.ChannelMember, error) {
	return a.p.client.Channel.GetMember(channelID, userID)
}

func (a mattermostAdapter) Preference(userID string) string {
	pref, appErr := a.p.API.GetPreferenceForUser(userID, preferenceCategory(manifest.Id), PreferenceName)
	if appErr != nil {
		return ""
	}
	return pref.Value
}

func (a mattermostAdapter) BotDM(userID, message string) error {
	if a.p.botID == "" {
		return errors.New("bot not ready")
	}
	return a.p.client.Post.DM(a.p.botID, userID, &model.Post{Message: message})
}

func (a mattermostAdapter) SiteURL() string {
	if cfg := a.p.API.GetConfig(); cfg != nil && cfg.ServiceSettings.SiteURL != nil {
		return *cfg.ServiceSettings.SiteURL
	}
	return ""
}
