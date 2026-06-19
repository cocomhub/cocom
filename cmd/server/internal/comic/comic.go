// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/comic"
)

// Comic 实现comic.Comic接口
type Comic struct {
	*api.ComicInfo
}

// NewComic 创建Comic实例
func NewComic(info *api.ComicInfo) *Comic {
	return &Comic{
		ComicInfo: info,
	}
}

func NewComicByObject(obj any) (*Comic, error) {
	switch v := obj.(type) {
	case Comic:
		return &v, nil
	case *Comic:
		return v, nil
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var comic Comic
		err = json.Unmarshal(data, &comic)
		if err != nil {
			return nil, err
		}
		return &comic, nil
	default:
		return nil, fmt.Errorf("invalid object type: %T", obj)
	}
}

// GetID 实现Comic接口
func (c *Comic) GetID() string {
	return strconv.Itoa(c.CID)
}

// GetArchivePath 实现Comic接口
func (c *Comic) GetArchivePath() string {
	if c.Archive != nil {
		return c.Archive.Path
	}
	return ""
}

// GetTitle 实现Comic接口
func (c *Comic) GetTitle() string {
	return c.Title.Pretty
}

// GetTitleEnglish 实现Comic接口
func (c *Comic) GetTitleEnglish() string {
	return c.Title.English
}

// GetTitleJapanese 实现Comic接口
func (c *Comic) GetTitleJapanese() string {
	return c.Title.Japanese
}

// GetTitlePretty 实现Comic接口
func (c *Comic) GetTitlePretty() string {
	return c.Title.Pretty
}

// IsStatus 实现Comic接口
func (c *Comic) IsStatus() bool {
	return c.Status
}

// IsDeleted 实现Comic接口
func (c *Comic) IsDeleted() bool {
	return c.Deleted
}

// GetRedirectCID 实现Comic接口
func (c *Comic) GetRedirectCID() int {
	if c.RedirectTo != nil {
		return *c.RedirectTo
	}
	return 0
}

// GetImages 实现Comic接口
func (c *Comic) GetImages() []comic.Image {
	images := make([]comic.Image, 0, len(c.Images.Pages))
	for i := range c.Images.Pages {
		images = append(images, comic.Image{
			ID:   strconv.Itoa(i),
			Path: c.PageSavePathByIndex(i),
			URL:  c.PageOriginUrlByIndex(i),
		})
	}
	return images
}

// GetTags 实现Comic接口
func (c *Comic) GetTags() []comic.Tag {
	tags := make([]comic.Tag, 0, len(c.Tags))
	for _, t := range c.Tags {
		tags = append(tags, comic.Tag{
			Count: t.Count,
			ID:    t.ID,
			Name:  t.Name,
			Type:  t.Type,
			URL:   t.URL,
		})
	}
	return tags
}

// Object 实现Comic接口
func (c *Comic) Object() any {
	return c
}

// MarshalJSON 实现comic.Comic接口，用于 GetComicInfo JSON round-trip
// 从 MemoryStorage 读取 ComicImpl 后通过 c.MarshalJSON() 桥接到 api.ComicInfo。
func (c *Comic) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ComicInfo)
}

// UnmarshalJSON 实现comic.Comic接口
func (c *Comic) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, c.ComicInfo)
}
