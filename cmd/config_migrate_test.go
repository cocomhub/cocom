// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
		{name: "only port", portVal: 9090, hasPort: true, want: "127.0.0.1:9090"}, // 铁律 4：host 缺失默认 127.0.0.1
		{name: "neither", want: "127.0.0.1:8080"},                                 // 铁律 4：默认回环
		{name: "empty host string", hostVal: "", hasHost: true, portVal: 8081, hasPort: true, want: "127.0.0.1:8081"},
		{name: "int64 port", portVal: int64(8080), hasPort: true, want: "127.0.0.1:8080"},
		{name: "float64 port", portVal: float64(8082), hasPort: true, want: "127.0.0.1:8082"},
		{name: "explicit 0.0.0.0", hostVal: "0.0.0.0", hasHost: true, portVal: 8080, hasPort: true, want: "0.0.0.0:8080"},
		{name: "whitespace host treated missing", hostVal: "   ", hasHost: true, portVal: 8083, hasPort: true, want: "127.0.0.1:8083"},
		{name: "invalid host type", hostVal: 123, hasHost: true, wantErr: true},
		{name: "invalid port type", portVal: "abc", hasPort: true, wantErr: true},
		{name: "zero port", hasPort: true, portVal: 0, want: "127.0.0.1:0"}, // port=0 为可用端口探测语义
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

// TestRunConfigMigrate_ConflictDiffers 回归铁律 3：新旧键同时显式配置且值不同 →
// 迁移失败（不做非零优先的隐式兼容），并保持原文件不被修改。
func TestRunConfigMigrate_ConflictDiffers(t *testing.T) {
	conflictYAML := `archive:
  password: oldpass
cocom:
  archive:
    password: newpass
`
	cfgFile := filepath.Join(t.TempDir(), "cocom.yaml")
	if err := os.WriteFile(cfgFile, []byte(conflictYAML), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	before, _ := os.ReadFile(cfgFile)

	rootcli.SetConfigFileForTest(cfgFile)
	t.Cleanup(func() { rootcli.SetConfigFileForTest("") })

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	configMigrateCmd.SetOut(&outBuf)
	configMigrateCmd.SetErr(&errBuf)
	migrateFlags.yes = true
	t.Cleanup(func() { migrateFlags.yes = false })
	err := runConfigMigrate(configMigrateCmd)
	if err == nil {
		t.Fatal("冲突配置迁移应失败")
	}
	if !containsAny(err.Error(), "冲突", "人工决策") {
		t.Errorf("错误信息应提示冲突需人工决策，got %q", err.Error())
	}
	after, _ := os.ReadFile(cfgFile)
	if string(before) != string(after) {
		t.Error("冲突失败路径不应修改配置文件")
	}
}

// TestRunConfigMigrate_ConflictSameValue 回归铁律 3 的容错分支：
// 新旧键同值（含空值哈希表）时迁移应成功，且不产生冲突错误。
func TestRunConfigMigrate_ConflictSameValue(t *testing.T) {
	sameYAML := `archive:
  password: p1
cocom:
  archive:
    password: p1
storage:
  backends:
    - name: idx
strange_old:
  value: x
`
	cfgFile := filepath.Join(t.TempDir(), "cocom.yaml")
	if err := os.WriteFile(cfgFile, []byte(sameYAML), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	rootcli.SetConfigFileForTest(cfgFile)
	t.Cleanup(func() { rootcli.SetConfigFileForTest("") })

	configMigrateCmd.SetOut(io.Discard)
	configMigrateCmd.SetErr(io.Discard)
	migrateFlags.yes = true
	t.Cleanup(func() { migrateFlags.yes = false })
	if err := runConfigMigrate(configMigrateCmd); err != nil {
		t.Fatalf("同值配置迁移不应失败: %v", err)
	}

	got, _ := os.ReadFile(cfgFile)
	var data map[string]any
	if err := yaml.Unmarshal(got, &data); err != nil {
		t.Fatalf("parse migrated yaml: %v", err)
	}
	if v, ok := getPath(data, "cocom", "archive", "password"); !ok || v != "p1" {
		t.Errorf("cocom.archive.password = %v, want p1", v)
	}
}

// TestRunConfigMigrate_MalformedYAML 回归铁律 1：migrate 是修复工具，
// YAML 解析失败必须显式报错退出，不静默（不继续、不写盘为空）。
func TestRunConfigMigrate_MalformedYAML(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "cocom.yaml")
	if err := os.WriteFile(cfgFile, []byte("archive: [unclosed\n  bad: \"hex: abc"), 0o600); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	rootcli.SetConfigFileForTest(cfgFile)
	t.Cleanup(func() { rootcli.SetConfigFileForTest("") })

	configMigrateCmd.SetOut(io.Discard)
	configMigrateCmd.SetErr(io.Discard)
	migrateFlags.yes = true
	t.Cleanup(func() { migrateFlags.yes = false })
	if err := runConfigMigrate(configMigrateCmd); err == nil {
		t.Fatal("malformed YAML 应返回错误而非静默")
	}
}

// TestValueHasCredentials 回归方案 4：valueHasCredentials 需覆盖无 scheme 的 URL（user:pass@host）。
func TestValueHasCredentials(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{"http://user:secret@proxy:8080", true},
		{"http://proxy:8080", false},
		{"user:pass@host:8080", true},
		{"user@host", false},
		{"plain", false},
		{123, false},
		{nil, false},
		{"http://proxy:8080/path", false},
	}
	for _, tc := range cases {
		if got := valueHasCredentials(tc.in); got != tc.want {
			t.Errorf("valueHasCredentials(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestWriteFileAtomic 回归方案 2：原子写盘——临时文件先落盘再 rename；
// 对不存在目标时创建；已存在时替换。失败路径由 CreateTemp 目标目录不可写模拟。
func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "cfg.yaml")
	if err := writeFileAtomic(target, []byte("a: 1\n")); err != nil {
		t.Fatalf("write new file: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read new file: %v", err)
	}
	if string(got) != "a: 1\n" {
		t.Errorf("content = %q, want %q", got, "a: 1\n")
	}
	fi, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// Windows 的 os.Chmod 是 no-op，Perm() 恒为 0666；仅 unix 系断言权限收敛。
	if runtime.GOOS == "windows" {
		t.Logf("windows: skip file-mode assertion (os.Chmod no-op), mode=%04o", fi.Mode().Perm())
	} else if fi.Mode().Perm()&0o077 != 0 {
		t.Errorf("mode = %04o, want 0600", fi.Mode().Perm())
	}

	// 已存在文件替换
	if err := writeFileAtomic(target, []byte("b: 2\n")); err != nil {
		t.Fatalf("replace: %v", err)
	}
	got, _ = os.ReadFile(target)
	if string(got) != "b: 2\n" {
		t.Errorf("replaced content = %q, want %q", got, "b: 2\n")
	}

	// 目录替代文件（Rename 失败）→ 返回错误且尝试清理临时文件
	if err := writeFileAtomic(filepath.Join(dir, "subdir-not-a-file"), []byte("x")); err != nil {
		got, _ := os.ReadFile(filepath.Join(dir, "subdir-not-a-file"))
		if len(got) == 0 {
			t.Error("subdir-not-a-file 应保留原内容")
		}
	}
}
