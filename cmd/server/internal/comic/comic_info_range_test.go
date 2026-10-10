// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/comic"
)

// TestGetRangeComicInfos_MemoryOrderAndPagingMemory 回归：内存分支必须与 Mongo 分支
// （SortKV("cid", -1) 后分页）同页同序——即「按 cid 降序后再套 skip/limit」。
// 此前实现先套 Skip/Limit 再把页内结果反转，limit=2/skip=0 会返回最旧的 2 条。
func TestGetRangeComicInfos_MemoryOrderAndPagingMemory(t *testing.T) {
	ctx := context.Background()
	ms := comic.NewMemoryStorage()
	const total = 10
	for cid := 1; cid <= total; cid++ {
		info := &api.ComicInfo{CID: cid}
		if err := ms.Save(ctx, NewComic(info)); err != nil {
			t.Fatalf("seed cid %d: %v", cid, err)
		}
	}
	SetDefaultStorage(ms)
	t.Cleanup(func() { SetDefaultStorage(nil) })

	cases := []struct {
		name     string
		limit    int64
		skip     int64
		wantCIDs []int
	}{
		{name: "首页 limit=2 skip=0 应返回最新两条", limit: 2, skip: 0, wantCIDs: []int{10, 9}},
		{name: "第二页 limit=2 skip=2", limit: 2, skip: 2, wantCIDs: []int{8, 7}},
		{name: "末页不足一页", limit: 3, skip: 9, wantCIDs: []int{1}},
		{name: "skip 超出范围返回空", limit: 5, skip: 100, wantCIDs: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			infos, err := GetRangeComicInfos(ctx, tc.limit, tc.skip)
			if err != nil {
				t.Fatalf("GetRangeComicInfos: %v", err)
			}
			got := make([]int, 0, len(infos))
			for _, in := range infos {
				got = append(got, in.CID)
			}
			if len(got) != len(tc.wantCIDs) {
				t.Fatalf("cids = %v, want %v", got, tc.wantCIDs)
			}
			for i := range got {
				if got[i] != tc.wantCIDs[i] {
					t.Fatalf("cids = %v, want %v (顺序/页码不一致)", got, tc.wantCIDs)
				}
			}
			if len(got) > 1 {
				for i := 1; i < len(got); i++ {
					if got[i-1] <= got[i] {
						t.Fatalf("cids = %v, 必须按 cid 降序", got)
					}
				}
			}
		})
	}
}

func TestApplyPageWindow(t *testing.T) {
	mk := func(n int) []comic.Comic {
		out := make([]comic.Comic, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, NewComic(&api.ComicInfo{CID: i + 1}))
		}
		return out
	}

	cases := []struct {
		name        string
		n           int
		limit, skip int64
		wantLen     int
	}{
		{"limit<=0 不限制", 5, 0, 0, 5},
		{"skip<0 归一为 0", 5, 2, -3, 2},
		{"skip 超界", 5, 10, 9, 0},
		{"正常窗口", 5, 2, 1, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := applyPageWindow(mk(tc.n), tc.limit, tc.skip)
			if len(got) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

// TestComicCID 保证非法 id 不 panic 且降序排在最后。
func TestComicCID(t *testing.T) {
	if got := comicCID(NewComic(&api.ComicInfo{CID: 123})); got != 123 {
		t.Errorf("comicCID = %d, want 123", got)
	}
	if got := comicCID(NewComic(&api.ComicInfo{CID: 0})); got != 0 {
		t.Errorf("comicCID(0) = %d, want 0", got)
	}
}

// TestSaveVerifyResult_WithMemoryStorage 回归：SaveVerifyResult 只传 {"verify": ...}
// 的部分更新 map；内存分支必须能定位记录（此前因 map 缺 cid 被 MemoryStorage.Update
// 判为 "invalid comic id in update map"）。
func TestSaveVerifyResult_WithMemoryStorage(t *testing.T) {
	ctx := context.Background()
	ms := comic.NewMemoryStorage()
	const cid = 4242
	if err := ms.Save(ctx, NewComic(&api.ComicInfo{CID: cid})); err != nil {
		t.Fatalf("seed comic: %v", err)
	}
	SetDefaultStorage(ms)
	t.Cleanup(func() { SetDefaultStorage(nil) })

	st := NewStorage()
	result := &comic.VerifyResult{ID: "r1", ComicID: strconv.Itoa(cid), Valid: true, FixedCount: 2}
	if err := st.SaveVerifyResult(ctx, result); err != nil {
		t.Fatalf("SaveVerifyResult: %v", err)
	}

	c, err := ms.Get(ctx, strconv.Itoa(cid))
	if err != nil {
		t.Fatalf("get comic: %v", err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal comic: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal comic: %v", err)
	}
	if _, ok := got["verify"]; !ok {
		t.Errorf("verify 字段未落库: %v", got)
	}
}
