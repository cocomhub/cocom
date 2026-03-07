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

func NewComicByObject(obj interface{}) (*Comic, error) {
	switch v := obj.(type) {
	case Comic:
		return &v, nil
	case *Comic:
		return v, nil
	case map[string]interface{}:
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

// GetTitle 实现Comic接口
func (c *Comic) GetTitle() string {
	return c.Title.Pretty
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

// Object 实现Comic接口
func (c *Comic) Object() interface{} {
	return c
}

// MarshalJSON 实现Comic接口
func (c *Comic) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ComicInfo)
}

// UnmarshalJSON 实现Comic接口
func (c *Comic) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, c.ComicInfo)
}
