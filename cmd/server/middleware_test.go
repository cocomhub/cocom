// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestCORSAndGzip(t *testing.T) {
	viper.Set("server.cors.enabled", true)
	viper.Set("server.cors.allow_origins", "*")
	viper.Set("server.cors.allow_methods", "GET,POST,DELETE,OPTIONS")
	viper.Set("server.cors.allow_headers", "X-Requested-With,Content-Type")
	viper.Set("server.gzip.enabled", true)

	r := BuildEngine(context.Background(), nil)
	s := httptest.NewServer(r)
	defer s.Close()

	reqOpt, _ := http.NewRequest(http.MethodOptions, s.URL+"/healthz", nil)
	reqOpt.Header.Set("Origin", "http://example.com")
	reqOpt.Header.Set("Access-Control-Request-Method", "GET")
	reqOpt.Header.Set("Access-Control-Request-Headers", "X-Requested-With,Content-Type")

	respOpt, err := http.DefaultClient.Do(reqOpt)
	if err != nil {
		t.Fatalf("OPTIONS /healthz request error: %v", err)
	}
	defer respOpt.Body.Close()

	if respOpt.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS /healthz status = %d, want 204", respOpt.StatusCode)
	}
	if respOpt.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("missing Access-Control-Allow-Origin header for CORS preflight")
	}
	if respOpt.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("missing Access-Control-Allow-Methods header for CORS preflight")
	}
	if respOpt.Header.Get("Access-Control-Allow-Headers") == "" {
		t.Fatalf("missing Access-Control-Allow-Headers header for CORS preflight")
	}

	reqGet, _ := http.NewRequest(http.MethodGet, s.URL+"/healthz", nil)
	reqGet.Header.Set("Accept-Encoding", "gzip")
	reqGet.Header.Set("Origin", "http://example.com")

	respGet, err := http.DefaultClient.Do(reqGet)
	if err != nil {
		t.Fatalf("GET /healthz request error: %v", err)
	}
	defer respGet.Body.Close()

	if ce := respGet.Header.Get("Content-Encoding"); ce != "gzip" {
		t.Fatalf("Content-Encoding = %q, want %q", ce, "gzip")
	}
	if respGet.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("missing Access-Control-Allow-Origin header for GET")
	}
	_, _ = io.Copy(io.Discard, respGet.Body)
}

func TestRateLimit(t *testing.T) {
	viper.Set("server.ratelimit.enabled", true)
	viper.Set("server.ratelimit.rps", 1)
	viper.Set("server.ratelimit.burst", 1)

	r := BuildEngine(context.Background(), nil)
	s := httptest.NewServer(r)
	defer s.Close()

	client := &http.Client{Timeout: 3 * time.Second}

	req1, _ := http.NewRequest(http.MethodGet, s.URL+"/healthz", nil)
	req2, _ := http.NewRequest(http.MethodGet, s.URL+"/healthz", nil)

	resp1, err1 := client.Do(req1)
	if err1 != nil {
		t.Fatalf("first request error: %v", err1)
	}
	defer resp1.Body.Close()

	resp2, err2 := client.Do(req2)
	if err2 != nil {
		t.Fatalf("second request error: %v", err2)
	}
	defer resp2.Body.Close()

	s1 := resp1.StatusCode
	s2 := resp2.StatusCode

	if !((s1 == http.StatusOK && s2 == http.StatusTooManyRequests) ||
		(s2 == http.StatusOK && s1 == http.StatusTooManyRequests)) {
		t.Fatalf("unexpected statuses: got (%d, %d), want one 200 and one 429", s1, s2)
	}

	viper.Set("server.ratelimit.enabled", false)
}
