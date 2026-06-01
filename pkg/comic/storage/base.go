// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"log/slog"

	"github.com/cocomhub/cocom/pkg/comic"
)

// FindChannelHelper 提供 FindChannel 的通用分页循环实现，供各个 MongoDB Storage 复用。
//
// findFn 是实际的查询逻辑（由调用方提供，注入不同的 collection + filter）。
// advanceFn 是可选的分页推进函数；为 nil 时使用默认的 Skip 推进。
//
// 示例 — 用默认 Skip 推进：
//
//	func (s *Storage) FindChannel(ctx, filter) (chan comic.Comic, error) {
//	    return FindChannelHelper(ctx, filter, s.Find, nil)
//	}
//
// 示例 — 用 NotArchived 专用推进（comic storage）：
//
//	func (s *Storage) FindChannel(ctx, filter) (chan comic.Comic, error) {
//	    advance := func(impls []comic.Comic, f *comic.ComicFilter) {
//	        cid, _ := strconv.Atoi(impls[len(impls)-1].GetID())
//	        f.IDRangeLeft = new(int64(cid + 1))
//	        f.Skip = 0
//	    }
//	    return FindChannelHelper(ctx, filter, s.Find, advance)
//	}
func FindChannelHelper(
	ctx context.Context,
	filter *comic.ComicFilter,
	findFn func(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error),
	advanceFn func(impls []comic.Comic, filter *comic.ComicFilter),
) (chan comic.Comic, error) {
	comics := make(chan comic.Comic, 100)
	go func() {
		defer close(comics)
		oriLimit := filter.Limit + filter.Skip
		filter.Limit = min(100, oriLimit)
		for filter.Limit+filter.Skip <= oriLimit {
			impls, err := findFn(ctx, filter)
			if err != nil {
				slog.ErrorContext(ctx, "failed to find comics", slog.String("err", err.Error()))
				return
			}
			if len(impls) == 0 {
				break
			}
			for _, c := range impls {
				comics <- c
			}
			if advanceFn != nil {
				advanceFn(impls, filter)
			} else {
				filter.Skip += int64(len(impls))
			}
		}
	}()
	return comics, nil
}
