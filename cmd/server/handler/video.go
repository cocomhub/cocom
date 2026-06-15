// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cocomhub/cocom/cmd/server/internal/video"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mutex"
)

func SaveVideoInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	info := map[string]any{}
	err := json.NewDecoder(req.Body).Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode body failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	slog.DebugContext(ctx, "req info", slog.String("info", conv.JSON(info)))

	_, exist := info["id"]
	if !exist {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "video id not found failed")
		httpwrap.ResponseFail(ctx, w, "video id not found failed")
		return
	}

	vid := fmt.Sprint(info["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse vid failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse vid failed. errmsg: %s", err))
		return
	}

	unlock, err := mutex.Lock(ctx, fmt.Sprintf("video/%s", vid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		slog.ErrorContext(ctx, "mutex lock failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	err = video.UpdateVideoInfo(ctx, vid, info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "update video info failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("update video info failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, "")
}

func GetVideoInfo(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse form failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	vid := req.FormValue("id")
	if len(vid) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse vid failed", slog.String("errmsg", "vid not found"))
		httpwrap.ResponseFail(ctx, w, "vid not found")
		return
	}

	unlock, err := mutex.Lock(ctx, fmt.Sprintf("video/%s", vid))
	if err != nil {
		w.WriteHeader(http.StatusTooManyRequests)
		slog.ErrorContext(ctx, "mutex lock failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("mutex lock failed. errmsg: %s", err))
		return
	}
	defer unlock()

	info := map[string]any{}
	err = video.GetVideoInfo(ctx, vid, &info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "get video info failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get video info failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, info)
}
