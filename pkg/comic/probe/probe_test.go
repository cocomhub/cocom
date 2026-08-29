// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
)

func TestParseIDsFromIndexV2(t *testing.T) {
	path := filepath.Join("index.html")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("index.html not found, skip")
		}
		t.Fatalf("failed to read index.html: %v", err)
	}
	ids, err := parseIDsFromIndexV2(string(data), 0)
	if err != nil {
		t.Fatalf("parseIDsFromIndexV2 error: %v", err)
	}
	if len(ids) == 0 {
		t.Fatalf("expected at least one id, got 0")
	}
	t.Logf("parsed %d ids: %v", len(ids), ids)
	for i, id := range ids {
		if id <= 0 {
			t.Fatalf("id at index %d is not positive: %d", i, id)
		}
	}
}

func TestV2DetailJSONMapping(t *testing.T) {
	raw := `{
		"id": 123456,
		"media_id": "9999999",
		"title": {
			"english": "Sample Title EN",
			"japanese": "サンプルタイトル"
		},
		"images": {
			"pages": [
				{ "t": "j", "w": 1000, "h": 1500 }
			],
			"cover": { "t": "j", "w": 350, "h": 500 },
			"thumbnail": { "t": "j", "w": 250, "h": 350 }
		},
		"num_pages": 1
	}`
	info := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		t.Fatalf("unmarshal gallery json failed: %v", err)
	}
	info["cid"] = 123456
	// Validate that our info can be marshaled into api.ComicInfo like genDownList does
	b, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal back failed: %v", err)
	}
	var ci api.ComicInfo
	if err := json.Unmarshal(b, &ci); err != nil {
		t.Fatalf("unmarshal into api.ComicInfo failed: %v", err)
	}
	if ci.MediaId == "" || len(ci.Images.Pages) != 1 {
		t.Fatalf("unexpected mapping: media_id=%q pages=%d", ci.MediaId, len(ci.Images.Pages))
	}
}

func TestParseV2DetailFromHTML(t *testing.T) {
	ctx := context.Background()
	info, err := parseComicPageV2(ctx, 640503)
	if err != nil {
		t.Fatalf("parseComicPageV2 error: %v", err)
	}
	if info["media_id"] == nil || info["media_id"] == "" {
		t.Fatalf("media_id missing")
	}
	// ensure images.pages exist and non-empty
	images, ok := info["images"].(map[string]any)
	if !ok {
		t.Fatalf("images not found")
	}
	pages, ok := images["pages"].([]any)
	if !ok || len(pages) == 0 {
		t.Fatalf("pages not found or empty")
	}
	// ensure can unmarshal into api.ComicInfo
	b, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal back failed: %v", err)
	}
	var ci api.ComicInfo
	if err := json.Unmarshal(b, &ci); err != nil {
		t.Fatalf("unmarshal into api.ComicInfo failed: %v", err)
	}
	if ci.MediaId == "" || len(ci.Images.Pages) == 0 {
		t.Fatalf("unexpected mapping: media_id=%q pages=%d", ci.MediaId, len(ci.Images.Pages))
	}
}

func TestNormalizeV2ToV1(t *testing.T) {
	p := filepath.Join("comicInfo.v2.json")
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("comicInfo.v2.json not found, skip")
		}
		t.Fatalf("read v2 sample failed: %v", err)
	}
	var info map[string]any
	if umErr := json.Unmarshal(data, &info); umErr != nil {
		t.Fatalf("unmarshal v2 sample failed: %v", umErr)
	}
	out := normalizeV2ToV1(info)
	img, ok := out["images"].(map[string]any)
	if !ok {
		t.Fatalf("images missing after normalize")
	}
	pages, ok := img["pages"].([]any)
	if !ok || len(pages) == 0 {
		t.Fatalf("pages missing after normalize")
	}
	for i, p := range pages {
		m, _ := p.(map[string]any)
		if m == nil {
			t.Fatalf("page %d not a map", i)
		}
		if _, ok := m["t"]; !ok {
			t.Fatalf("page %d missing t", i)
		}
		if _, ok := m["w"]; !ok {
			t.Fatalf("page %d missing w", i)
		}
		if _, ok := m["h"]; !ok {
			t.Fatalf("page %d missing h", i)
		}
		if _, ok := m["path"]; ok {
			t.Fatalf("page %d should not contain path", i)
		}
	}
	for _, k := range []string{"cover", "thumbnail"} {
		m, _ := img[k].(map[string]any)
		if m == nil {
			t.Fatalf("%s missing after normalize", k)
		}
		for _, key := range []string{"t", "w", "h"} {
			if _, ok := m[key]; !ok {
				t.Fatalf("%s missing key %s", k, key)
			}
		}
		if _, ok := m["path"]; ok {
			t.Fatalf("%s should not contain path", k)
		}
	}
	// Unmarshal to api.ComicInfo and verify URL generation uses extension from t
	b, err := json.MarshalIndent(out, "", "\t")
	if err != nil {
		t.Fatalf("marshal normalized failed: %v", err)
	}
	os.WriteFile("convert_v2_to_v1.json", b, 0o766)
	var ci api.ComicInfo
	if err := json.Unmarshal(b, &ci); err != nil {
		t.Fatalf("unmarshal into api.ComicInfo failed: %v", err)
	}
	if ci.MediaId == "" || len(ci.Images.Pages) == 0 {
		t.Fatalf("unexpected ci mapping")
	}
	url := ci.PageOriginUrlByIndex(0)
	if !strings.Contains(url, "galleries/") {
		t.Fatalf("origin url invalid: %s", url)
	}
	_ = t                                                                                                                                         //nolint:staticcheck
	if !(strings.HasSuffix(url, ".jpg") || strings.HasSuffix(url, ".png") || strings.HasSuffix(url, ".webp") || strings.HasSuffix(url, ".gif")) { //nolint:staticcheck
		t.Fatalf("origin url extension invalid: %s", url)
	}
}
