// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import "github.com/cocomhub/cocom/cmd/server/api"

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
			MockComicInfo(1006, WithStatus(false), WithTagsV2(
				MockTag(6, "tag", "disabled"),
			)),
			MockComicInfo(1007, WithDeleted(), WithTagsV2(
				MockTag(7, "tag", "deleted"),
			)),
			MockComicInfo(1008, WithRedirect(1001), WithTagsV2(
				MockTag(8, "tag", "redirect"),
			)),
		},
	}
}

// SearchScenario 搜索场景：用于测试标题搜索匹配
func SearchScenario() *Scenario {
	return &Scenario{
		Name: "search",
		Comics: []*api.ComicInfo{
			// 注意标题搜索一般搜索 English 和 Japanese 字段
			MockComicInfo(1011, WithTitle("Naruto", "Naruto", "ナルト"),
				WithTagsV2(MockTag(1, "tag", "action"))),
			MockComicInfo(1012, WithTitle("One Piece", "One Piece", "ワンピース"),
				WithTagsV2(MockTag(2, "tag", "adventure"))),
			MockComicInfo(1013, WithTitle("Bleach", "Bleach", "ブリーチ"),
				WithTagsV2(MockTag(3, "tag", "action"))),
		},
	}
}

// ArchiveScenario 归档场景：包含归档/启用/删除等不同归档状态的漫画
func ArchiveScenario() *Scenario {
	return &Scenario{
		Name: "archive",
		Comics: []*api.ComicInfo{
			MockComicInfo(1021, WithArchived("archive/1021.zip"),
				WithStatus(true),
				WithTagsV2(MockTag(1, "tag", "archived"))),
			MockComicInfo(1022, WithArchived("archive/1022.zip"),
				WithStatus(false),
				WithTagsV2(MockTag(2, "tag", "disabled"))),
			MockComicInfo(1023, WithNoArchive(),
				WithTagsV2(MockTag(3, "tag", "normal"))),
			MockComicInfo(1024, WithDeleted(),
				WithTagsV2(MockTag(4, "tag", "deleted"))),
			MockComicInfo(1025, WithRedirect(1021),
				WithTagsV2(MockTag(5, "tag", "redirect"))),
		},
	}
}

// TagManagementScenario 标签管理场景：用于测试标签聚合、标签点赞、标签关系等
func TagManagementScenario() *Scenario {
	return &Scenario{
		Name: "tag-management",
		Comics: []*api.ComicInfo{
			MockComicInfo(1031, WithStatus(true), WithTagsV2(
				MockTag(1, "tag", "action"),
				MockTag(2, "tag", "adventure"),
				MockTag(3, "tag", "comedy"),
			)),
			MockComicInfo(1032, WithStatus(true), WithTagsV2(
				MockTag(1, "tag", "action"),
				MockTag(4, "parody", "naruto"),
				MockTag(5, "tag", "romance"),
			)),
			MockComicInfo(1033, WithStatus(true), WithTagsV2(
				MockTag(1, "tag", "action"),
				MockTag(2, "tag", "adventure"),
				MockTag(3, "tag", "comedy"),
				MockTag(4, "parody", "naruto"),
				MockTag(5, "tag", "romance"),
			)),
		},
	}
}

// E2ECompareScenario 漫画比对场景：两部可比对、一部已禁用
func E2ECompareScenario() *Scenario {
	return &Scenario{
		Name: "e2e-compare",
		Comics: []*api.ComicInfo{
			MockComicInfo(2001, WithStatus(true), WithTagsV2(
				MockTag(1, "tag", "action"),
				MockTag(2, "tag", "adventure"),
				MockTag(4, "parody", "naruto"),
			), WithTitle("Compare A", "Compare A", "Compare A")),
			MockComicInfo(2002, WithStatus(true), WithTagsV2(
				MockTag(2, "tag", "adventure"),
				MockTag(3, "tag", "comedy"),
				MockTag(4, "parody", "naruto"),
			), WithTitle("Compare B", "Compare B", "Compare B")),
			MockComicInfo(2003, WithStatus(false), WithTitle(
				"Compare Disabled", "Compare Disabled", "Compare Disabled",
			)),
		},
	}
}

// E2ESidebarScenario 侧边栏操作场景：可 Like/可归档/可恢复等
func E2ESidebarScenario() *Scenario {
	return &Scenario{
		Name: "e2e-sidebar",
		Comics: []*api.ComicInfo{
			// Comic 3001: 已归档 → 可恢复
			MockComicInfo(3001, WithStatus(true), WithTagsV2(
				MockTag(1, "tag", "action"),
			), WithArchived("archive/3001.zip"),
				WithPages(api.PicInfo{T: "j", W: 800, H: 1200}, api.PicInfo{T: "j", W: 800, H: 1200}),
				WithTitle("Sidebar Comic 1", "Sidebar Comic 1", "Sidebar Comic 1")),
			// Comic 3002: 未归档 → 可归档
			MockComicInfo(3002, WithStatus(true), WithTagsV2(
				MockTag(2, "tag", "adventure"),
			), WithNoArchive(),
				WithPages(api.PicInfo{T: "j", W: 800, H: 1200}, api.PicInfo{T: "j", W: 800, H: 1200}),
				WithTitle("Sidebar Comic 2", "Sidebar Comic 2", "Sidebar Comic 2")),
			// Comic 3003: 有多个页面用于页面管理
			MockComicInfo(3003, WithStatus(true), WithTagsV2(
				MockTag(1, "tag", "action"),
				MockTag(3, "tag", "comedy"),
			), WithNoArchive(),
				WithPages(
					api.PicInfo{T: "j", W: 800, H: 1200},
					api.PicInfo{T: "j", W: 800, H: 1200},
					api.PicInfo{T: "j", W: 800, H: 1200},
				),
				WithTitle("Sidebar Comic 3", "Sidebar Comic 3", "Sidebar Comic 3")),
		},
	}
}
