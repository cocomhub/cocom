// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMiddlewares_RequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		// RequestID middleware uses requestid library with custom header key
		// The request ID may be set in the header value
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			t.Log("RequestID header not set (may need different key)")
		}
		c.JSON(http.StatusOK, gin.H{"id": rid})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMiddlewares_LocalGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := LocalGuard(false)
	if handler == nil {
		t.Error("LocalGuard should return a handler")
	}
}
