// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package setting

import (
	"context"
	"maps"
	"sync"
)

// MemorySettingsStore 是 SettingsStore 接口的内存实现。
type MemorySettingsStore struct {
	mu     sync.Mutex
	stores map[string]map[string]any
}

// NewMemorySettingsStore 创建一个新的 MemorySettingsStore。
func NewMemorySettingsStore() *MemorySettingsStore {
	return &MemorySettingsStore{
		stores: make(map[string]map[string]any),
	}
}

// Get 实现了 SettingsStore.Get。
func (s *MemorySettingsStore) Get(ctx context.Context, settingType string, keys ...string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, ok := s.stores[settingType]
	if !ok {
		return map[string]any{}, nil
	}
	// if keys is empty or first is empty string, return all
	if len(keys) == 0 || keys[0] == "" {
		out := map[string]any{}
		maps.Copy(out, store)
		return out, nil
	}
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := store[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

// Set 实现了 SettingsStore.Set。
func (s *MemorySettingsStore) Set(ctx context.Context, settingType string, kvs map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, ok := s.stores[settingType]
	if !ok {
		store = map[string]any{}
		s.stores[settingType] = store
	}
	maps.Copy(store, kvs)
	return nil
}

// Del 实现了 SettingsStore.Del。
func (s *MemorySettingsStore) Del(ctx context.Context, settingType string, keys ...string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, ok := s.stores[settingType]
	if !ok {
		return 0, nil
	}
	var deleted int64
	// if keys empty or first empty string, delete all under type
	if len(keys) == 0 || keys[0] == "" {
		deleted = int64(len(store))
		delete(s.stores, settingType)
		return deleted, nil
	}
	for _, k := range keys {
		if _, ok := store[k]; ok {
			delete(store, k)
			deleted++
		}
	}
	if len(store) == 0 {
		delete(s.stores, settingType)
	}
	return deleted, nil
}
