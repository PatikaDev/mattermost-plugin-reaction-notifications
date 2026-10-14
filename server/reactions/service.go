// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

package reactions

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	casRetries   = 5
	quoteRunes   = 200
	namedReactor = 2 // "@a, @b and 3 others"
)

type Service struct {
	store Store
	mm    Mattermost
	clock Clock
	log   func(msg string, keyValuePairs ...any)
}

func NewService(store Store, mm Mattermost, clock Clock, log func(msg string, keyValuePairs ...any)) *Service {
	return &Service{store: store, mm: mm, clock: clock, log: log}
}

// OnAdded records a reaction for the post's author, unless it shouldn't notify anyone.
func (s *Service) OnAdded(r *model.Reaction) {
	post, err := s.mm.GetPost(r.PostId)
	if err != nil || post == nil || post.DeleteAt != 0 || post.UserId == r.UserId || post.IsSystemMessage() {
		return
	}
	if reactor, err := s.mm.GetUser(r.UserId); err != nil || reactor == nil || reactor.IsBot {
		return
	}
	if author, err := s.mm.GetUser(post.UserId); err != nil || author == nil || author.IsBot || author.DeleteAt != 0 {
		return
	}
	if s.preference(post.UserId) == NotifyOff {
		return
	}

	s.update(post.Id, func(p *Pending) *Pending {
		if p == nil {
			p = &Pending{PostID: post.Id, AuthorID: post.UserId, ChannelID: post.ChannelId, FirstAt: s.clock.Now().UnixMilli()}
		}
		for _, existing := range p.Reactions {
			if existing.UserID == r.UserId && existing.Emoji == r.EmojiName {
				return p
			}
		}
		p.Reactions = append(p.Reactions, Reaction{UserID: r.UserId, Emoji: r.EmojiName})
		return p
	})
}

// OnRemoved forgets a reaction that was taken back before the group was sent.
func (s *Service) OnRemoved(r *model.Reaction) {
	s.update(r.PostId, func(p *Pending) *Pending {
		if p == nil {
			return nil
		}
		kept := p.Reactions[:0:0]
		for _, existing := range p.Reactions {
			if existing.UserID != r.UserId || existing.Emoji != r.EmojiName {
				kept = append(kept, existing)
			}
		}
		if len(kept) == 0 {
			return nil
		}
		p.Reactions = kept
		return p
	})
}

// update applies change atomically, retrying when another reaction landed at the same time.
func (s *Service) update(postID string, change func(*Pending) *Pending) {
	for range casRetries {
		current, raw, err := s.store.Get(postID)
		if err != nil {
			s.log("Failed to read pending reactions", "post_id", postID, "error", err.Error())
			return
		}
		next := change(current)
		if current == nil && next == nil {
			return
		}
		ok, err := s.store.CompareAndSet(postID, raw, next)
		if err != nil {
			s.log("Failed to save pending reactions", "post_id", postID, "error", err.Error())
			return
		}
		if ok {
			return
		}
	}
	s.log("Gave up saving pending reactions after concurrent updates", "post_id", postID)
}

// Flush sends every group that has been collecting for GroupFor. Called by the cluster job.
func (s *Service) Flush() {
	ids, err := s.store.PostIDs()
	if err != nil {
		s.log("Failed to list pending reactions", "error", err.Error())
		return
	}
	now := s.clock.Now().UnixMilli()
	for _, id := range ids {
		p, raw, err := s.store.Get(id)
		if err != nil || p == nil || now-p.FirstAt < GroupFor.Milliseconds() {
			continue
		}
		// Claim the group by deleting it; if that fails another server got it first.
		if ok, err := s.store.CompareAndSet(id, raw, nil); err != nil || !ok {
			continue
		}
		s.deliver(p)
	}
}

func (s *Service) deliver(p *Pending) {
	author, err := s.mm.GetUser(p.AuthorID)
	if err != nil || author == nil || author.DeleteAt != 0 {
		return
	}
	pref := s.preference(p.AuthorID)
	if pref == NotifyOff {
		return
	}
	channel, err := s.mm.GetChannel(p.ChannelID)
	if err != nil || channel == nil {
		return
	}
	if pref == NotifyDMs && channel.Type != model.ChannelTypeDirect && channel.Type != model.ChannelTypeGroup {
		return
	}
	member, err := s.mm.GetChannelMember(p.ChannelID, p.AuthorID)
	if err != nil || member == nil {
		return // left the channel
	}
	if member.NotifyProps[model.MarkUnreadNotifyProp] == model.ChannelMarkUnreadMention {
		return // muted
	}
	post, err := s.mm.GetPost(p.PostID)
	if err != nil || post == nil || post.DeleteAt != 0 {
		return
	}
	if err := s.mm.BotDM(p.AuthorID, s.message(p, post, channel)); err != nil {
		s.log("Failed to send reaction notification", "post_id", p.PostID, "error", err.Error())
	}
}

// message reads e.g. ":+1: :tada: **@ayse**, **@mehmet** and 2 others reacted to your message
// in ~marketing", then a quote of the message and a link to it.
func (s *Service) message(p *Pending, post *model.Post, channel *model.Channel) string {
	var emojis, users []string
	seenEmoji, seenUser := map[string]bool{}, map[string]bool{}
	for _, r := range p.Reactions {
		if !seenEmoji[r.Emoji] {
			seenEmoji[r.Emoji] = true
			emojis = append(emojis, ":"+r.Emoji+":")
		}
		if !seenUser[r.UserID] {
			seenUser[r.UserID] = true
			users = append(users, r.UserID)
		}
	}

	var names []string
	for _, id := range users {
		if len(names) == namedReactor && len(users) > namedReactor+1 {
			break
		}
		name := "someone"
		if u, err := s.mm.GetUser(id); err == nil && u != nil {
			name = "**@" + u.Username + "**"
		}
		names = append(names, name)
	}
	var who string
	switch others := len(users) - len(names); {
	case others > 0:
		who = fmt.Sprintf("%s and %d others", strings.Join(names, ", "), others)
	case len(names) == 1:
		who = names[0]
	default:
		who = strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}

	what := "your message"
	if post.RootId != "" {
		what = "your reply"
	}
	where := "in a group message"
	switch channel.Type {
	case model.ChannelTypeOpen, model.ChannelTypePrivate:
		where = "in ~" + channel.Name
	case model.ChannelTypeDirect:
		where = "in a direct message"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s reacted to %s %s\n", strings.Join(emojis, " "), who, what, where)
	b.WriteString("> " + quote(post) + "\n\n")
	fmt.Fprintf(&b, "[Open message](%s/_redirect/pl/%s)", strings.TrimRight(s.mm.SiteURL(), "/"), post.Id)
	return b.String()
}

// quote is the first line-ish of the post, shortened, on one line.
func quote(post *model.Post) string {
	text := strings.Join(strings.Fields(post.Message), " ")
	if text == "" {
		if len(post.FileIds) > 0 {
			return "_(attachment)_"
		}
		return "_(no text)_"
	}
	if utf8.RuneCountInString(text) > quoteRunes {
		text = string([]rune(text)[:quoteRunes]) + "…"
	}
	return text
}

func (s *Service) preference(userID string) string {
	switch v := s.mm.Preference(userID); v {
	case NotifyDMs, NotifyOff:
		return v
	default:
		return NotifyAll
	}
}
