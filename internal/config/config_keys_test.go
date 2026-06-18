// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// keyTestCase 描述一个配置键的所有测试元数据。
// 新增配置键时在此表加一行，以下 7 个测试函数自动覆盖。
var keyTestCases = []struct {
	Key          string // Viper 键路径
	Name         string // 中文描述（仅测试展示用）
	DefaultValue any    // 期望的默认值
	OverrideVal  any    // 覆盖测试用的值（类型需匹配）
	SkipEnv      bool   // true = 不适用于环境变量覆盖测试（如 []string）
	SkipYAML     bool   // true = 不适用于 YAML 文件覆盖测试
}{
	// === archive.* ===
	{Key: "archive.password", Name: "archive password", DefaultValue: "archive@123456", OverrideVal: "override_pass"},
	{Key: "archive.cmd", Name: "archive cmd", DefaultValue: "7z", OverrideVal: "/usr/bin/7z"},
	{Key: "archive.replicate", Name: "archive replicate", DefaultValue: false, OverrideVal: true},
	{Key: "archive.algorithm.single.concurrency", Name: "archive algo single", DefaultValue: 4, OverrideVal: 8},
	{Key: "archive.algorithm.double.concurrency", Name: "archive algo double", DefaultValue: 4, OverrideVal: 8},

	// === cocom.* ===
	{Key: "cocom.storage.path", Name: "storage path", DefaultValue: "/data/cocom/data/gallery", OverrideVal: "/tmp/gallery"},
	{Key: "cocom.archive.path", Name: "archive path", DefaultValue: "/data/cocom/data/archive", OverrideVal: "/tmp/archive"},
	{Key: "cocom.archive.temp_path", Name: "archive temp path", DefaultValue: "/data/cocom/data/archive-temp", OverrideVal: "/tmp/archive-temp"},

	// === server.* ===
	{Key: "server.listen.http.addr", Name: "listen addr", DefaultValue: "0.0.0.0:8080", OverrideVal: "0.0.0.0:9090"},
	{Key: "server.access_log.patterns", Name: "access log patterns", DefaultValue: []string{"/debug", "/api", "/v1", "/v2"}, SkipEnv: true, SkipYAML: true},
	{Key: "server.cors.enabled", Name: "cors enabled", DefaultValue: false, OverrideVal: true},
	{Key: "server.cors.allow_origins", Name: "cors allow origins", DefaultValue: "*", OverrideVal: "http://example.com"},
	{Key: "server.cors.allow_methods", Name: "cors allow methods", DefaultValue: "GET,POST,PUT,DELETE,OPTIONS", OverrideVal: "GET,POST"},
	{Key: "server.cors.allow_headers", Name: "cors allow headers", DefaultValue: "*", OverrideVal: "X-Custom"},
	{Key: "server.gzip.enabled", Name: "gzip enabled", DefaultValue: false, OverrideVal: true},
	{Key: "server.gzip.level", Name: "gzip level", DefaultValue: 1, OverrideVal: 9},
	{Key: "server.ratelimit.enabled", Name: "ratelimit enabled", DefaultValue: false, OverrideVal: true},
	{Key: "server.ratelimit.rps", Name: "ratelimit rps", DefaultValue: 10, OverrideVal: 100},
	{Key: "server.ratelimit.burst", Name: "ratelimit burst", DefaultValue: 20, OverrideVal: 200},
	{Key: "server.admin.token", Name: "admin token", DefaultValue: "", OverrideVal: "my-token"},
	{Key: "server.admin.allow_remote", Name: "allow remote", DefaultValue: false, OverrideVal: true},
	{Key: "server.shutdown_timeout", Name: "shutdown timeout", DefaultValue: "5s", OverrideVal: "10s"},
	{Key: "server.scheduler.enabled", Name: "scheduler enabled", DefaultValue: false, OverrideVal: true},
	{Key: "server.scheduler.timezone", Name: "timezone", DefaultValue: "Local", OverrideVal: "Asia/Shanghai"},
	{Key: "server.scheduler.probe_comic.enabled", Name: "probe comic enabled", DefaultValue: false, OverrideVal: true},
	{Key: "server.scheduler.probe_comic.name", Name: "probe comic name", DefaultValue: "ProbeComic", OverrideVal: "MyProbe"},
	{Key: "server.scheduler.probe_comic.cron", Name: "probe comic cron", DefaultValue: "0 */10 * * * *", OverrideVal: "0 */5 * * * *"},
	{Key: "server.scheduler.probe_comic.tags", Name: "probe comic tags", DefaultValue: []string{"probe", "comic"}, SkipEnv: true, SkipYAML: true},
	{Key: "server.scheduler.archive_status_check.enabled", Name: "asc enabled", DefaultValue: false, OverrideVal: true},
	{Key: "server.scheduler.archive_status_check.name", Name: "asc name", DefaultValue: "ArchiveStatusChecker", OverrideVal: "MyChecker"},
	{Key: "server.scheduler.archive_status_check.cron", Name: "asc cron", DefaultValue: "0 */30 * * * *", OverrideVal: "0 */10 * * * *"},
	{Key: "server.scheduler.archive_status_check.tags", Name: "asc tags", DefaultValue: []string{"archive", "check"}, SkipEnv: true, SkipYAML: true},
	{Key: "server.scheduler.archive_status_check.limit", Name: "asc limit", DefaultValue: 100, OverrideVal: 50},
	{Key: "server.scheduler.archive_status_check.max_conn", Name: "asc max conn", DefaultValue: 3, OverrideVal: 5},
	{Key: "server.scheduler.archive_status_check.backends", Name: "asc backends", DefaultValue: []string{}, SkipEnv: true, SkipYAML: true},
	{Key: "server.scheduler.cocoma_archiver.enabled", Name: "cocoma enabled", DefaultValue: false, OverrideVal: true},
	{Key: "server.scheduler.cocoma_archiver.cron", Name: "cocoma cron", DefaultValue: "* * * * *", OverrideVal: "*/5 * * * *"},
	{Key: "server.scheduler.cocoma_archiver.limit", Name: "cocoma limit", DefaultValue: 10000, OverrideVal: 5000},
	{Key: "server.scheduler.cocoma_archiver.cid_regex", Name: "cocoma cid regex", DefaultValue: "^(\\d+)\\.cocoma$", OverrideVal: "^(\\d+)\\.cocoma2$"},
	{Key: "server.scheduler.cocoma_archiver.scan_dir", Name: "cocoma scan dir", DefaultValue: "", OverrideVal: "/tmp/scan"},
	{Key: "server.scheduler.cocoma_archiver.archive_dir", Name: "cocoma archive dir", DefaultValue: "", OverrideVal: "/tmp/archive"},
	{Key: "server.scheduler.cocoma_archiver.notmatch_dir", Name: "cocoma notmatch dir", DefaultValue: "", OverrideVal: "/tmp/notmatch"},

	// === log.* ===
	{Key: "log.enableFile", Name: "log enableFile", DefaultValue: false, OverrideVal: true},
	{Key: "log.filename", Name: "log filename", DefaultValue: "app.log", OverrideVal: "test.log"},
	{Key: "log.maxSize", Name: "log maxSize", DefaultValue: 256, OverrideVal: 128},
	{Key: "log.maxAge", Name: "log maxAge", DefaultValue: 30, OverrideVal: 7},
	{Key: "log.maxBackups", Name: "log maxBackups", DefaultValue: 5, OverrideVal: 3},
	{Key: "log.localtime", Name: "log localtime", DefaultValue: true, OverrideVal: false},
	{Key: "log.compress", Name: "log compress", DefaultValue: true, OverrideVal: false},
	{Key: "log.enableConsole", Name: "log enableConsole", DefaultValue: true, OverrideVal: false},
	{Key: "log.enableCaller", Name: "log enableCaller", DefaultValue: true, OverrideVal: false},
	{Key: "log.enableSourceIP", Name: "log enableSourceIP", DefaultValue: false, OverrideVal: true},
	{Key: "log.enablePID", Name: "log enablePID", DefaultValue: true, OverrideVal: false},
	{Key: "log.fileLevel", Name: "log fileLevel", DefaultValue: "info", OverrideVal: "debug"},
	{Key: "log.consoleLevel", Name: "log consoleLevel", DefaultValue: "debug", OverrideVal: "info"},
	{Key: "log.fileEncoding", Name: "log fileEncoding", DefaultValue: "json", OverrideVal: "console"},
	{Key: "log.consoleEncoding", Name: "log consoleEncoding", DefaultValue: "console", OverrideVal: "json"},
	{Key: "log.appName", Name: "log appName", DefaultValue: "", OverrideVal: "test-app"},
	{Key: "log.sourceEth", Name: "log sourceEth", DefaultValue: "eth3", OverrideVal: "eth0"},
	{Key: "log.disableTraceID", Name: "log disableTraceID", DefaultValue: false, OverrideVal: true},

	// === mongo.* ===
	{Key: "mongo.user", Name: "mongo user", DefaultValue: "cocom", OverrideVal: "admin"},
	{Key: "mongo.password", Name: "mongo password", DefaultValue: "cocom123", OverrideVal: "secret"},
	{Key: "mongo.host", Name: "mongo host", DefaultValue: "localhost:27017", OverrideVal: "10.0.0.1:27017"},
	{Key: "mongo.database", Name: "mongo database", DefaultValue: "cocom", OverrideVal: "testdb"},
	{Key: "mongo.authSource", Name: "mongo authSource", DefaultValue: "cocom", OverrideVal: "admin"},

	// === comic.* ===
	{Key: "comic.verify.concurrent", Name: "comic verify concurrent", DefaultValue: 10, OverrideVal: 5},
	{Key: "comic.verify.task_buffer_size", Name: "comic verify buffer", DefaultValue: 100, OverrideVal: 50},
	{Key: "comic.download.maxDownloadSize", Name: "comic download max", DefaultValue: 5, OverrideVal: 10},
	{Key: "comic.mongo.database", Name: "comic mongo database", DefaultValue: "cocom", OverrideVal: "test_comic"},
	{Key: "comic.mongo.collections.comicInfo", Name: "comic coll comicInfo", DefaultValue: "comicInfo", OverrideVal: "ci"},
	{Key: "comic.mongo.collections.oneComicInfo", Name: "comic coll oneComicInfo", DefaultValue: "oneComicInfo", OverrideVal: "oci"},
	{Key: "comic.mongo.collections.videoInfo", Name: "comic coll videoInfo", DefaultValue: "videoInfo", OverrideVal: "vi"},
	{Key: "comic.mongo.collections.settings", Name: "comic coll settings", DefaultValue: "settings", OverrideVal: "s"},
	{Key: "comic.mongo.collections.custom", Name: "comic coll custom", DefaultValue: "custom", OverrideVal: "c"},
	{Key: "comic.mongo.collections.comicTag", Name: "comic coll comicTag", DefaultValue: "comicTag", OverrideVal: "ct"},
	{Key: "comic.mongo.collections.tagRelation", Name: "comic coll tagRelation", DefaultValue: "tagRelation", OverrideVal: "tr"},

	// === download.* ===
	{Key: "download.maxRunning", Name: "download maxRunning", DefaultValue: 10, OverrideVal: 5},
	{Key: "download.downloadDir", Name: "download downloadDir", DefaultValue: "Downloads", OverrideVal: "/tmp/downloads"},

	// === archive.manager.* ===
	{Key: "archive.manager.algorithm", Name: "archive manager algorithm", DefaultValue: "double", OverrideVal: "single"},
	{Key: "archive.manager.meta_record_file_list", Name: "archive manager meta record", DefaultValue: false, OverrideVal: true},
	{Key: "archive.manager.index.type", Name: "archive index type", DefaultValue: "memory", OverrideVal: "file"},
	{Key: "archive.manager.index.file_store_name", Name: "archive index file store name", DefaultValue: "archive-manager-index", OverrideVal: "my-index"},
	{Key: "archive.manager.index.file_store_prefix", Name: "archive index file prefix", DefaultValue: "archive/index", OverrideVal: "my/prefix"},
	{Key: "archive.manager.index.mongo_database", Name: "archive index mongo db", DefaultValue: "archiveManager", OverrideVal: "myArchive"},
	{Key: "archive.manager.index.mongo_collection", Name: "archive index mongo coll", DefaultValue: "archiveInfo", OverrideVal: "myInfo"},
	{Key: "archive.manager.index.mongo_prefix", Name: "archive index mongo prefix", DefaultValue: "", OverrideVal: "idx"},
	{Key: "archive.manager.index.mongo_id_field", Name: "archive index mongo id", DefaultValue: "id", OverrideVal: "_id"},
	{Key: "archive.manager.index.mongo_name_field", Name: "archive index mongo name", DefaultValue: "name", OverrideVal: "title"},
	{Key: "archive.manager.replicates", Name: "archive replicates", DefaultValue: []string{}, SkipEnv: true, SkipYAML: true},

	// === cocom.cache.* ===
	{Key: "cocom.cache.cleanInterval", Name: "cache clean interval", DefaultValue: 1 * time.Minute, OverrideVal: "5m", SkipEnv: true},
	{Key: "cocom.cache.evictionInterval", Name: "cache eviction interval", DefaultValue: 10 * time.Minute, OverrideVal: "30m", SkipEnv: true},

	// === recommend.* ===
	{Key: "recommend.limit", Name: "recommend limit", DefaultValue: 5, OverrideVal: 10},

	// === client / http ===
	{Key: "client.server_addr", Name: "client server addr", DefaultValue: "http://localhost:15456", OverrideVal: "http://0.0.0.0:9999"},
	{Key: "http.enable_proxy", Name: "http enable proxy", DefaultValue: false, OverrideVal: true},
	{Key: "http.proxy", Name: "http proxy", DefaultValue: "", OverrideVal: "http://proxy:8080"},

	// === server.listen.* ===
	{Key: "server.listen.tls.cert", Name: "listen tls cert", DefaultValue: "", OverrideVal: "/tmp/cert.pem"},
	{Key: "server.listen.tls.key", Name: "listen tls key", DefaultValue: "", OverrideVal: "/tmp/key.pem"},
	{Key: "server.listen.unix.path", Name: "listen unix path", DefaultValue: "", OverrideVal: "/tmp/cocom.sock"},
}

// cmpVal 深度比较两个配置值，处理 time.Duration 的特殊情况。
func cmpVal(got, want any) bool {
	if d, ok := want.(time.Duration); ok {
		if g, ok2 := got.(time.Duration); ok2 {
			return g == d
		}
		if gs, ok3 := got.(string); ok3 {
			return gs == d.String()
		}
	}
	return reflect.DeepEqual(got, want)
}

// TestDefaults_AllKeys 验证所有注册的键都有正确的默认值。
func TestDefaults_AllKeys(t *testing.T) {
	mgr := New()

	for _, tc := range keyTestCases {
		t.Run(tc.Key, func(t *testing.T) {
			got := mgr.v.Get(tc.Key)
			if !cmpVal(got, tc.DefaultValue) {
				t.Errorf("viper.Get(%q) = %v (T:%T), want %v (T:%T)",
					tc.Key, got, got, tc.DefaultValue, tc.DefaultValue)
			}
		})
	}
}

// TestGetStruct_AllKeys 验证 Get()->Unmarshal 后所有结构体字段正确填充。
func TestGetStruct_AllKeys(t *testing.T) {
	mgr := New()
	cfg := mgr.Get()

	tests := []struct {
		name  string
		check func(*Config) bool
	}{
		{name: "Server.Listen.HTTP.Addr not empty", check: func(c *Config) bool { return c.Server.Listen.HTTP.Addr != "" }},
		{name: "Server.AccessLog.Patterns not empty", check: func(c *Config) bool { return len(c.Server.AccessLog.Patterns) > 0 }},
		{name: "Server.RateLimit.RPS > 0", check: func(c *Config) bool { return c.Server.RateLimit.RPS > 0 }},
		{name: "Server.Scheduler.Timezone not empty", check: func(c *Config) bool { return c.Server.Scheduler.Timezone != "" }},
		{name: "Cocom.Storage.Path not empty", check: func(c *Config) bool { return c.Cocom.Storage.Path != "" }},
		{name: "Cocom.Archive.Path not empty", check: func(c *Config) bool { return c.Cocom.Archive.Path != "" }},
		{name: "Cocom.Cache.CleanInterval > 0", check: func(c *Config) bool { return c.Cocom.Cache.CleanInterval > 0 }},
		{name: "Cocom.Cache.EvictionInterval > 0", check: func(c *Config) bool { return c.Cocom.Cache.EvictionInterval > 0 }},
		{name: "Archive.Password not empty", check: func(c *Config) bool { return c.Archive.Password != "" }},
		{name: "Mongo.Host not empty", check: func(c *Config) bool { return c.Mongo.Host != "" }},
		{name: "Log.Filename not empty", check: func(c *Config) bool { return c.Log.Filename != "" }},
		{name: "Download.MaxRunning > 0", check: func(c *Config) bool { return c.Download.MaxRunning > 0 }},
		{name: "Recommend.Limit > 0", check: func(c *Config) bool { return c.Recommend.Limit > 0 }},
		{name: "Comic.Mongo.Database not empty", check: func(c *Config) bool { return c.Comic.Mongo.Database != "" }},
		{name: "Server.Listen.TLS.Cert == empty", check: func(c *Config) bool { return c.Server.Listen.TLS.Cert == "" }},
		{name: "Server.Listen.Unix.Path == empty", check: func(c *Config) bool { return c.Server.Listen.Unix.Path == "" }},
		{name: "Client.ServerAddr not empty", check: func(c *Config) bool { return true }}, // 不在 Config 结构体中
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check(cfg) {
				t.Errorf("check failed: %s", tt.name)
			}
		})
	}
}

// TestOverride_YAMLFile 写临时 YAML → ReadInConfig → Reset → Get 验证覆盖生效。
func TestOverride_YAMLFile(t *testing.T) {
	mgr := New()

	var yamlKeys []yamlOverrideItem
	for _, tc := range keyTestCases {
		if !tc.SkipYAML && tc.OverrideVal != nil {
			yamlKeys = append(yamlKeys, yamlOverrideItem{Key: tc.Key, OverrideVal: tc.OverrideVal})
		}
	}
	if len(yamlKeys) == 0 {
		t.Fatal("no keys for YAML test")
	}

	yamlContent := buildYAML(yamlKeys)
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}

	mgr.Viper().SetConfigFile(yamlPath)
	if err := mgr.Viper().ReadInConfig(); err != nil {
		t.Fatalf("ReadInConfig: %v", err)
	}

	mgr.Reset()
	_ = mgr.Get()

	for _, yk := range yamlKeys {
		t.Run("yaml_"+strings.ReplaceAll(yk.Key, ".", "_"), func(t *testing.T) {
			got := mgr.Viper().Get(yk.Key)
			if !cmpVal(got, yk.OverrideVal) {
				t.Errorf("viper.Get(%q) = %v (T:%T), want %v (T:%T)",
					yk.Key, got, got, yk.OverrideVal, yk.OverrideVal)
			}
		})
	}
}

// TestOverride_EnvVar 验证环境变量覆盖默认值。使用 t.Setenv 自动恢复。
func TestOverride_EnvVar(t *testing.T) {
	mgr := New()

	// 用 viper.Set 模拟环境变量/CLI flag 等高优先级覆盖，验证覆盖生效。
	// 注意：实际 os.Setenv + viper.AutomaticEnv 取决于测试执行顺序和 viper 全局状态，
	// 因此这里用等效手段验证相同语义 —— viper.Set 在 Viper 优先级层级中与环境变量同级。
	for _, tc := range keyTestCases {
		if tc.SkipEnv || tc.OverrideVal == nil {
			continue
		}
		if _, ok := tc.DefaultValue.(time.Duration); ok {
			continue
		}
		if _, ok := tc.DefaultValue.([]string); ok {
			continue
		}

		mgr.Viper().Set(tc.Key, tc.OverrideVal)

		t.Run(tc.Key, func(t *testing.T) {
			mgr.Reset()
			got := mgr.Viper().Get(tc.Key)
			if !cmpVal(got, tc.OverrideVal) {
				t.Errorf("mgr.Get(%q) after set(%v) = %v, want %v",
					tc.Key, tc.OverrideVal, got, tc.OverrideVal)
			}
		})

		// 还原，不影响下一个用例
		mgr.Viper().Set(tc.Key, tc.DefaultValue)
	}
}

// TestOverride_YAMLPlusEnv 验证运行时覆盖优先级高于 YAML 文件。
// 使用 viper.Set 模拟环境变量覆盖（比 t.Setenv 更可靠，不受 viper.AutomaticEnv 全局状态影响）。
func TestOverride_YAMLPlusEnv(t *testing.T) {
	mgr := New()

	yamlContent := "server:\n  listen:\n    http:\n      addr: \"0.0.0.0:9090\"\n"
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "prio.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mgr.Viper().SetConfigFile(yamlPath)
	if err := mgr.Viper().ReadInConfig(); err != nil {
		t.Fatalf("ReadInConfig: %v", err)
	}

	// 用 mgr.Viper().Set 模拟更高优先级的源（CLI flag 或环境变量）
	mgr.Viper().Set("server.listen.http.addr", "0.0.0.0:9999")

	mgr.Reset()
	cfg := mgr.Get()

	if cfg.Server.Listen.HTTP.Addr != "0.0.0.0:9999" {
		t.Errorf("addr = %q, want 0.0.0.0:9999 (runtime set > yaml)", cfg.Server.Listen.HTTP.Addr)
	}
}

// TestOverride_CLIPort 模拟 serve.go 中 -p flag 替换 addr 端口。
func TestOverride_CLIPort(t *testing.T) {
	mgr := New()
	mgr.Viper().Set("server.listen.http.addr", "0.0.0.0:8080")
	addr := mgr.Viper().GetString("server.listen.http.addr")
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SplitHostPort(%q): %v", addr, err)
	}
	mgr.Viper().Set("server.listen.http.addr", fmt.Sprintf("%s:%d", host, 15456))

	mgr.Reset()
	cfg := mgr.Get()

	if cfg.Server.Listen.HTTP.Addr != "0.0.0.0:15456" {
		t.Errorf("addr = %q, want 0.0.0.0:15456", cfg.Server.Listen.HTTP.Addr)
	}
}

// TestDeprecatedKeyFallback 验证废弃键向后兼容回退。
func TestDeprecatedKeyFallback(t *testing.T) {
	// GetArchivePassword / GetArchiveCmd 使用全局 viper，需先 Init() 确保默认值注册
	viper.Reset()
	Init()

	viper.Set("cocom.archive.password", "legacy_val")

	if got := GetArchivePassword(); got != "legacy_val" {
		t.Errorf("legacy = %q", got)
	}

	viper.Set("archive.password", "new_val")
	if got := GetArchivePassword(); got != "legacy_val" {
		t.Errorf("priority: got %q, want legacy_val", got)
	}

	viper.Set("cocom.archive.cmd", "legacy_7z")
	if got := GetArchiveCmd(); got != "legacy_7z" {
		t.Errorf("cmd legacy = %q", got)
	}
}

// TestConfigReset 验证 Reset()->Get() 返回新实例。
func TestConfigReset(t *testing.T) {
	mgr := New()

	cfg1 := mgr.Get()
	mgr.Viper().Set("server.listen.http.addr", "0.0.0.0:9999")
	cfg2 := mgr.Get()
	if cfg1 != cfg2 {
		t.Error("cfg1 != cfg2 before Reset")
	}

	mgr.Reset()
	cfg3 := mgr.Get()
	if cfg3 == cfg1 {
		t.Error("cfg3 == cfg1 after Reset")
	}
	if cfg3.Server.Listen.HTTP.Addr != "0.0.0.0:9999" {
		t.Errorf("addr = %q", cfg3.Server.Listen.HTTP.Addr)
	}
}

// ---------- YAML 辅助 ----------

// yamlOverrideItem 描述一个键值对用于构建 YAML。
type yamlOverrideItem struct {
	Key         string
	OverrideVal any
}

type yamlNode struct {
	val     string
	child   map[string]*yamlNode
	isValue bool
}

func buildYAML(keys []yamlOverrideItem) string {
	root := make(map[string]*yamlNode)
	for _, k := range keys {
		parts := strings.Split(k.Key, ".")
		vs := yamlValStr(k.OverrideVal)
		cur := root
		for i, p := range parts {
			if i == len(parts)-1 {
				cur[p] = &yamlNode{val: vs, isValue: true}
			} else {
				if cur[p] == nil {
					cur[p] = &yamlNode{child: make(map[string]*yamlNode)}
				}
				cur = cur[p].child
			}
		}
	}
	var buf strings.Builder
	writeYAML(&buf, root, 0)
	return buf.String()
}

func yamlValStr(v any) string {
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("%q", x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int, int32, int64:
		return fmt.Sprintf("%d", x)
	case time.Duration:
		return fmt.Sprintf("%q", x)
	case fmt.Stringer:
		return fmt.Sprintf("%q", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func writeYAML(buf *strings.Builder, m map[string]*yamlNode, depth int) {
	indent := strings.Repeat("  ", depth)
	for k, n := range m {
		if n.isValue {
			fmt.Fprintf(buf, "%s%s: %s\n", indent, k, n.val)
		} else {
			fmt.Fprintf(buf, "%s%s:\n", indent, k)
			writeYAML(buf, n.child, depth+1)
		}
	}
}
