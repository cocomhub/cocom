// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"strings"
	"testing"
)

func TestStoragePrefix(t *testing.T) {
	tests := []struct {
		cid  int
		want string
	}{
		{1, "00/00"},
		{99, "00/00"},
		{100, "00/01"},
		{101, "00/01"},
		{12345, "01/23"},
		{123456, "12/34"},
	}
	for _, tt := range tests {
		if got := StoragePrefix(tt.cid); got != tt.want {
			t.Errorf("StoragePrefix(%d) = %q, want %q", tt.cid, got, tt.want)
		}
	}
}

func TestPicInfo_PicType(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"j", "jpg"},
		{"g", "gif"},
		{"p", "png"},
		{"w", "webp"},
		{"x", "jpg"}, // 未知类型回落 jpg
		{"", "jpg"},
	}
	for _, tt := range tests {
		if got := (PicInfo{T: tt.kind}).PicType(); got != tt.want {
			t.Errorf("PicType(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestComicImages_PageName(t *testing.T) {
	images := ComicImages{
		Pages: []PicInfo{{T: "j"}, {T: "p"}, {T: "g"}},
	}
	tests := []struct {
		no   int
		want string
	}{
		{1, "1.jpg"},
		{2, "2.png"},
		{3, "3.gif"},
		{0, ""}, // 越界返回空串
		{4, ""},
	}
	for _, tt := range tests {
		if got := images.PageName(tt.no); got != tt.want {
			t.Errorf("PageName(%d) = %q, want %q", tt.no, got, tt.want)
		}
	}
	// PageNameByIndex = PageName(index+1)
	if got := images.PageNameByIndex(1); got != "2.png" {
		t.Errorf("PageNameByIndex(1) = %q, want %q", got, "2.png")
	}
}

func TestComicInfo_SaveDirName(t *testing.T) {
	jp := &ComicInfo{CID: 1}
	jp.Title.Japanese = "日本語"
	en := &ComicInfo{CID: 2}
	en.Title.English = "English"
	pretty := &ComicInfo{CID: 3}
	pretty.Title.Pretty = "Pretty"

	tests := []struct {
		name string
		info *ComicInfo
		want string
	}{
		{"japanese preferred", jp, "[1] 日本語"},
		{"english fallback", en, "[2] English"},
		{"pretty fallback", pretty, "[3] Pretty"},
		{"unknown fallback", &ComicInfo{CID: 4}, "[4] [[unknown]]4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.SaveDirName(); got != tt.want {
				t.Errorf("SaveDirName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDownloadComicOriginUrl(t *testing.T) {
	got := DownloadComicOriginUrl("123456", "1.jpg")
	valid := false
	for _, domain := range []string{"i1.", "i2.", "i4."} {
		if strings.HasPrefix(got, "https://"+domain+"nhentai.net/galleries/123456/1.jpg") {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("DownloadComicOriginUrl() = %q, want https://i{1,2,4}.nhentai.net/galleries/123456/1.jpg", got)
	}
}

func TestTags_IdString(t *testing.T) {
	tags := Tags{
		{ID: 1, Name: "a"},
		{ID: 2, Name: "b"},
	}
	if got := tags.IdString(); got != "1 2" {
		t.Errorf("IdString() = %q, want %q", got, "1 2")
	}
	if got := (Tags{}).IdString(); got != "" {
		t.Errorf("empty IdString() = %q, want %q", got, "")
	}
}

func TestComicInfo_CheckStatus(t *testing.T) {
	// 全部页面完成 → Status=true
	info := &ComicInfo{Images: ComicImages{Pages: []PicInfo{{Status: true}, {Status: true}}}}
	info.CheckStatus()
	if !info.Status {
		t.Error("CheckStatus should set Status=true when all pages complete")
	}

	// 任一页面未完成 → 双向重算回 false
	info.Images.Pages[1].Status = false
	info.CheckStatus()
	if info.Status {
		t.Error("CheckStatus should set Status=false when any page incomplete")
	}

	// 再次补全 → 回 true
	info.Images.Pages[1].Status = true
	info.CheckStatus()
	if !info.Status {
		t.Error("CheckStatus should recompute Status back to true")
	}

	// 无页面 → true
	empty := &ComicInfo{}
	empty.CheckStatus()
	if !empty.Status {
		t.Error("CheckStatus on empty pages should set Status=true")
	}
}
