// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"log/slog"

	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/testutil"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
)

// SeedTestData 向 MemoryStorage 填充种子测试数据，供 handler 测试和 chromedp 测试复用
func SeedTestData(ctx context.Context, store *comicpkg.MemoryStorage) {
	// 使用默认首页场景作为种子数据
	scenario := testutil.HomePageScenario()

	for _, info := range scenario.Comics {
		if err := store.Save(ctx, comic.NewComic(info)); err != nil {
			slog.ErrorContext(
				ctx, "seed test data failed",
				slog.String("errmsg", err.Error()),
				slog.Int("cid", info.CID),
			)
		}
	}
}
