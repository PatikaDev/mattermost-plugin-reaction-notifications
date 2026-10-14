// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

package main

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/pkg/errors"

	"github.com/PatikaDev/mattermost-plugin-reaction-notifications/server/reactions"
)

// flushEvery is how often grouped reactions are checked; with reactions.GroupFor this sends
// a notification 60–80 seconds after the first reaction.
const flushEvery = 20 * time.Second

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	client  *pluginapi.Client
	service *reactions.Service
	botID   string
	flusher *cluster.Job

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    "reactions",
		DisplayName: "Reactions",
		Description: "Tells you when someone reacts to your message. Change it in Settings → Reaction Notifications.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot")
	}
	p.botID = botID

	p.service = reactions.NewService(
		reactions.NewKVStore(&p.client.KV),
		mattermostAdapter{p: p},
		realClock{},
		func(msg string, keyValuePairs ...any) { p.API.LogWarn(msg, keyValuePairs...) },
	)

	// cluster.Schedule runs the job on exactly one node of a cluster at a time.
	job, err := cluster.Schedule(p.API, "FlushReactionNotifications", cluster.MakeWaitForInterval(flushEvery), p.service.Flush)
	if err != nil {
		return errors.Wrap(err, "failed to schedule the reaction sender")
	}
	p.flusher = job
	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	if p.flusher != nil {
		if err := p.flusher.Close(); err != nil {
			p.API.LogError("Failed to stop the reaction sender", "err", err)
		}
	}
	return nil
}

// ReactionHasBeenAdded is invoked after a reaction is saved.
func (p *Plugin) ReactionHasBeenAdded(_ *plugin.Context, reaction *model.Reaction) {
	if p.service != nil {
		p.service.OnAdded(reaction)
	}
}

// ReactionHasBeenRemoved is invoked after a reaction is removed.
func (p *Plugin) ReactionHasBeenRemoved(_ *plugin.Context, reaction *model.Reaction) {
	if p.service != nil {
		p.service.OnRemoved(reaction)
	}
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
