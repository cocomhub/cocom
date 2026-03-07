// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

type OneComicInfo struct {
	Comicid       string `json:"comicid"`
	Name          string `json:"name"`
	Desc          string `json:"desc"`
	Tag           string `json:"tag"`
	CoverImageUrl string `json:"cover_image_url"`
	Author        string `json:"author"`
	SourceUrl     string `json:"source_url"`
	SourceName    string `json:"source_name"`
	CrawlTime     string `json:"crawl_time"`
	Chapters      []struct {
		Title         string `json:"title"`
		ChapterNumber int    `json:"chapter_number"`
		SourceUrl     string `json:"source_url"`
	} `json:"chapters"`
	ExtChapters    []any  `json:"ext_chapters"`
	Status         string `json:"status"`
	Tags           []any  `json:"tags"`
	Site           string `json:"site"`
	LastUpdateTime string `json:"last_update_time"`

	VerifyInfo `json:"verify" bson:"verify"`
}
