// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cocomhub/cocom/cmd/server/internal/custom"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func AddLikeGroup(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse form failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	cid, err := strconv.Atoi(req.FormValue("cid"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse cid failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse cid failed. errmsg: %s", err))
		return
	}
	// 与 UpdateComicTags 的 cid<=0 校验一致，避免写入 {cid:0/-1} 垃圾文档
	if cid <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		slog.WarnContext(ctx, "request cid must be positive", slog.Int("cid", cid))
		httpwrap.ResponseFail(ctx, w, "cid must be a positive integer")
		return
	}

	err = custom.AddLikeGroup(ctx, cid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request add like group failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request add like group failed. errmsg: %s", err))
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}
