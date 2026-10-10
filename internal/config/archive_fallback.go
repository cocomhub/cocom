// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"log/slog"
	"strings"
	"sync"
)

// warnedKeys 记录已输出过弃用/冲突告警的键，避免每请求刷屏。
var warnedKeys sync.Map // map[string]struct{}  —— 由 warnOnce 写入

// warnOnce 对同一个 key 只输出一次告警（多协程并发安全）。
// 首调用后同 key 的后续调用直接沉默，避免每请求刷屏。
func warnOnce(key string, msg string, attrs ...any) {
	if _, loaded := warnedKeys.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	slog.Warn(msg, attrs...)
}

// warnDeprecatedArchive 输出「archive.* 已废弃」的弃用告警（每 key 一次）。
func warnDeprecatedArchive(key string) {
	warnOnce("deprecated:"+key,
		"archive.* 已废弃，请迁移到 cocom.archive.*",
		slog.String("deprecated_key", "archive."+key),
		slog.String("canonical_key", "cocom.archive."+key),
		slog.String("deprecated", "v0.0.59 将移除 archive.* 回退"))
}

// warnArchiveConflict 输出新旧归档键冲突告警（每 key 一次）。
// 两者都显式配置且值不同时调用；行为仍回退到新键，仅要求尽快运行 config migrate 统一配置。
func warnArchiveConflict(key string) {
	warnOnce("conflict:"+key,
		"archive.* 与 cocom.archive.* 同时配置且值不同，已按新键生效——请运行 cocom config migrate 统一配置",
		slog.String("deprecated_key", "archive."+key),
		slog.String("canonical_key", "cocom.archive."+key))
}

// isZeroOrMissingString 判断字符串键是否未显式配置。
// viper.IsSet 对注册了 SetDefault 的键恒返回 true，无法区分「默认值」与「显式配置」，
// 因此字符串键的「未配置」判定用空串近似：默认空串 + 显式空串等价于未配置，
// 语义上无冲突风险（空 vs 空）。
func isZeroString(s string) bool { return strings.TrimSpace(s) == "" }

// ArchiveString 返回归档字符串配置：优先 cocom.archive.<key>。
// 两者都显式配置且值不同时输出冲突告警（仍按新键生效）；仅旧键非空时回退并告警。
func ArchiveString(newVal, oldVal, key string) string {
	if !isZeroString(newVal) {
		if !isZeroString(oldVal) && newVal != oldVal {
			warnArchiveConflict(key)
		}
		return newVal
	}
	if !isZeroString(oldVal) {
		warnDeprecatedArchive(key)
		return oldVal
	}
	return ""
}

// ArchiveBool 返回归档布尔配置：优先 cocom.archive.<key>。
// 两者都 true 时告警冲突（按新键生效）；仅旧键 true 时回退并告警。
func ArchiveBool(newVal, oldVal bool, key string) bool {
	if newVal {
		if oldVal {
			warnArchiveConflict(key)
		}
		return true
	}
	if oldVal {
		warnDeprecatedArchive(key)
		return true
	}
	return false
}

// ArchiveInt 返回归档整数配置：优先 cocom.archive.<key>（新键 >0）。
// 两者都是正数且不相等时告警冲突（按新键生效）；仅旧键为正时回退并告警。
func ArchiveInt(newVal, oldVal int, key string) int {
	if newVal > 0 {
		if oldVal > 0 && newVal != oldVal {
			warnArchiveConflict(key)
		}
		return newVal
	}
	if oldVal > 0 {
		warnDeprecatedArchive(key)
		return oldVal
	}
	return 0
}
