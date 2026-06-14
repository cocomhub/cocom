// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
)

// MockComicInfo 创建测试用 ComicInfo，可通过 opts 自定义字段。
//
// 用法:
//
//	comic := MockComicInfo(1001)
func MockComicInfo(cid int, opts ...func(*api.ComicInfo)) *api.ComicInfo {
	info := &api.ComicInfo{
		CID: cid,
		Title: struct {
			English  string `json:"english,omitempty" bson:"english"`
			Japanese string `json:"japanese,omitempty" bson:"japanese"`
			Pretty   string `json:"pretty,omitempty" bson:"pretty"`
		}{
			Pretty:   "测试漫画",
			English:  "Test Comic",
			Japanese: "テストコミック",
		},
		Images: api.ComicImages{
			Pages:     make([]api.PicInfo, 0),
			Cover:     api.PicInfo{T: "j", W: 350, H: 500},
			Thumbnail: api.PicInfo{T: "j", W: 250, H: 375},
		},
		NumPages: 0,
		Status:   true,
		MediaId:  strconv.Itoa(cid + 100000),
	}
	for _, opt := range opts {
		opt(info)
	}
	return info
}

// WithTitle 设置标题各语言版本
func WithTitle(pretty, english, japanese string) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Title = struct {
			English  string `json:"english,omitempty" bson:"english"`
			Japanese string `json:"japanese,omitempty" bson:"japanese"`
			Pretty   string `json:"pretty,omitempty" bson:"pretty"`
		}{
			Pretty:   pretty,
			English:  english,
			Japanese: japanese,
		}
	}
}

// WithTags 设置标签列表（通过 []api.TagBrief 格式）
func WithTags(tags ...api.TagBrief) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		for _, t := range tags {
			info.Tags = append(info.Tags, api.Tag{
				ID:   t.ID,
				Type: t.Type,
				Name: t.Name,
			})
		}
	}
}

// WithPages 设置页面列表，同时更新 NumPages
func WithPages(pages ...api.PicInfo) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Images.Pages = pages
		info.NumPages = len(pages)
	}
}

// WithArchive 设置归档路径和 MD5
func WithArchive(path, md5 string) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Archive = &api.ArchiveInfo{
			Path: path,
			MD5:  md5,
		}
	}
}

// WithDeleted 标记为已删除（设置 Deleted=true, Status=false）
func WithDeleted() func(*api.ComicInfo) {
	return func(info *api.ComicInfo) {
		info.Deleted = true
		info.Status = false
	}
}

// WithUploadDate 设置上传时间戳
func WithUploadDate(ts int64) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) { info.UploadDate = ts }
}

// WithStatus 设置启用状态
func WithStatus(status bool) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) { info.Status = status }
}

// WithTagsV2 设置标签列表（直接替换 Tags 字段）
func WithTagsV2(tags ...api.Tag) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) { info.Tags = tags }
}

// WithArchived 设置归档状态（仅路径版本，兼容新旧 Archive 结构）
func WithArchived(path string) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) { info.Archive = &api.ArchiveInfo{Path: path} }
}

// WithNoArchive 清除归档信息
func WithNoArchive() func(*api.ComicInfo) {
	return func(info *api.ComicInfo) { info.Archive = nil }
}

// WithRedirect 设置重定向 CID
func WithRedirect(cid int) func(*api.ComicInfo) {
	return func(info *api.ComicInfo) { info.RedirectTo = &cid }
}

// MockTag 快速创建 Tag
func MockTag(id int, typ, name string) api.Tag {
	return api.Tag{ID: id, Type: typ, Name: name}
}
