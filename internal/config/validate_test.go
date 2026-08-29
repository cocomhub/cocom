// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/cocomhub/cocom/pkg/download"
	"github.com/cocomhub/cocom/pkg/mongowrap"
)

// TestValidate_AllowOrigins 校验 CORS allow_origins 启动期校验：
// 整体 * 合法；合法 http/https 列表合法；含 * 中缀 / 非 http(s) scheme / 空元素 -> error。
func TestValidate_AllowOrigins(t *testing.T) {
	cases := []struct {
		name    string
		origins string
		wantErr bool
	}{
		{name: "整体星号合法", origins: "*", wantErr: false},
		{name: "星号带空格合法", origins: " * ", wantErr: false},
		{name: "多个合法来源", origins: "https://a.example.com, http://b.example.com", wantErr: false},
		{name: "合法来源带尾路径", origins: "https://a.example.com/foo", wantErr: false},
		{name: "含中缀星号", origins: "https://*.example.com", wantErr: true},
		{name: "中缀星号混合列表", origins: "https://a.com,*", wantErr: true},
		{name: "无 scheme 的 host", origins: "example.com", wantErr: true},
		{name: "ftp scheme", origins: "ftp://example.com", wantErr: true},
		{name: "空元素", origins: "https://a.com,,https://b.com", wantErr: true},
		{name: "整体为空串", origins: "", wantErr: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := minimalValidConfig()
			cfg.Server.CORS.AllowOrigins = tt.origins
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("validateAllowOrigins(%q) = nil, want error", tt.origins)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate(%q) = %v, want nil", tt.origins, err)
			}
		})
	}
}

// minimalValidConfig 构造一个所有非空校验字段合法的 Config，
// 让 Tests 只针对 allow_origins 的差异产生结果。
func minimalValidConfig() *Config {
	return &Config{
		Archive: Archive{Manager: ArchiveManager{Index: ArchiveIndex{Type: "memory"}}},
		Cocom: Cocom{
			Archive: CocomArchive{
				Algorithm: ArchiveAlgo{Single: ArchiveAlgoConcurrency{Concurrency: 4}, Double: ArchiveAlgoConcurrency{Concurrency: 4}},
			},
			Cache: CocomCache{CleanInterval: "1m", EvictionInterval: "10m"},
		},
		Server:   Server{Listen: Listen{HTTP: ListenAddr{Addr: "127.0.0.1:8080"}}, ShutdownTimeout: "5s", CORS: CORS{AllowOrigins: "*"}, RateLimit: RateLimit{RPS: 10}},
		Comic:    Comic{Verify: ComicVerify{Concurrent: 10}},
		Mongo:    mongowrap.Config{Host: "localhost:27017", Database: "cocom", User: "cocom"},
		Download: download.Config{MaxRunning: 10},
	}
}
