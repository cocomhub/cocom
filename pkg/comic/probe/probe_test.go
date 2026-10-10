// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package probe

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
)

// sveltekitHTML 构造一个仅含「sveltekit-fetched JSON script 标签」的最小 HTML，
// 使解析用例完全离线、确定性（不再依赖 index.html / comicInfo.v2.json 等外部 fixture，
// 也不再经 scraperNative 访问外网）。
func sveltekitHTML(t *testing.T, dataURL string, body any) string {
	t.Helper()

	rawBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal fixture body: %v", err)
	}
	wrapper, err := json.Marshal(map[string]string{"body": string(rawBody)})
	if err != nil {
		t.Fatalf("marshal fixture wrapper: %v", err)
	}
	return `<html><body><script type="application/json" data-sveltekit-fetched data-url="` +
		dataURL + `">` + string(wrapper) + `</script></body></html>`
}

func TestParseIDsFromIndexV2(t *testing.T) {
	body := map[string]any{
		"result": []map[string]any{
			{"id": 111, "tag_ids": []int{6346}},  // 语言 tag 命中
			{"id": 222, "tag_ids": []int{1, 2}},  // 未命中
			{"id": 333, "tag_ids": []int{29963}}, // 语言 tag 命中
		},
	}
	html := sveltekitHTML(t, "/api/v2/galleries?page=1", body)

	ids, err := parseIDsFromIndexV2(html, 0)
	if err != nil {
		t.Fatalf("parseIDsFromIndexV2 error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 111 || ids[1] != 333 {
		t.Fatalf("ids = %v, want [111 333]", ids)
	}
}

func TestParseIDsFromIndexV2_NoMatchingTag(t *testing.T) {
	body := map[string]any{"result": []map[string]any{{"id": 111, "tag_ids": []int{1}}}}
	html := sveltekitHTML(t, "/api/v2/galleries?page=1", body)

	ids, err := parseIDsFromIndexV2(html, 0)
	if err != nil {
		t.Fatalf("parseIDsFromIndexV2 error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids = %v, want empty", ids)
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
	const cid = 640503
	body := map[string]any{
		"id":       cid,
		"media_id": "99999",
		"title":    map[string]any{"english": "E"},
		"images": map[string]any{
			"pages": []map[string]any{{"t": "j", "w": 1, "h": 1}},
		},
		"num_pages": 1,
	}
	html := sveltekitHTML(t, "/api/v2/galleries/640503", body)

	info, err := parseComicPageV2FromHTML(html, cid)
	if err != nil {
		t.Fatalf("parseComicPageV2FromHTML error: %v", err)
	}
	if info["media_id"] == nil || info["media_id"] == "" {
		t.Fatalf("media_id missing")
	}
	images, ok := info["images"].(map[string]any)
	if !ok {
		t.Fatalf("images not found")
	}
	pages, ok := images["pages"].([]any)
	if !ok || len(pages) == 0 {
		t.Fatalf("pages not found or empty")
	}

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

func TestParseV2DetailFromHTML_MissingCID(t *testing.T) {
	body := map[string]any{"id": 1, "media_id": "1", "images": map[string]any{}}
	html := sveltekitHTML(t, "/api/v2/galleries/1", body)

	if _, err := parseComicPageV2FromHTML(html, 999999); err == nil {
		t.Fatal("cid 不匹配时必须返回错误")
	}
}

func TestNormalizeV2ToV1(t *testing.T) {
	info := map[string]any{
		"id":       float64(640503),
		"media_id": "99999",
		"title":    map[string]any{"english": "E"},
		"images": map[string]any{
			"pages":     []any{map[string]any{"t": "j", "w": float64(1), "h": float64(1)}},
			"cover":     map[string]any{"t": "j", "w": float64(1), "h": float64(1)},
			"thumbnail": map[string]any{"t": "j", "w": float64(1), "h": float64(1)},
		},
		"num_pages": float64(1),
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
		for _, key := range []string{"t", "w", "h"} {
			if _, ok := m[key]; !ok {
				t.Fatalf("page %d missing %s", i, key)
			}
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

	// 归一化结果必须能映射为 api.ComicInfo 并生成合法的原图 URL。
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal normalized failed: %v", err)
	}
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
	if !strings.HasSuffix(url, ".jpg") && !strings.HasSuffix(url, ".png") &&
		!strings.HasSuffix(url, ".webp") && !strings.HasSuffix(url, ".gif") {
		t.Fatalf("origin url extension invalid: %s", url)
	}
}

func TestApplyScraperProxy(t *testing.T) {
	t.Cleanup(func() { SetScraperProxyPrefix("") })

	t.Run("未配置代理时原样返回", func(t *testing.T) {
		SetScraperProxyPrefix("")
		if got := applyScraperProxy("https://nhentai.net/g/1/"); got != "https://nhentai.net/g/1/" {
			t.Errorf("got %q, want 原样返回", got)
		}
	})

	t.Run("配置代理时加前缀", func(t *testing.T) {
		SetScraperProxyPrefix("http://proxy.example:18081/")
		if got := applyScraperProxy("https://nhentai.net/g/1/"); got != "http://proxy.example:18081/nhentai.net/g/1/" {
			t.Errorf("got %q, want 加前缀后的 URL", got)
		}
	})

	t.Run("已指向该代理时不重复加前缀", func(t *testing.T) {
		SetScraperProxyPrefix("http://proxy.example:18081")
		if got := applyScraperProxy("http://proxy.example:18081/nhentai.net/g/1/"); got != "http://proxy.example:18081/nhentai.net/g/1/" {
			t.Errorf("got %q, want 不重复加前缀", got)
		}
	})
}
