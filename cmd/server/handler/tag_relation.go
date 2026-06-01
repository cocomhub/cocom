// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

// CreateTagRelation POST /api/comic/tags/relation
func CreateTagRelation(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var body api.CreateRelationRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode CreateRelationRequest failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid request body: %s", err))
		return
	}

	if len(body.Tags) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "at least 2 tags are required")
		return
	}

	doc, err := tag.CreateRelation(ctx, body.Tags)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "CreateRelation failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("create relation failed: %s", err))
		return
	}

	var createdAt string
	if !doc.CreatedAt.IsZero() {
		createdAt = doc.CreatedAt.Format("2006-01-02 15:04:05")
	}

	tags := make([]api.TagBrief, len(doc.Tags))
	for i, t := range doc.Tags {
		tags[i] = api.TagBrief{ID: t.ID, Name: t.Name, Type: t.Type, URL: t.URL}
	}

	httpwrap.ResponseSucc(ctx, w, api.CreateRelationResponse{
		ID:        doc.ID.Hex(),
		Tags:      tags,
		CreatedAt: createdAt,
	})
}

// DeleteTagRelation DELETE /api/comic/tags/relation
func DeleteTagRelation(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var body api.DeleteRelationRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode DeleteRelationRequest failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid request body: %s", err))
		return
	}

	if body.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "id is required")
		return
	}

	if err := tag.DeleteRelation(ctx, body.ID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "DeleteRelation failed", slog.String("id", body.ID), slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("delete relation failed: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, "relation deleted")
}

// GetTagRelations GET /api/comic/tags/relation?type=...&name=...&id=...
func GetTagRelations(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	tagType := req.URL.Query().Get("type")
	tagName := req.URL.Query().Get("name")
	if tagType == "" || tagName == "" {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "type and name are required")
		return
	}

	// 通过 name 查找 tag id
	curTag, _ := tag.GetTagByTypeName(ctx, tagType, tagName)
	if curTag == nil || curTag.ID == 0 {
		httpwrap.ResponseSucc(ctx, w, api.GetRelationsResponse{Groups: []api.RelationGroup{}})
		return
	}

	// 指定 id 查询，提高准确性
	if idStr := req.URL.Query().Get("id"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil && id > 0 {
			curTag.ID = id
		}
	}

	groups, err := tag.GetRelationsGroupList(ctx, curTag.Type, curTag.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "GetRelationsGroupList failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get relations failed: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, api.GetRelationsResponse{Groups: groups})
}
