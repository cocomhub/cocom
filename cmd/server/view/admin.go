// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminPageData struct {
	URL string
}

func (p *AdminPageData) IsNavigationActive(name string) bool {
	return false
}

func AdminPage(c *gin.Context) {
	data := &AdminPageData{
		URL: c.Request.URL.Path,
	}
	c.HTML(http.StatusOK, "admin.tpl", data)
}
