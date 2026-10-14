// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

package reactions

import (
	"encoding/json"

	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

const keyPrefix = "pend_"

// KVStore keeps each post's pending reactions under its own key in the plugin's KV store.
// Values are raw JSON bytes so CompareAndSet compares exactly what was read.
type KVStore struct {
	kv *pluginapi.KVService
}

func NewKVStore(kv *pluginapi.KVService) *KVStore { return &KVStore{kv: kv} }

func (s *KVStore) Get(postID string) (*Pending, []byte, error) {
	var raw []byte
	if err := s.kv.Get(keyPrefix+postID, &raw); err != nil {
		return nil, nil, errors.Wrap(err, "failed to read pending reactions")
	}
	if len(raw) == 0 {
		return nil, nil, nil
	}
	var p Pending
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode pending reactions")
	}
	return &p, raw, nil
}

func (s *KVStore) CompareAndSet(postID string, old []byte, p *Pending) (bool, error) {
	var value any // nil deletes
	if p != nil {
		raw, err := json.Marshal(p)
		if err != nil {
			return false, err
		}
		value = raw
	}
	ok, err := s.kv.Set(keyPrefix+postID, value, pluginapi.SetAtomic(old))
	return ok, errors.Wrap(err, "failed to update pending reactions")
}

func (s *KVStore) PostIDs() ([]string, error) {
	const perPage = 200
	var ids []string
	for page := 0; ; page++ {
		keys, err := s.kv.ListKeys(page, perPage, pluginapi.WithPrefix(keyPrefix))
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pending reactions")
		}
		for _, key := range keys {
			ids = append(ids, key[len(keyPrefix):])
		}
		if len(keys) < perPage {
			return ids, nil
		}
	}
}
