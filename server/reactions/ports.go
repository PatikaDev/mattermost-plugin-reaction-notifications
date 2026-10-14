// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

package reactions

import (
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// Store keeps pending groups. CompareAndSet must be atomic; a nil p deletes the entry.
type Store interface {
	Get(postID string) (*Pending, []byte, error) // nil, nil, nil when missing
	CompareAndSet(postID string, old []byte, p *Pending) (bool, error)
	PostIDs() ([]string, error)
}

// Mattermost is the slice of the server API the service needs.
type Mattermost interface {
	GetPost(postID string) (*model.Post, error)
	GetUser(userID string) (*model.User, error)
	GetChannel(channelID string) (*model.Channel, error)
	GetChannelMember(channelID, userID string) (*model.ChannelMember, error)
	// Preference returns the user's Reaction Notifications setting, or "" if unset.
	Preference(userID string) string
	// BotDM sends a direct message from the plugin's bot.
	BotDM(userID, message string) error
	SiteURL() string
}

// Clock is the time source (fixed in tests).
type Clock interface {
	Now() time.Time
}
