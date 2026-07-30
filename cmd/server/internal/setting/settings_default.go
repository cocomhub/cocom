// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package setting

import (
	"context"
	"sync"
)

const (
	SettingKeyType string = "type"
	SettingKeyKey  string = "key"
	SettingKeyVal  string = "val"
)

// SettingsStore 是一个可替换的 settings 存储接口，用于测试注入。
// 实现可以是 MemorySettingsStore（测试用）或 MongoDB 存储（生产用）。
type SettingsStore interface {
	Get(ctx context.Context, settingType string, keys ...string) (map[string]any, error)
	Set(ctx context.Context, settingType string, kvs map[string]any) error
	Del(ctx context.Context, settingType string, keys ...string) (int64, error)
}

var (
	defaultSettingsStoreMu sync.RWMutex
	defaultSettingsStore   SettingsStore
)

// SetDefaultSettingsStore 设置默认的 SettingsStore 实现。
// 设置后将优先于 MongoDB 路径使用。
func SetDefaultSettingsStore(s SettingsStore) {
	defaultSettingsStoreMu.Lock()
	defaultSettingsStore = s
	defaultSettingsStoreMu.Unlock()
}

// GetDefaultSettingsStore 返回当前注入的 SettingsStore。
func GetDefaultSettingsStore() SettingsStore {
	defaultSettingsStoreMu.RLock()
	defer defaultSettingsStoreMu.RUnlock()
	return defaultSettingsStore
}

// ResetDefaultSettingsStore 重置默认的 SettingsStore。
func ResetDefaultSettingsStore() {
	defaultSettingsStoreMu.Lock()
	defaultSettingsStore = nil
	defaultSettingsStoreMu.Unlock()
}
