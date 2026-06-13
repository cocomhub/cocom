// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

// TagInfo 标签信息（从漫画数据推导）
type TagInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	URL   string `json:"url"`
	Count int    `json:"count"`
	Like  bool   `json:"like"`
}
