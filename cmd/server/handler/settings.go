// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/setting"
	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/httpwrap"
)

func GetSetting(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse form failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	settings, err := setting.GetSettings(ctx, req.FormValue("type"), strings.Split(req.FormValue("keys"), ",")...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "get settings failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("get settings failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, settings)
}

func SetSetting(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	info := api.SetSettingsRequest{}
	err := json.NewDecoder(req.Body).Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "decode body failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("decode body failed. errmsg: %s", err))
		return
	}
	slog.DebugContext(ctx, "req info", slog.String("info", conv.JSON(info)))

	err = setting.SetSettings(ctx, info.Type, info.Settings)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "set settings failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("set settings failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, "")
}

func DelSetting(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.ErrorContext(ctx, "request parse form failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("request parse form failed. errmsg: %s", err))
		return
	}

	_, err = setting.DelSettings(ctx, req.FormValue("type"), strings.Split(req.FormValue("keys"), ",")...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.ErrorContext(ctx, "del settings failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, fmt.Sprintf("del settings failed. errmsg: %s", err))
		return
	}

	httpwrap.ResponseSucc(ctx, w, "")
}
