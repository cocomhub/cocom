// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package httpwrap

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cocomhub/cocom/pkg/clog"
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
	data, _ := json.Marshal(ResponseInfo[T]{
		Head: ResponseHeadInfo{
			Code:      code,
			Msg:       msg,
			RequestID: clog.GetTraceID(ctx),
			Time:      time.Now().Format(time.RFC3339Nano),
		},
		Body: body,
	})
	_, _ = w.Write(data)
}

func ResponseSucc[T any](ctx context.Context, w http.ResponseWriter, body T) {
	Response(ctx, w, 0, "succ", body)
}

func ResponseFail(ctx context.Context, w http.ResponseWriter, msg string) {
	Response[any](ctx, w, -1, msg, nil)
}
