// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/cocomhub/cocom/pkg/comic"
)

func TestStorage_FindChannelHelper(t *testing.T) {
	ctx := context.Background()
	filter := &comic.ComicFilter{}
	filter.SetLimit(10)

	advance := func(impls []comic.Comic, f *comic.ComicFilter) {
		f.Skip += int64(len(impls))
	}

	ch, err := FindChannelHelper(ctx, filter, func(ctx context.Context, f *comic.ComicFilter) ([]comic.Comic, error) {
		return []comic.Comic{}, nil
	}, advance)
	if err != nil {
		t.Fatalf("FindChannelHelper failed: %v", err)
	}
	if ch == nil {
		t.Error("FindChannelHelper should return a channel")
	}
	// Channel should close immediately (no results)
	for range ch {
	}
	t.Log("FindChannelHelper with empty results completed")
}

// TestStorage_FindChannelHelper_Pagination 分页算术回归（S10）：
// 循环边界按 work.Skip < oriLimit 判断（而非 clamp 后 Limit+Skip<=oriLimit），
// 否则 Limit>100 时提前停取、丢失尾页。表驱动三种场景（锁步取整）：
//   - L=250/S0 / 数据 250：应拉满 250（此前 bug 只取 200）；
//   - L=50/S30 / 数据 200：首页应拉取（此前条件恒 false 一页都不拉）；
//     work.Limit=min(100, oriLimit)=80，单页即达 oriLimit 停循环 → 交付 80；
//   - L=100/S30 / 数据 200：首页拉 100，Skip=130 达 oriLimit 停 → 交付 100，全量。
//
// 注意：分页按 Skip 推进时交付量以 oriLimit（Limit+Skip）为上界为准，不另做截断——
// 这是当前 FindChannelHelper 锁步行为，keep min(100, oriLimit) 的既有语义。
func TestStorage_FindChannelHelper_Pagination(t *testing.T) {
	mksource := func(n int) (func(ctx context.Context, f *comic.ComicFilter) ([]comic.Comic, error), func() (int, int)) {
		var total, pages int
		findFn := func(ctx context.Context, f *comic.ComicFilter) ([]comic.Comic, error) {
			var out []comic.Comic
			for i := f.Skip; i < int64(n); i++ {
				out = append(out, comic.NewComic(fmt.Sprintf("%d", i+1), "", nil))
				if f.Limit > 0 && int64(i) >= f.Skip+f.Limit-1 {
					break
				}
			}
			if len(out) == 0 {
				return out, nil
			}
			total += len(out)
			pages++
			return out, nil
		}
		return findFn, func() (int, int) { return total, pages }
	}

	tests := []struct {
		name        string
		limit, skip int64
		n           int // 数据条数
		wantLen     int // 期望产出条数
		wantFetched int // 期望 findFn 被调用拉到的总条数
	}{
		{name: "L250/S0 拉满 250（不丢尾页）", limit: 250, skip: 0, n: 250, wantLen: 250, wantFetched: 250},
		{name: "L50/S30 首页应拉取", limit: 50, skip: 30, n: 200, wantLen: 80, wantFetched: 80},
		{name: "L100/S30 全量", limit: 100, skip: 30, n: 200, wantLen: 100, wantFetched: 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findFn, stats := mksource(tc.n)
			f := &comic.ComicFilter{Limit: tc.limit, Skip: tc.skip}
			ch, err := FindChannelHelper(context.Background(), f, findFn, nil)
			if err != nil {
				t.Fatalf("FindChannelHelper failed: %v", err)
			}
			var got []comic.Comic
			for c := range ch {
				got = append(got, c)
			}
			if len(got) != tc.wantLen {
				t.Errorf("collected = %d, want %d", len(got), tc.wantLen)
			}
			fetched, pages := stats()
			if fetched != tc.wantFetched {
				t.Errorf("fetched = %d, want %d (丢失尾页回归)", fetched, tc.wantFetched)
			}
			if pages == 0 {
				t.Error("at least one page expected")
			}
		})
	}
}

// TestStorage_FindChannelHelper_Keyset 验证 keyset 推进（advanceFn 重置 Skip 为 0）
// 时用累计已交付 delivered 与预算判定，不会无限循环；且回归只改动循环边界、
// 不改动交付结果——默认 Skip 推进下从不会因为 delivered 判定提前终止正循环。
func TestStorage_FindChannelHelper_Keyset(t *testing.T) {
	// keyset 源：数据 n 条，advanceFn 把 LastID 转为 IDRangeLeft 并重置 Skip=0。
	mksource := func(n int) (func(ctx context.Context, f *comic.ComicFilter) ([]comic.Comic, error), func() (int, int)) {
		var total, pages int
		findFn := func(ctx context.Context, f *comic.ComicFilter) ([]comic.Comic, error) {
			var out []comic.Comic
			if f.IDRangeLeft == nil {
				f = &comic.ComicFilter{Limit: f.Limit, Skip: 0, IDRangeLeft: new(int64(1))}
			}
			start := *f.IDRangeLeft - 1
			for i := start; i < int64(n); i++ {
				out = append(out, comic.NewComic(fmt.Sprintf("%d", i+1), "", nil))
				if f.Limit > 0 && int64(i) >= start+f.Limit-1 {
					break
				}
			}
			if len(out) == 0 {
				return out, nil
			}
			total += len(out)
			pages++
			return out, nil
		}
		return findFn, func() (int, int) { return total, pages }
	}

	advance := func(impls []comic.Comic, f *comic.ComicFilter) {
		last, _ := strconv.Atoi(impls[len(impls)-1].GetID())
		f.IDRangeLeft = new(int64(last + 1))
		f.Skip = 0
	}

	tests := []struct {
		name        string
		limit, skip int64
		n           int
		wantMin     int // 至少交付量（预算 oriLimit），keyset 命中真预算即停
		wantMax     int // 至多交付量（预算 + 一个页大小，批次粒度内可多拉一页）
	}{
		// L=250/S=0 keyset 全量：预算 250，页 100 → 交付 300（≥250 即停，整页粒度过盈）。
		{name: "keyset L250/S0 全量", limit: 250, skip: 0, n: 100000, wantMin: 250, wantMax: 350},
		// L=100/S30 keyset bounded：预算 130，页 100 → 交付 200（同样整页过盈，但受限）。
		{name: "keyset L100/S30 bounded", limit: 100, skip: 30, n: 100000, wantMin: 130, wantMax: 230},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findFn, stats := mksource(tc.n)
			f := &comic.ComicFilter{Limit: tc.limit, Skip: tc.skip}
			ch, err := FindChannelHelper(context.Background(), f, findFn, advance)
			if err != nil {
				t.Fatalf("FindChannelHelper failed: %v", err)
			}
			var got []comic.Comic
			for c := range ch {
				got = append(got, c)
			}
			if len(got) < tc.wantMin || len(got) > tc.wantMax {
				t.Errorf("collected = %d, want in [%d, %d]（分页上界有界）", len(got), tc.wantMin, tc.wantMax)
			}
			fetched, pages := stats()
			if pages == 0 {
				t.Error("at least one page expected")
			}
			if fetched != len(got) {
				t.Errorf("fetched = %d, collect = %d 不一致", fetched, len(got))
			}
		})
	}
}
