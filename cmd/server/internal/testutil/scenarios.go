// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"github.com/cocomhub/cocom/cmd/server/api"
)

// Scenario 代表一个预定义的测试场景
type Scenario struct {
	Name   string
	Comics []*api.ComicInfo
}

// HomePageScenario 首页场景：热门漫画 + 新上传 + 各种标签
func HomePageScenario() *Scenario {
	return &Scenario{
		Name: "home-page",
		Comics: []*api.ComicInfo{
			MockComicInfo(1001, WithStatus(true), WithTagsV2(
				MockTag(1, "tag", "action"),
				MockTag(2, "tag", "adventure"),
			)),
			MockComicInfo(1002, WithStatus(true), WithTagsV2(
				MockTag(1, "tag", "action"),
			)),
			MockComicInfo(1003, WithStatus(true), WithTagsV2(
				MockTag(3, "tag", "comedy"),
			)),
			MockComicInfo(1004, WithStatus(true), WithTagsV2(
				MockTag(4, "parody", "naruto"),
			)),
			MockComicInfo(1005, WithStatus(true), WithTagsV2(
				MockTag(5, "tag", "romance"),
			), WithArchived("archive/1005.zip")),
			MockComicInfo(1006, WithStatus(false)),  // 未启用
			MockComicInfo(1007, WithDeleted()),      // 已删除
			MockComicInfo(1008, WithRedirect(1001)), // 重定向
		},
	}
}

// SearchScenario 搜索结果场景：不同标题匹配的漫画
func SearchScenario() *Scenario {
	return &Scenario{
		Name: "search",
		Comics: []*api.ComicInfo{
			MockComicInfo(
				2001, WithStatus(true),
				WithTagsV2(MockTag(10, "tag", "naruto")),
				WithTitle("Naruto Shippuden", "Naruto Shippuden", "NARUTO -ナルト- 疾風伝"),
			),
			MockComicInfo(
				2002,
				WithTitle("One Piece", "One Piece", "ONE PIECE"),
			),
			MockComicInfo(
				2003,
				WithTitle("Bleach", "Bleach", "BLEACH"),
			),
		},
	}
}

// ArchiveScenario 归档场景：已归档/未归档/重定向/已删除混合
func ArchiveScenario() *Scenario {
	return &Scenario{
		Name: "archive",
		Comics: []*api.ComicInfo{
			MockComicInfo(3001, WithArchived("archive/3001.zip")),
			MockComicInfo(3002, WithArchived("archive/3002.zip"), WithStatus(true)),
			MockComicInfo(3003),
			MockComicInfo(3004, WithDeleted()),
			MockComicInfo(3005, WithRedirect(3001)),
		},
	}
}

// TagManagementScenario 标签管理场景：多种标签类型、点赞状态、关系
func TagManagementScenario() *Scenario {
	return &Scenario{
		Name: "tag-management",
		Comics: []*api.ComicInfo{
			MockComicInfo(4001, WithStatus(true), WithTagsV2(
				MockTag(101, "tag", "action"),
				MockTag(102, "tag", "comedy"),
				MockTag(201, "artist", "tanaka"),
			)),
			MockComicInfo(4002, WithStatus(true), WithTagsV2(
				MockTag(101, "tag", "action"),
				MockTag(103, "tag", "drama"),
				MockTag(301, "parody", "one-piece"),
			)),
			MockComicInfo(4003, WithStatus(true), WithTagsV2(
				MockTag(102, "tag", "comedy"),
				MockTag(104, "tag", "romance"),
			)),
		},
	}
}
