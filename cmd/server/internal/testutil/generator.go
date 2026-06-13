// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"fmt"

	"github.com/cocomhub/cocom/cmd/server/api"
)

// GenerateComics 批量生成漫画用于基准测试
func GenerateComics(n int, opts ...func(*api.ComicInfo)) []*api.ComicInfo {
	comics := make([]*api.ComicInfo, n)
	for i := range n {
		cid := 10000 + i
		info := MockComicInfo(cid, opts...)
		// 设置唯一标题以免搜索结果互相干扰
		info.Title.English = fmt.Sprintf("Test Comic %d", cid)
		info.Title.Pretty = fmt.Sprintf("Test Comic %d", cid)
		comics[i] = info
	}
	return comics
}

// GenerateTags 批量生成标签
func GenerateTags(baseID int, typ string, n int) []api.Tag {
	tags := make([]api.Tag, n)
	for i := range n {
		tags[i] = MockTag(baseID+i, typ, fmt.Sprintf("tag-%d", baseID+i))
	}
	return tags
}
