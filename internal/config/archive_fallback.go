// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import "log/slog"

// warnDeprecatedArchive 输出「archive.* 已废弃」的弃用告警。
// 消费者命中旧键回退时调用一次。
func warnDeprecatedArchive(key string) {
	slog.Warn("archive.* 已废弃，请迁移到 cocom.archive.*",
		slog.String("deprecated_key", "archive."+key),
		slog.String("canonical_key", "cocom.archive."+key),
		slog.String("deprecated", "v0.0.59 将移除 archive.* 回退"))
}

// ArchiveString 返回归档字符串配置：优先 cocom.archive.<key>，
// 命中旧键 archive.<key> 时回退并输出弃用告警。
func ArchiveString(newVal, oldVal, key string) string {
	if newVal != "" {
		return newVal
	}
	if oldVal != "" {
		warnDeprecatedArchive(key)
		return oldVal
	}
	return ""
}

// ArchiveBool 返回归档布尔配置：优先 cocom.archive.<key>，
// 命中旧键 archive.<key>（true）时回退并输出弃用告警。
func ArchiveBool(newVal, oldVal bool, key string) bool {
	if newVal {
		return true
	}
	if oldVal {
		warnDeprecatedArchive(key)
		return true
	}
	return false
}

// ArchiveInt 返回归档整数配置：优先 cocom.archive.<key>，
// 命中旧键 archive.<key> 且新键为零值时回退并输出弃用告警。
func ArchiveInt(newVal, oldVal int, key string) int {
	if newVal > 0 {
		return newVal
	}
	if oldVal > 0 {
		warnDeprecatedArchive(key)
		return oldVal
	}
	return newVal
}
