package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/spf13/viper"
)

type head struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	Time      string `json:"time"`
}
type respBody struct {
	Head head                   `json:"head"`
	Body map[string]interface{} `json:"body"`
}

func TestSettingsV1AndAlias(t *testing.T) {
	viper.Set("server.ratelimit.enabled", false)
	r := BuildEngine(context.Background(), nil)
	s := httptest.NewServer(r)
	defer s.Close()

	// GET empty should return OK with empty map
	v := url.Values{}
	v.Set("type", "it_case")
	getURL := s.URL + "/api/settings?" + v.Encode()
	res1, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("GET /api/settings error: %v", err)
	}
	defer res1.Body.Close()
	if res1.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/settings status = %d", res1.StatusCode)
	}
	var r1 respBody
	if err := json.NewDecoder(res1.Body).Decode(&r1); err != nil {
		t.Fatalf("decode response error: %v", err)
	}
	if r1.Head.Code != 0 || r1.Head.RequestID == "" {
		t.Fatalf("unexpected head: %+v", r1.Head)
	}
	if len(r1.Body) != 0 {
		t.Fatalf("expected empty settings, got: %v", r1.Body)
	}

	// POST create settings
	postBody := map[string]interface{}{
		"type": "it_case",
		"settings": map[string]interface{}{
			"a": 1,
			"b": "x",
		},
	}
	data, _ := json.Marshal(postBody)
	res2, err := http.Post(s.URL+"/api/settings", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST /api/settings error: %v", err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/settings status = %d", res2.StatusCode)
	}

	// GET with keys should return subset
	v2 := url.Values{}
	v2.Set("type", "it_case")
	v2.Set("keys", "a,b")
	res3, err := http.Get(s.URL + "/api/settings?" + v2.Encode())
	if err != nil {
		t.Fatalf("GET /api/settings with keys error: %v", err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/settings with keys status = %d", res3.StatusCode)
	}
	var r3 respBody
	_ = json.NewDecoder(res3.Body).Decode(&r3)
	if _, ok := r3.Body["a"]; !ok {
		t.Fatalf("missing key a in body: %v", r3.Body)
	}
	if _, ok := r3.Body["b"]; !ok {
		t.Fatalf("missing key b in body: %v", r3.Body)
	}

	// DELETE one key
	reqDel, _ := http.NewRequest(http.MethodDelete, s.URL+"/api/settings?type=it_case&keys=a", nil)
	res4, err := http.DefaultClient.Do(reqDel)
	if err != nil {
		t.Fatalf("DELETE /api/settings error: %v", err)
	}
	defer res4.Body.Close()
	if res4.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /api/settings status = %d", res4.StatusCode)
	}
	// verify only b remains
	res5, err := http.Get(s.URL + "/api/settings?type=it_case&keys=a,b")
	if err != nil {
		t.Fatalf("GET /api/settings after delete error: %v", err)
	}
	defer res5.Body.Close()
	var r5 respBody
	_ = json.NewDecoder(res5.Body).Decode(&r5)
	if _, ok := r5.Body["a"]; ok {
		t.Fatalf("key a should be deleted, got body: %v", r5.Body)
	}
	if _, ok := r5.Body["b"]; !ok {
		t.Fatalf("key b expected, got body: %v", r5.Body)
	}
}
