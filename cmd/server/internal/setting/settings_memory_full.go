// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build memory_storage_integration

package setting

import "context"

// memorySettingsStore 包级单例，保证 tag 构建下 Set/Get/Del 读写同一实例。
// 与 settings.go 一致：优先委托 SetDefaultSettingsStore 注入的 store，否则回退单例。
var memorySettingsStore = NewMemorySettingsStore()

// GetSettings 从内存存储中获取设置。
func GetSettings(ctx context.Context, settingType string, keys ...string) (map[string]any, error) {
	if s := GetDefaultSettingsStore(); s != nil {
		return s.Get(ctx, settingType, keys...)
	}
	return memorySettingsStore.Get(ctx, settingType, keys...)
}

// SetSettings 将设置写入内存存储。
// 空 kvs 视为 no-op，与 Mongo 路径（BulkWrite 空 models 报 ErrEmptySlice）行为对齐。
func SetSettings(ctx context.Context, settingType string, kvs map[string]any) error {
	if len(kvs) == 0 {
		return nil
	}
	if s := GetDefaultSettingsStore(); s != nil {
		return s.Set(ctx, settingType, kvs)
	}
	return memorySettingsStore.Set(ctx, settingType, kvs)
}

// DelSettings 从内存存储中删除设置。
// 空 keys 返回参数错误，避免"无 keys 误删整个 type"的数据丢失 footgun。
func DelSettings(ctx context.Context, settingType string, keys ...string) (int64, error) {
	if len(keys) == 0 || keys[0] == "" {
		return 0, errSettingsKeysRequired
	}
	if s := GetDefaultSettingsStore(); s != nil {
		return s.Del(ctx, settingType, keys...)
	}
	return memorySettingsStore.Del(ctx, settingType, keys...)
}
