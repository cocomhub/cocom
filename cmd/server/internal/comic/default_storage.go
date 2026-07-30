// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"sync"

	"github.com/cocomhub/cocom/pkg/comic"
)

var (
	defaultStorageMu sync.RWMutex
	defaultStorage   comic.Storage
)

// SetDefaultStorage 设置包级默认存储，用于测试注入 MemoryStorage
func SetDefaultStorage(s comic.Storage) {
	defaultStorageMu.Lock()
	defaultStorage = s
	defaultStorageMu.Unlock()
}

// GetDefaultStorage 返回包级默认存储
func GetDefaultStorage() comic.Storage {
	defaultStorageMu.RLock()
	defer defaultStorageMu.RUnlock()
	return defaultStorage
}

// ResetDefaultStorage 重置包级默认存储
func ResetDefaultStorage() {
	defaultStorageMu.Lock()
	defaultStorage = nil
	defaultStorageMu.Unlock()
}
