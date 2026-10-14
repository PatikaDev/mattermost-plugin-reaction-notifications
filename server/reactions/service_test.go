// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

package reactions

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type memStore struct {
	data     map[string][]byte
	failNext int // number of CompareAndSet calls to fail as if someone else wrote first
}

func newMemStore() *memStore { return &memStore{data: map[string][]byte{}} }

func (m *memStore) Get(id string) (*Pending, []byte, error) {
	raw, ok := m.data[id]
	if !ok {
		return nil, nil, nil
	}
	var p Pending
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, nil, err
	}
	return &p, raw, nil
}

func (m *memStore) CompareAndSet(id string, old []byte, p *Pending) (bool, error) {
	if m.failNext > 0 {
		m.failNext--
		return false, nil
	}
	if string(m.data[id]) != string(old) {
		return false, nil
	}
	if p == nil {
		delete(m.data, id)
		return true, nil
	}
	raw, err := json.Marshal(p)
	m.data[id] = raw
	return true, err
}

func (m *memStore) PostIDs() ([]string, error) {
	var ids []string
	for id := range m.data {
		ids = append(ids, id)
	}
	return ids, nil
}

type dm struct{ to, message string }

type fakeMM struct {
	posts    map[string]*model.Post
	users    map[string]*model.User
	channels map[string]*model.Channel
	members  map[string]*model.ChannelMember // channelID/userID
	prefs    map[string]string
	dms      []dm
}

func newFakeMM() *fakeMM {
	f := &fakeMM{
		posts: map[string]*model.Post{
			"p1":     {Id: "p1", UserId: "alice", ChannelId: "town", Message: "Campaign brief is ready\nplease review"},
			"reply":  {Id: "reply", UserId: "alice", ChannelId: "town", RootId: "p1", Message: "Updated"},
			"dmpost": {Id: "dmpost", UserId: "alice", ChannelId: "dm", Message: "hi"},
			"files":  {Id: "files", UserId: "alice", ChannelId: "town", FileIds: []string{"f1"}},
			"sys":    {Id: "sys", UserId: "alice", ChannelId: "town", Type: model.PostTypeJoinChannel},
			"bots":   {Id: "bots", UserId: "bot", ChannelId: "town", Message: "deploy done"},
		},
		users: map[string]*model.User{
			"alice": {Id: "alice", Username: "alice"},
			"bob":   {Id: "bob", Username: "bob"},
			"cem":   {Id: "cem", Username: "cem"},
			"deniz": {Id: "deniz", Username: "deniz"},
			"ece":   {Id: "ece", Username: "ece"},
			"bot":   {Id: "bot", Username: "deploybot", IsBot: true},
		},
		channels: map[string]*model.Channel{
			"town": {Id: "town", Name: "town-square", Type: model.ChannelTypeOpen},
			"dm":   {Id: "dm", Name: "alice__bob", Type: model.ChannelTypeDirect},
		},
		members: map[string]*model.ChannelMember{},
		prefs:   map[string]string{},
	}
	for _, ch := range []string{"town", "dm"} {
		f.members[ch+"/alice"] = &model.ChannelMember{ChannelId: ch, UserId: "alice", NotifyProps: model.StringMap{model.MarkUnreadNotifyProp: model.ChannelMarkUnreadAll}}
	}
	return f
}

func (f *fakeMM) GetPost(id string) (*model.Post, error) {
	if p, ok := f.posts[id]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeMM) GetUser(id string) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeMM) GetChannel(id string) (*model.Channel, error) {
	if c, ok := f.channels[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeMM) GetChannelMember(channelID, userID string) (*model.ChannelMember, error) {
	if m, ok := f.members[channelID+"/"+userID]; ok {
		return m, nil
	}
	return nil, errors.New("not a member")
}

func (f *fakeMM) Preference(userID string) string { return f.prefs[userID] }

func (f *fakeMM) BotDM(userID, message string) error {
	f.dms = append(f.dms, dm{userID, message})
	return nil
}

func (f *fakeMM) SiteURL() string { return "https://team.example.com/" }

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

var t0 = time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)

func newTestService() (*Service, *memStore, *fakeMM, *fakeClock) {
	st, mm, clock := newMemStore(), newFakeMM(), &fakeClock{now: t0}
	return NewService(st, mm, clock, func(string, ...any) {}), st, mm, clock
}

func react(s *Service, user, post, emoji string) {
	s.OnAdded(&model.Reaction{UserId: user, PostId: post, EmojiName: emoji})
}

func flushLater(s *Service, clock *fakeClock) {
	clock.now = clock.now.Add(GroupFor)
	s.Flush()
}

// --- tests ---

func TestOneReactionSendsOneDMAfterTheGroupingWindow(t *testing.T) {
	s, st, mm, clock := newTestService()
	react(s, "bob", "p1", "+1")

	s.Flush()
	assert.Empty(t, mm.dms, "still grouping")

	flushLater(s, clock)
	require.Len(t, mm.dms, 1)
	assert.Equal(t, "alice", mm.dms[0].to)
	assert.Equal(t, ":+1: **@bob** reacted to your message in ~town-square\n"+
		"> Campaign brief is ready please review\n\n"+
		"[Open message](https://team.example.com/_redirect/pl/p1)", mm.dms[0].message)
	assert.Empty(t, st.data)

	s.Flush()
	assert.Len(t, mm.dms, 1, "sent once")
}

func TestReactionsWithinTheWindowAreGrouped(t *testing.T) {
	s, _, mm, clock := newTestService()
	react(s, "bob", "p1", "+1")
	react(s, "cem", "p1", "tada")
	react(s, "bob", "p1", "+1") // duplicate event
	react(s, "bob", "p1", "tada")
	flushLater(s, clock)
	require.Len(t, mm.dms, 1)
	assert.True(t, strings.HasPrefix(mm.dms[0].message, ":+1: :tada: **@bob** and **@cem** reacted to your message"), mm.dms[0].message)
}

func TestManyReactorsAreSummarised(t *testing.T) {
	s, _, mm, clock := newTestService()
	for _, u := range []string{"bob", "cem", "deniz", "ece"} {
		react(s, u, "p1", "heart")
	}
	flushLater(s, clock)
	assert.Contains(t, mm.dms[0].message, ":heart: **@bob**, **@cem** and 2 others reacted")

	react(s, "bob", "reply", "eyes")
	react(s, "cem", "reply", "eyes")
	react(s, "deniz", "reply", "eyes")
	flushLater(s, clock)
	assert.Contains(t, mm.dms[1].message, ":eyes: **@bob**, **@cem** and **@deniz** reacted to your reply in ~town-square")
}

func TestRemovedReactionIsNotSent(t *testing.T) {
	s, st, mm, clock := newTestService()
	react(s, "bob", "p1", "+1")
	react(s, "cem", "p1", "+1")
	s.OnRemoved(&model.Reaction{UserId: "bob", PostId: "p1", EmojiName: "+1"})
	flushLater(s, clock)
	require.Len(t, mm.dms, 1)
	assert.NotContains(t, mm.dms[0].message, "@bob")

	react(s, "bob", "p1", "+1")
	s.OnRemoved(&model.Reaction{UserId: "bob", PostId: "p1", EmojiName: "+1"})
	assert.Empty(t, st.data, "an empty group is dropped")
}

func TestReactionsThatNeverNotify(t *testing.T) {
	cases := map[string]struct{ user, post string }{
		"your own reaction":  {"alice", "p1"},
		"a bot reacting":     {"bot", "p1"},
		"a bot's post":       {"bob", "bots"},
		"a system message":   {"bob", "sys"},
		"an unknown post":    {"bob", "nope"},
		"an unknown reactor": {"nobody", "p1"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s, st, mm, clock := newTestService()
			react(s, tc.user, tc.post, "+1")
			flushLater(s, clock)
			assert.Empty(t, st.data)
			assert.Empty(t, mm.dms)
		})
	}
}

func TestDeactivatedAuthorAndDeletedPostAreSkipped(t *testing.T) {
	s, _, mm, clock := newTestService()
	react(s, "bob", "p1", "+1")
	mm.posts["p1"].DeleteAt = 1
	flushLater(s, clock)
	assert.Empty(t, mm.dms)

	mm.posts["p1"].DeleteAt = 0
	mm.users["alice"].DeleteAt = 1
	react(s, "bob", "p1", "+1")
	flushLater(s, clock)
	assert.Empty(t, mm.dms)
}

func TestSettingsAreRespected(t *testing.T) {
	s, st, mm, clock := newTestService()
	mm.prefs["alice"] = NotifyOff
	react(s, "bob", "p1", "+1")
	assert.Empty(t, st.data, "nothing is collected when off")

	mm.prefs["alice"] = NotifyDMs
	react(s, "bob", "p1", "+1")
	react(s, "bob", "dmpost", "+1")
	flushLater(s, clock)
	require.Len(t, mm.dms, 1)
	assert.Contains(t, mm.dms[0].message, "reacted to your message in a direct message")

	// Turned off while a group was collecting.
	react(s, "bob", "dmpost", "smile")
	mm.prefs["alice"] = NotifyOff
	flushLater(s, clock)
	assert.Len(t, mm.dms, 1)
}

func TestMutedOrLeftChannelIsSkipped(t *testing.T) {
	s, _, mm, clock := newTestService()
	mm.members["town/alice"].NotifyProps[model.MarkUnreadNotifyProp] = model.ChannelMarkUnreadMention
	react(s, "bob", "p1", "+1")
	flushLater(s, clock)
	assert.Empty(t, mm.dms, "muted")

	delete(mm.members, "town/alice")
	react(s, "bob", "p1", "+1")
	flushLater(s, clock)
	assert.Empty(t, mm.dms, "left the channel")
}

func TestAttachmentOnlyPostAndLongTextAreQuotedSensibly(t *testing.T) {
	s, _, mm, clock := newTestService()
	react(s, "bob", "files", "+1")
	mm.posts["p1"].Message = strings.Repeat("é", 250)
	react(s, "bob", "p1", "+1")
	flushLater(s, clock)
	require.Len(t, mm.dms, 2)
	all := mm.dms[0].message + mm.dms[1].message
	assert.Contains(t, all, "> _(attachment)_")
	assert.Contains(t, all, "> "+strings.Repeat("é", 200)+"…")
}

func TestConcurrentWritesAreRetried(t *testing.T) {
	s, st, mm, clock := newTestService()
	st.failNext = 2 // two other reactions won the race first
	react(s, "bob", "p1", "+1")
	flushLater(s, clock)
	assert.Len(t, mm.dms, 1)
}

func TestLostClaimMeansAnotherServerSends(t *testing.T) {
	s, st, mm, clock := newTestService()
	react(s, "bob", "p1", "+1")
	clock.now = clock.now.Add(GroupFor)
	st.failNext = 1
	s.Flush()
	assert.Empty(t, mm.dms)
	assert.Len(t, st.data, 1, "left for the server that claimed it")
}
