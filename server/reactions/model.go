// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

// Package reactions turns emoji reactions into a grouped direct message to the post's
// author. Mattermost, the KV store and the clock are behind interfaces so all of it is
// unit-testable.
package reactions

import "time"

// GroupFor is how long reactions to the same post are collected before one DM is sent.
const GroupFor = time.Minute

// Preference values (Settings → Reaction Notifications).
const (
	NotifyAll = "all" // every message (the default)
	NotifyDMs = "dms" // only direct and group messages
	NotifyOff = "off"
)

// Reaction is one emoji from one person.
type Reaction struct {
	UserID string `json:"user_id"`
	Emoji  string `json:"emoji"`
}

// Pending collects reactions to one post until they are sent.
type Pending struct {
	PostID    string     `json:"post_id"`
	AuthorID  string     `json:"author_id"`
	ChannelID string     `json:"channel_id"`
	FirstAt   int64      `json:"first_at"` // Unix milliseconds
	Reactions []Reaction `json:"reactions"`
}
