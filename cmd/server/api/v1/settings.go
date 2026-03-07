// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"net/http"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/setting"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/gin-gonic/gin"
)

func GetSettings(c *gin.Context) {
	ctx := c.Request.Context()
	settingType := c.Query("type")
	keysParam := c.Query("keys")
	var keys []string
	if keysParam != "" {
		keys = strings.Split(keysParam, ",")
	} else {
		keys = []string{""}
	}
	result, err := setting.GetSettings(ctx, settingType, keys...)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}
	httpwrap.GinRespondOK(c, result)
}

func SetSettings(c *gin.Context) {
	ctx := c.Request.Context()
	var req api.SetSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpwrap.GinRespondError(c, http.StatusBadRequest, -1, err.Error())
		return
	}
	if err := setting.SetSettings(ctx, req.Type, req.Settings); err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}
	httpwrap.GinRespondOK(c, "")
}

func DelSettings(c *gin.Context) {
	ctx := c.Request.Context()
	settingType := c.Query("type")
	keysParam := c.Query("keys")
	var keys []string
	if keysParam != "" {
		keys = strings.Split(keysParam, ",")
	} else {
		keys = []string{""}
	}
	if _, err := setting.DelSettings(ctx, settingType, keys...); err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}
	httpwrap.GinRespondOK(c, "")
}
