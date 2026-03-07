// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"fmt"
	"net/http"

	"github.com/cocomhub/cocom/cmd/server/internal/cache"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func ResetCache(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := cache.Reset()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "reset cache failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("reset cache failed. errmsg: %s", err))
		return
	}

	err = cache.ResetStats()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		clog.Errorf(ctx, "reset cache stats failed. errmsg: %s", err)
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("reset cache stats failed. errmsg: %s", err))
		return
	}
	httpwrap.ResponseSucc(ctx, w, "")
}
