// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package httpwrap

import (
	"net/http"
	"time"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/gin-gonic/gin"
)

func GinRespond[T any](c *gin.Context, httpStatus int, code int, msg string, body T) {
	info := ResponseInfo[T]{
		Head: ResponseHeadInfo{
			Code:      code,
			Msg:       msg,
			RequestID: clog.GetTraceID(c.Request.Context()),
			Time:      time.Now().Format(time.RFC3339Nano),
		},
		Body: body,
	}
	c.JSON(httpStatus, info)
}

func GinRespondOK[T any](c *gin.Context, body T) {
	GinRespond(c, http.StatusOK, 0, "succ", body)
}

func GinRespondError(c *gin.Context, httpStatus int, code int, msg string) {
	GinRespond[any](c, httpStatus, code, msg, nil)
}
