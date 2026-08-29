// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"testing"
)

func TestNewConfig_ReturnsNonNil(t *testing.T) {
	cfg := NewConfig()
	if cfg == nil {
		t.Fatal("NewConfig() returned nil")
	}
}

func TestNewConfig_ZeroValues(t *testing.T) {
	cfg := NewConfig()
	if cfg.DownloadDir != "" {
		t.Errorf("DownloadDir = %q, want empty", cfg.DownloadDir)
	}
	if cfg.MaxRunning != 0 {
		t.Errorf("MaxRunning = %d, want 0", cfg.MaxRunning)
	}
	if cfg.EnableProxy {
		t.Error("EnableProxy = true, want false")
	}
	if cfg.ProxyURL != "" {
		t.Errorf("ProxyURL = %q, want empty", cfg.ProxyURL)
	}
}

func TestNewInitConfig_SetsFields(t *testing.T) {
	cfg := NewInitConfig(Config{
		MaxRunning:  5,
		DownloadDir: "/tmp/dl",
		EnableProxy: true,
		ProxyURL:    "http://proxy:8080",
	})
	if cfg == nil {
		t.Fatal("NewInitConfig() returned nil")
	}
	if cfg.MaxRunning != 5 {
		t.Errorf("MaxRunning = %d, want 5", cfg.MaxRunning)
	}
	if cfg.DownloadDir != "/tmp/dl" {
		t.Errorf("DownloadDir = %q, want /tmp/dl", cfg.DownloadDir)
	}
	if !cfg.EnableProxy {
		t.Error("EnableProxy = false, want true")
	}
	if cfg.ProxyURL != "http://proxy:8080" {
		t.Errorf("ProxyURL = %q, want http://proxy:8080", cfg.ProxyURL)
	}
}

func TestNewConfig_ChainSetters(t *testing.T) {
	cfg := NewConfig().
		SetDownloadDir("/tmp/dl").
		SetMaxRunning(8).
		SetEnableProxy(true).
		SetProxyURL("http://proxy:3128")

	if cfg.DownloadDir != "/tmp/dl" {
		t.Errorf("DownloadDir = %q, want /tmp/dl", cfg.DownloadDir)
	}
	if cfg.MaxRunning != 8 {
		t.Errorf("MaxRunning = %d, want 8", cfg.MaxRunning)
	}
	if !cfg.EnableProxy {
		t.Error("EnableProxy = false, want true")
	}
	if cfg.ProxyURL != "http://proxy:3128" {
		t.Errorf("ProxyURL = %q, want http://proxy:3128", cfg.ProxyURL)
	}
}

func TestNewConfig_InitDefaults(t *testing.T) {
	cfg := NewConfig()
	cfg.Init()
	if cfg.DownloadDir != "./Downloads" {
		t.Errorf("DownloadDir = %q, want ./Downloads", cfg.DownloadDir)
	}
	if cfg.MaxRunning != 3 {
		t.Errorf("MaxRunning = %d, want 3", cfg.MaxRunning)
	}
}

func TestNewConfig_InitPreservesValues(t *testing.T) {
	cfg := NewConfig().
		SetDownloadDir("/custom/path").
		SetMaxRunning(20)
	cfg.Init()
	if cfg.DownloadDir != "/custom/path" {
		t.Errorf("DownloadDir = %q, want /custom/path", cfg.DownloadDir)
	}
	if cfg.MaxRunning != 20 {
		t.Errorf("MaxRunning = %d, want 20", cfg.MaxRunning)
	}
}

func TestReplaceDownloader_Restore(t *testing.T) {
	old := DefaultDownloader
	restore := ReplaceDownloader(NewDownloader(NewConfig()))
	if DefaultDownloader == old {
		t.Error("DefaultDownloader was not replaced")
	}
	restore()
	if DefaultDownloader != old {
		t.Error("DefaultDownloader was not restored")
	}
}

func TestInit_SetsDefaultDownloader(t *testing.T) {
	old := DefaultDownloader
	Init(Config{
		MaxRunning:  7,
		DownloadDir: "/test/dl",
	})
	if DefaultDownloader == old {
		t.Error("DefaultDownloader was not updated after Init()")
	}
	// Restore for other tests
	ReplaceDownloader(old)
}

func TestDoBatch_ValidWorkers(t *testing.T) {
	// DoBatch with 0 workers should not panic (it delegates to DefaultDownloader)
	ch, err := DoBatch(1)
	if err != nil {
		t.Fatalf("DoBatch(1) unexpected error: %v", err)
	}
	if ch == nil {
		t.Error("DoBatch(1) returned nil channel")
	}
}

// TestDownloaderConfig_Init_NegativeMaxRunning 回归 Batch C：负并发数不能
// 触发 make(chan, -1) panic（Init 必须兜底为正数）。
func TestDownloaderConfig_Init_NegativeMaxRunning(t *testing.T) {
	cfg := &DownloaderConfig{MaxRunning: -5}
	cfg.Init()
	if cfg.MaxRunning != 3 {
		t.Errorf("MaxRunning after Init = %d, want 3 (negative fallback)", cfg.MaxRunning)
	}

	// 0 也兜底为 3（原逻辑只判 ==0，负数会 panic）。
	cfg0 := &DownloaderConfig{MaxRunning: 0}
	cfg0.Init()
	if cfg0.MaxRunning != 3 {
		t.Errorf("MaxRunning after Init = %d, want 3 (zero fallback)", cfg0.MaxRunning)
	}

	// 构造 downloader 不应 panic。
	_ = NewDownloader(NewConfig().SetMaxRunning(-1))
}
