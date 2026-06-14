// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
)

// InitE2EStorage 初始化 E2E 测试需要的内存存储并注入到各包默认存储中。
// E2E 测试（独立 module）无法直接导入 internal 包，所以通过 handler 包间接完成初始化。
func InitE2EStorage() *comicpkg.MemoryStorage {
	store := comicpkg.NewMemoryStorage()
	comic.SetDefaultStorage(store)
	tag.SetDefaultLikeStore(tag.NewMemoryLikeStore())
	tag.SetDefaultComicStore(store)
	tag.SetDefaultRelationStore(tag.NewMemoryRelationStore())
	return store
}
