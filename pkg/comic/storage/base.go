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
//
// 设计决策（probe-retry-policy 记忆）：拉取数据失败不在本函数无上限重试——
// findFn 出错立即返回并关闭通道，调用方各自决定重试策略（pkg/comic/probe/probe.go
// 的拉取为无上限重试、上传为有上限；见该处设计决策注释）。
func FindChannelHelper(
	ctx context.Context,
	filter *comic.ComicFilter,
	findFn func(ctx context.Context, filter *comic.ComicFilter) ([]comic.Comic, error),
	advanceFn func(impls []comic.Comic, filter *comic.ComicFilter),
) (chan comic.Comic, error) {
	comics := make(chan comic.Comic, 100)
	go func() {
		defer close(comics)
		// 浅拷贝，避免修改调用方传入的 filter（副作用泄漏到调用方）
		work := *filter
		oriLimit := work.Limit + work.Skip
		// work.Limit 作为单页大小上限（min(100, oriLimit)）。
		// 设计决策：循环边界必须用 work.Skip < oriLimit，不能用 clamp 后的 Limit 参与，
		// 否则在 Limit>100 时提前停取、丢失尾页（此前 bug：L=250/S=0 只取 200）。
		work.Limit = min(100, oriLimit)
		// delivered 累计本次已交付数量。默认 Skip 推进时 delivered == work.Skip - 初始 Skip；
		// keyset 推进（advanceFn 重置 work.Skip 为 0）时 (work.Skip<oriLimit) 恒真，
		// 必须用 delivered 与预算或iLimit（Limit+Skip，与 Skip 推进的交付上界一致）判断
		// 是否已满，防止死循环；oriLimit==0（NoLimit 全量 keyset）不做限制，
		// 靠数据取尽返回空 break 收束。默认 Skip 推进时此判定不会提前终止正循环。
		var delivered int64
		for work.Skip < oriLimit {
			if advanceFn != nil && oriLimit > 0 && delivered >= oriLimit {
				break
			}
			impls, err := findFn(ctx, &work)
			if err != nil {
				slog.ErrorContext(ctx, "failed to find comics", slog.String("err", err.Error()))
				return
			}
			if len(impls) == 0 {
				break
			}
			for _, c := range impls {
				select {
				case comics <- c:
				case <-ctx.Done():
					// 调用方已停止消费（如验证任务取消），不再阻塞推送，
					// defer close(comics) 会关闭通道让消费方 range 正常结束。
					return
				}
			}
			delivered += int64(len(impls))
			if advanceFn != nil {
				advanceFn(impls, &work)
			} else {
				work.Skip += int64(len(impls))
			}
		}
	}()
	return comics, nil
}
