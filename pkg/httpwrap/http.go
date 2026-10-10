// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package httpwrap

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/cocomhub/cocom/pkg/logging"
)

type ResponseHeadInfo struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	Time      string `json:"time"`
}

type ResponseInfo[T any] struct {
	Head ResponseHeadInfo `json:"head"`
	Body T                `json:"body,omitempty"`
}

func Response[T any](ctx context.Context, w http.ResponseWriter, code int, msg string, body T) {
	data, err := json.Marshal(ResponseInfo[T]{
		Head: ResponseHeadInfo{
			Code:      code,
			Msg:       msg,
			RequestID: logging.GetTraceID(ctx),
			Time:      time.Now().Format(time.RFC3339Nano),
		},
		Body: body,
	})
	if err != nil {
		// Marshal 失败不 panic，记录日志后写兜底空响应，保持与 Marshal 成功路径一致的调用方行为。
		slog.ErrorContext(ctx, "httpwrap.Response marshal failed", slog.Any("err", err))
	}
	n, err := w.Write(data)
	if err != nil {
		slog.ErrorContext(ctx, "httpwrap.Response write failed", slog.Any("err", err), slog.Int("bytes_written", n))
	}
}

func ResponseSucc[T any](ctx context.Context, w http.ResponseWriter, body T) {
	Response(ctx, w, 0, "succ", body)
}

func ResponseFail(ctx context.Context, w http.ResponseWriter, msg string) {
	Response[any](ctx, w, -1, msg, nil)
}
