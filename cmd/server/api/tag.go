// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

type TagInfo struct {
	ID    int    `bson:"id"`
	Name  string `bson:"name"`
	Type  string `bson:"type"`
	URL   string `bson:"url"`
	Count int    `bson:"count"`
	Like  bool   `bson:"like"`
}

func (t *TagInfo) SectionName() string {
	if len(t.Name) == 0 {
		return "#"
	}
	if t.Name[0] >= 'a' && t.Name[0] <= 'z' {
		return string(t.Name[0] - 'a' + 'A')
	}
	if t.Name[0] >= 'A' && t.Name[0] <= 'Z' {
		return string(t.Name[0])
	}
	return "#"
}

type TagSectionIndex struct {
	Name  string
	Index int
	Page  int
}
