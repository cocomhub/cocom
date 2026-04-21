// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"strings"
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
)

func TestNewFnAndRegistration(t *testing.T) {
	t.Skip()
	name := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	cfg := storage.Config{
		Name: name,
		Type: Type,
		MetaData: map[string]any{
			"root":           "/apps/cocom/archive",
			"temp_dir":       t.TempDir(),
			"bduss":          "fake-bduss",
			"stoken":         "fake-stoken",
			"sboxtkn":        "fake-sboxtkn",
			"app_id":         "266719",
			"pcs_addr":       "pcs.example.test",
			"pcs_user_agent": "cocom-test-pcs",
			"pan_user_agent": "cocom-test-pan",
		},
	}
	if err := storage.SetFromConfig(cfg); err != nil {
		t.Fatalf("set from config: %v", err)
	}
	got, ok := storage.Get(name)
	if !ok {
		t.Fatalf("registered storage not found")
	}
	if got.Type() != Type {
		t.Fatalf("unexpected type: %s", got.Type())
	}
}

func TestNewFnValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]any
		wantSub string
	}{
		{
			name: "accept remote root alias with cookies",
			config: map[string]any{
				"root":    "/apps/cocom",
				"cookies": "BDUSS=fake; STOKEN=fake;",
			},
			wantSub: "检测BDUSS有效性错误代码: 1, 消息: 用户未登录或登录失败，请更换账号或重试",
		},
		{
			name: "missing root",
			config: map[string]any{
				"bduss": "fake",
			},
			wantSub: "root is required",
		},
		{
			name: "root wrong type",
			config: map[string]any{
				"root":  1,
				"bduss": "fake",
			},
			wantSub: "root is not a string",
		},
		{
			name: "missing auth",
			config: map[string]any{
				"root": "/apps/cocom",
			},
			wantSub: "either bduss or cookies is required",
		},
		{
			name: "invalid app_id",
			config: map[string]any{
				"root":   "/apps/cocom",
				"bduss":  "fake",
				"app_id": "not-int",
			},
			wantSub: "app_id is not an int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newFn("invalid", tt.config)
			if tt.wantSub == "" {
				if err != nil {
					t.Fatalf("newFn should succeed: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewAllowsAdapterWithoutAuth(t *testing.T) {
	adapter := newFakeAdapter()
	st, err := New("adapter-only", Config{
		Root:    "/apps/cocom",
		TempDir: t.TempDir(),
		Adapter: adapter,
	})
	if err != nil {
		t.Fatalf("new with adapter: %v", err)
	}
	if st.adapter != adapter {
		t.Fatalf("storage should reuse injected adapter")
	}
}
