package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suixibing/cocom/pkg/middlewares"
)

func TestHealthzReadyz(t *testing.T) {
	r := BuildEngine(context.Background(), nil)
	s := httptest.NewServer(r)
	defer s.Close()

	resp, err := http.Get(s.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz request error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status = %d", resp.StatusCode)
	}
	if resp.Header.Get(middlewares.HeaderXRequestID) == "" {
		t.Fatalf("healthz missing X-Request-ID header")
	}

	resp2, err := http.Get(s.URL + "/readyz")
	if err != nil {
		t.Fatalf("readyz request error: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("readyz status = %d", resp2.StatusCode)
	}
	if resp2.Header.Get(middlewares.HeaderXRequestID) == "" {
		t.Fatalf("readyz missing X-Request-ID header")
	}
}
