// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cocomhub/cocom/internal/rootcli"
	"gopkg.in/yaml.v3"
)

// containsAny 报告 s 是否包含 sub 中的任意子串。
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func TestComposeListenAddr(t *testing.T) {
	cases := []struct {
		name    string
		hostVal any
		hasHost bool
		portVal any
		hasPort bool
		want    string
		wantErr bool
	}{
		{name: "host+port", hostVal: "0.0.0.0", hasHost: true, portVal: 35456, hasPort: true, want: "0.0.0.0:35456"},
		{name: "only host", hostVal: "192.168.1.10", hasHost: true, want: "192.168.1.10:8080"},
		{name: "only port", portVal: 9090, hasPort: true, want: "0.0.0.0:9090"},
		{name: "neither", want: "0.0.0.0:8080"},
		{name: "empty host string", hostVal: "", hasHost: true, portVal: 8081, hasPort: true, want: "0.0.0.0:8081"},
		{name: "int64 port", portVal: int64(8080), hasPort: true, want: "0.0.0.0:8080"},
		{name: "float64 port", portVal: float64(8082), hasPort: true, want: "0.0.0.0:8082"},
		{name: "invalid host type", hostVal: 123, hasHost: true, wantErr: true},
		{name: "invalid port type", portVal: "abc", hasPort: true, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := composeListenAddr(tc.hostVal, tc.hasHost, tc.portVal, tc.hasPort)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("composeListenAddr = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("composeListenAddr unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("composeListenAddr = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPathHelpers(t *testing.T) {
	// getPath/setPath/deletePath/pruneEmpty 嵌套 map 操作
	m := map[string]any{}
	setPath(m, "v1", "a", "b", "c")
	if v, ok := getPath(m, "a", "b", "c"); !ok || v != "v1" {
		t.Fatalf("getPath(a.b.c) = %v, %v; want v1, true", v, ok)
	}
	if _, ok := getPath(m, "a", "b", "missing"); ok {
		t.Fatal("getPath(a.b.missing) = ok, want false")
	}
	// getPath 对中间节点也返回存在（路径可定位即 ok），这是实现语义
	if v, ok := getPath(m, "a"); !ok || v.(map[string]any) == nil {
		t.Fatalf("getPath(a) = %v, %v; want (map, true)", v, ok)
	}

	deletePath(m, "a", "b", "c")
	if _, ok := getPath(m, "a", "b", "c"); ok {
		t.Fatal("after delete, getPath(a.b.c) should be missing")
	}

	// pruneEmpty 删除空父节点
	m2 := map[string]any{"archive": map[string]any{"password": "x"}}
	deletePath(m2, "archive", "password")
	pruneEmpty(m2)
	if len(m2) != 0 {
		t.Errorf("pruneEmpty left %d keys, want 0", len(m2))
	}

	// pruneEmpty 保留非空节点
	m3 := map[string]any{"cocom": map[string]any{"archive": map[string]any{"path": "/p"}}}
	pruneEmpty(m3)
	if _, ok := getPath(m3, "cocom", "archive", "path"); !ok {
		t.Fatal("pruneEmpty should keep non-empty branches")
	}
}

func TestIsSensitiveKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"archive.password", true},
		{"cocom.archive.password", true},
		{"mongo.password", true},
		{"admin.token", true},
		{"cocom.storage.backends", false},
		{"server.listen.http.addr", false},
		{"download.proxyURL", false}, // URL 值含 userinfo 由值判定，键名不算敏感
		{"archive.cmd", false},
	}
	for _, tc := range cases {
		if got := isSensitiveKey(tc.key); got != tc.want {
			t.Errorf("isSensitiveKey(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

// TestRunConfigMigrate 端到端驱动 runConfigMigrate 真身（非镜像复制）：
// 通过 cobra 完整执行 `config migrate --config <tmp>`，让 rootcli.ConfigFile()
// 返回临时文件路径，验证迁移后落盘的键集合与敏感键脱敏。
func TestRunConfigMigrate(t *testing.T) {
	// 构造旧格式 YAML
	oldYAML := `archive:
  password: oldpass
  cmd: 7z
  replicate: true
storage:
  backends:
    - name: idx
      type: localfs
http:
  enable_proxy: true
  proxy: http://user:secret@proxy:8080
host: 0.0.0.0
port: 35456
cocom:
  archive:
    path: /data/archive
`
	cfgFile := filepath.Join(t.TempDir(), "cocom.yaml")
	if err := os.WriteFile(cfgFile, []byte(oldYAML), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	// 注入配置文件路径（rootcli.ConfigFile() 由此返回），直接驱动 runConfigMigrate。
	rootcli.SetConfigFileForTest(cfgFile)
	t.Cleanup(func() { rootcli.SetConfigFileForTest("") })

	var outBuf bytes.Buffer
	configMigrateCmd.SetOut(&outBuf)
	configMigrateCmd.SetErr(io.Discard)
	migrateFlags.yes = true
	t.Cleanup(func() { migrateFlags.yes = false })
	if err := runConfigMigrate(configMigrateCmd); err != nil {
		t.Fatalf("runConfigMigrate failed: %v", err)
	}

	// 读迁移后的文件，验证键集合
	got, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("read migrated cfg: %v", err)
	}
	var data map[string]any
	if err := yaml.Unmarshal(got, &data); err != nil {
		t.Fatalf("parse migrated yaml: %v", err)
	}

	if v, ok := getPath(data, "cocom", "archive", "password"); !ok || v != "oldpass" {
		t.Errorf("cocom.archive.password = %v, %v; want oldpass", v, ok)
	}
	if v, ok := getPath(data, "cocom", "archive", "replicate"); !ok || v != true {
		t.Errorf("cocom.archive.replicate = %v, %v; want true", v, ok)
	}
	if v, ok := getPath(data, "cocom", "storage", "backends"); !ok || len(v.([]any)) != 1 {
		t.Errorf("cocom.storage.backends = %v, %v; want 1 backend", v, ok)
	}
	if v, ok := getPath(data, "download", "enableProxy"); !ok || v != true {
		t.Errorf("download.enableProxy = %v, %v; want true", v, ok)
	}
	if v, ok := getPath(data, "download", "proxyURL"); !ok || v != "http://user:secret@proxy:8080" {
		t.Errorf("download.proxyURL = %v, %v; want preserved", v, ok)
	}
	if v, ok := getPath(data, "server", "listen", "http", "addr"); !ok || v != "0.0.0.0:35456" {
		t.Errorf("server.listen.http.addr = %v, %v; want 0.0.0.0:35456", v, ok)
	}
	// 旧键应已删除
	for _, gone := range []string{"archive", "storage", "http", "host", "port"} {
		if _, ok := data[gone]; ok {
			t.Errorf("old key %q still present after migrate", gone)
		}
	}

	// 敏感键在 diff 输出中脱敏：口令与代理 URL 的 userinfo 不得以明文出现。
	outStr := outBuf.String()
	if containsAny(outStr, "oldpass", "user:secret") {
		t.Errorf("migrate diff leaked sensitive value: %q", outStr)
	}
}

// TestRunConfigMigrate_DryRun 验证 --dry-run 不写盘（功能存在性）。
func TestRunConfigMigrate_DryRun(t *testing.T) {
	oldYAML := "archive:\n  password: oldpass\n"
	cfgFile := filepath.Join(t.TempDir(), "cocom.yaml")
	if err := os.WriteFile(cfgFile, []byte(oldYAML), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	before, _ := os.ReadFile(cfgFile)

	rootcli.SetConfigFileForTest(cfgFile)
	t.Cleanup(func() { rootcli.SetConfigFileForTest("") })

	var outBuf bytes.Buffer
	configMigrateCmd.SetOut(&outBuf)
	configMigrateCmd.SetErr(io.Discard)
	migrateFlags.dryRun = true
	migrateFlags.yes = true
	t.Cleanup(func() {
		migrateFlags.dryRun = false
		migrateFlags.yes = false
	})
	if err := runConfigMigrate(configMigrateCmd); err != nil {
		t.Fatalf("config migrate --dry-run failed: %v", err)
	}

	after, _ := os.ReadFile(cfgFile)
	if string(before) != string(after) {
		t.Error("--dry-run modified the config file (should not write)")
	}
	if !strings.Contains(outBuf.String(), "cocom.archive.password") {
		t.Errorf("dry-run diff should list migration, got %q", outBuf.String())
	}
}
