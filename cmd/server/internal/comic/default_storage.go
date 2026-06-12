// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import "github.com/cocomhub/cocom/pkg/comic"

var defaultStorage comic.Storage

// SetDefaultStorage 设置包级默认存储，用于测试注入 MemoryStorage
func SetDefaultStorage(s comic.Storage) { defaultStorage = s }

// GetDefaultStorage 返回包级默认存储
func GetDefaultStorage() comic.Storage { return defaultStorage }

// ResetDefaultStorage 重置包级默认存储
func ResetDefaultStorage() { defaultStorage = nil }
