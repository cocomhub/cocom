// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build memory_storage_integration

package setting

import "context"

// GetSettings 从内存存储中获取设置。
func GetSettings(ctx context.Context, settingType string, keys ...string) (map[string]any, error) {
	return NewMemorySettingsStore().Get(ctx, settingType, keys...)
}

// SetSettings 将设置写入内存存储。
func SetSettings(ctx context.Context, settingType string, kvs map[string]any) error {
	return NewMemorySettingsStore().Set(ctx, settingType, kvs)
}

// DelSettings 从内存存储中删除设置。
func DelSettings(ctx context.Context, settingType string, keys ...string) (int64, error) {
	return NewMemorySettingsStore().Del(ctx, settingType, keys...)
}
