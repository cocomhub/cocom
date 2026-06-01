// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

type TagInfo struct {
	ID       int    `bson:"id" json:"id"`
	Name     string `bson:"name" json:"name"`
	Type     string `bson:"type" json:"type"`
	URL      string `bson:"url" json:"url"`
	Count    int    `bson:"count" json:"count"`
	Like     bool   `bson:"like" json:"like"`
	Explicit bool   `bson:"explicit,omitempty" json:"explicit,omitempty"`
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

// UpdateTagsRequest 单本漫画 tag 更新请求
type UpdateTagsRequest struct {
	CID     int   `json:"cid"`
	Added   []Tag `json:"added,omitempty"`
	Removed []Tag `json:"removed,omitempty"`
}

// BatchAddTagRequest 批量添加 tag 请求
type BatchAddTagRequest struct {
	CIDList []int `json:"cidList"`
	Tag     Tag   `json:"tag"`
}

// BatchAddTagResponse 批量添加 tag 响应
type BatchAddTagResponse struct {
	Updated int   `json:"updated"`
	Errors  []int `json:"errors,omitempty"`
}

// SearchUniqueTagsResponse 搜索结果去重 tag 响应
type SearchUniqueTagsResponse struct {
	Tags    []*TagInfo `json:"tags"`
	CIDList []int      `json:"cidList"`
	Total   int        `json:"total"`
}

// RelatedTagsResponse 关联 tag 响应
type RelatedTagsResponse struct {
	Tags []*TagInfo `json:"tags"`
}

// TagBrief 关系组中单个 tag 的简要信息
type TagBrief struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// CreateRelationRequest 创建关系组请求
type CreateRelationRequest struct {
	Tags []TagBrief `json:"tags"`
}

// CreateRelationResponse 创建关系组响应
type CreateRelationResponse struct {
	ID        string     `json:"id"`
	Tags      []TagBrief `json:"tags"`
	CreatedAt string     `json:"created_at"`
}

// DeleteRelationRequest 删除关系组请求
type DeleteRelationRequest struct {
	ID string `json:"id"`
}

// RelationGroup 关系组信息（用于前端展示）
type RelationGroup struct {
	ID        string     `json:"id"`
	Tags      []TagBrief `json:"tags"`
	CreatedAt string     `json:"created_at"`
}

// GetRelationsResponse 获取关系组列表响应
type GetRelationsResponse struct {
	Groups []RelationGroup `json:"groups"`
}
