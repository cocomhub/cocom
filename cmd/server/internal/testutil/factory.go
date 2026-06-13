// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"github.com/cocomhub/cocom/cmd/server/api"
)

// MockComicInfo 创建测试用 ComicInfo，可通过 opts 自定义字段。
//
// 用法:
//
//	comic := MockComicInfo(1001)
//	comic := MockComicInfo(1002, WithTitle("My Title", "My Title", "マイタイトル"))
//	comic := MockComicInfo(1003, WithTags(MockTag(1, "tag", "test")))
func MockComicInfo(cid int, opts ...func(*api.ComicInfo)) *api.ComicInfo {
	info := &api.ComicInfo{
		CID: cid,
		Title: struct {
			English  string `json:"english,omitempty" bson:"english"`
			Japanese string `json:"japanese,omitempty" bson:"japanese"`
			Pretty   string `json:"pretty,omitempty" bson:"pretty"`
		}{
			Pretty:   "Test Comic",
			English:  "Test Comic",
			Japanese: "テストコミック",
		},
		Images: api.ComicImages{
			Pages: []api.PicInfo{
				{W: 100, H: 100, T: "j"},
			},
		},
		NumPages: 1,
	}
	for _, opt := range opts {
		opt(info)
	}
	return info
}

// MockComicInfos 批量创建测试用 ComicInfo，每个 cid 创建一个。
//
//	comics := MockComicInfos([]int{101, 102}, WithTags(MockTag(1, "tag", "shared")))
func MockComicInfos(cids []int, opts ...func(*api.ComicInfo)) []*api.ComicInfo {
	infos := make([]*api.ComicInfo, len(cids))
	for i, cid := range cids {
		infos[i] = MockComicInfo(cid, opts...)
	}
	return infos
}

// WithTitle 设置漫画标题。
func WithTitle(pretty, english, japanese string) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Title.Pretty = pretty
		info.Title.English = english
		info.Title.Japanese = japanese
	}
}

// WithTags 设置漫画标签。
func WithTags(tags ...api.Tag) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Tags = tags
	}
}

// WithPages 设置漫画页面。会自动更新 NumPages。
func WithPages(pages ...api.PicInfo) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Images.Pages = pages
		info.NumPages = len(pages)
	}
}

// WithArchive 设置漫画归档信息。
func WithArchive(path, md5 string) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Archive = &api.ArchiveInfo{
			Path: path,
			MD5:  md5,
		}
	}
}

// WithDeleted 标记漫画为已删除。
func WithDeleted() func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Deleted = true
	}
}

// WithUploadDate 设置上传时间戳。
func WithUploadDate(ts int64) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.UploadDate = ts
	}
}

// MockTag 快速创建 Tag。
func MockTag(id int, typ, name string) api.Tag {
	return api.Tag{
		ID:   id,
		Type: typ,
		Name: name,
	}
}

// WithArchived 设置归档状态（仅路径版本）
func WithArchived(path string) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Archive = &api.ArchiveInfo{Path: path}
	}
}

// WithRedirect 设置重定向 CID
func WithRedirect(cid int) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.RedirectTo = &cid
	}
}

// WithStatus 设置启用状态
func WithStatus(status bool) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) { info.Status = status }
}

// WithTagsV2 设置标签列表（与 WithTags 功能相同，用于场景预设中的显式调用）
func WithTagsV2(tags ...api.Tag) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) { info.Tags = tags }
}
