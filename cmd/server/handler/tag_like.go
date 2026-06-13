// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/cmd/server/internal/tag"
	"github.com/cocomhub/cocom/pkg/httpwrap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func LikeTag(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	if err := req.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse form failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	tagType := req.FormValue("type")
	idStr := req.FormValue("id")
	name := req.FormValue("name")
	if len(tagType) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "type is required")
		return
	}

	filter := bson.M{"type": tagType}
	var tagID int
	hasID := false
	if len(idStr) != 0 {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid id: %s", err))
			return
		}
		filter["id"] = id
		tagID = id
		hasID = true
	} else if len(name) != 0 {
		filter["name"] = name
	} else {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "id or name is required")
		return
	}

	if s := tag.GetDefaultLikeStore(); s != nil && hasID {
		if err := s.Like(ctx, tagType, tagID); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			httpwrap.ResponseFail(ctx, w, err.Error())
			return
		}
		httpwrap.ResponseSucc(ctx, w, "")
		return
	}

	update := bson.M{"$set": bson.M{"like": true, "updated_at": time.Now()}}
	opts := options.Update().SetUpsert(false)

	result, err := mongo.ComicTag().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "comicTag like update failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("comicTag like update failed. errmsg: %s", err.Error()))
		return
	}
	if result.MatchedCount == 0 {
		w.WriteHeader(http.StatusNotFound)
		httpwrap.ResponseFail(ctx, w, "tag not found")
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}

func UnlikeTag(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	if err := req.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse form failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	tagType := req.FormValue("type")
	idStr := req.FormValue("id")
	name := req.FormValue("name")
	if len(tagType) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "type is required")
		return
	}

	filter := bson.M{"type": tagType}
	var tagID2 int
	hasID2 := false
	if len(idStr) != 0 {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			httpwrap.ResponseFail(ctx, w, fmt.Sprintf("invalid id: %s", err))
			return
		}
		filter["id"] = id
		tagID2 = id
		hasID2 = true
	} else if len(name) != 0 {
		filter["name"] = name
	} else {
		w.WriteHeader(http.StatusBadRequest)
		httpwrap.ResponseFail(ctx, w, "id or name is required")
		return
	}

	if s := tag.GetDefaultLikeStore(); s != nil && hasID2 {
		if err := s.Unlike(ctx, tagType, tagID2); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			httpwrap.ResponseFail(ctx, w, err.Error())
			return
		}
		httpwrap.ResponseSucc(ctx, w, "")
		return
	}

	update := bson.M{"$set": bson.M{"like": false, "updated_at": time.Now()}}
	opts := options.Update().SetUpsert(false)

	result, err := mongo.ComicTag().UpdateOne(ctx, filter, update, opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "comicTag unlike update failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("comicTag unlike update failed. errmsg: %s", err.Error()))
		return
	}
	if result.MatchedCount == 0 {
		w.WriteHeader(http.StatusNotFound)
		httpwrap.ResponseFail(ctx, w, "tag not found")
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}
