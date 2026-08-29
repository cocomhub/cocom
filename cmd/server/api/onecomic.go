// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

type OneComicInfo struct {
	Comicid       string `json:"comicid" bson:"comicid"`
	Name          string `json:"name" bson:"name"`
	Desc          string `json:"desc" bson:"desc"`
	Tag           string `json:"tag" bson:"tag"`
	CoverImageUrl string `json:"cover_image_url" bson:"cover_image_url"`
	Author        string `json:"author" bson:"author"`
	SourceUrl     string `json:"source_url" bson:"source_url"`
	SourceName    string `json:"source_name" bson:"source_name"`
	CrawlTime     string `json:"crawl_time" bson:"crawl_time"`
	Chapters      []struct {
		Title         string `json:"title" bson:"title"`
		ChapterNumber int    `json:"chapter_number" bson:"chapter_number"`
		SourceUrl     string `json:"source_url" bson:"source_url"`
	} `json:"chapters" bson:"chapters"`
	ExtChapters    []any  `json:"ext_chapters" bson:"ext_chapters"`
	Status         string `json:"status" bson:"status"`
	Tags           []any  `json:"tags" bson:"tags"`
	Site           string `json:"site" bson:"site"`
	LastUpdateTime string `json:"last_update_time" bson:"last_update_time"`

	VerifyInfo `json:"verify" bson:"verify"`
}
