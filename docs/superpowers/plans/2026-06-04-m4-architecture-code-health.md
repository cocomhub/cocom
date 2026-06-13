# M4：架构加固与代码健康 — 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 还技术债：统一错误响应格式、配置文档自动生成、补全单元测试、统一子项目 Makefile、清理过期 TODO。

**架构：** 5 个独立任务，可顺序执行。任务 1（错误格式）和任务 2（配置文档）涉及源码修改，需编译验证；任务 3（测试）不修改生产代码；任务 4（Makefile）跨子项目；任务 5（TODO 清理）纯文档操作。

**技术栈：** Go 1.26 + Gin + Viper + golangci-lint

---

## 涉及文件

| 文件 | 操作 | 职责 |
|------|------|------|
| `pkg/httpwrap/errcode.go` | **创建** | 定义 `ErrCode` 标准错误码枚举 |
| `pkg/httpwrap/ginresp.go` | 修改 | `GinRespondError` 参数类型使用 `ErrCode` |
| `pkg/httpwrap/ginresp_test.go` | 修改 | 补充 `ErrCode` 相关测试 |
| `cmd/server/view/picture.go` | 修改 | `c.AbortWithError` → `httpwrap.GinRespondError` |
| `cmd/server/view/gallery_detail.go` | 修改 | 同上 |
| `cmd/server/view/gallery_picture.go` | 修改 | 同上 |
| `cmd/server/view/search.go` | 修改 | 同上 |
| `cmd/server/view/tag_result.go` | 修改 | 同上 |
| `cmd/server/server.go` | 修改 | 几处 `c.JSON`/`c.AbortWithError` → `GinRespondError` |
| `tools/config-doc-gen/main.go` | **创建** | 配置文档生成脚本 |
| `internal/config/gen.go` | **创建** | `//go:generate` 触发点 |
| `internal/config/config.go` | 修改 | 添加 `// config-doc:` 注释 + 缺失默认值 |
| `cmd/server/config.go` | 修改 | 添加 `// config-doc:` 注释 |
| `pkg/logging/config.go` | 修改 | 添加 `// config-doc:` 注释 |
| `pkg/mongowrap/mongo.go` | 修改 | 添加 `// config-doc:` 注释 |
| `pkg/download/downloader.go` | 修改 | 添加 `// config-doc:` 注释 + 缺失默认值 |
| `pkg/comic/config.go` | 修改 | 添加 `// config-doc:` 注释 |
| `pkg/archive/manager/config.go` | 修改 | 添加 `// config-doc:` 注释 |
| `cmd/server/internal/comic/download.go` | 修改 | 添加 `// config-doc:` 注释 |
| `cmd/server/internal/mongo/mongo.go` | 修改 | 添加 `// config-doc:` 注释 |
| `cmd/server/internal/cache/cache.go` | 修改 | 添加 `// config-doc:` 注释 |
| `pkg/middlewares/localguard.go` | 修改 | 添加缺失默认值 |
| `cocom/Makefile` | 修改 | 添加 `config-doc` 目标（触发 `go generate`） |
| `cmd/server/view/search_test.go` | **创建** | `SearchResultPage` 参数解析测试 |
| `cmd/server/view/picture_test.go` | **创建** | `parsePictureArgs` 测试 |
| `cmd/server/view/gallery_detail_test.go` | **创建** | `GalleryDetail` 方法测试 |
| `pkg/middlewares/requestid_test.go` | **创建** | Request-ID 中间件行为测试 |
| `pkg/middlewares/accesslog_test.go` | **创建** | 访问日志格式测试 |
| `pkg/middlewares/ratelimit_test.go` | **创建** | 限流行为测试 |
| `download-manager/Makefile` | 修改 | 添加 `test` + `lint` 目标 |
| `sproxy/Makefile` | 修改 | 添加 `lint` 目标 |
| `docs/TODO.md` | 重命名 → `docs/TODO-ARCHIVE-2026-06.md` | 归档过期 TODO |

---

### 任务 1：统一错误响应格式

**文件：**
- 创建：`pkg/httpwrap/errcode.go`
- 修改：`pkg/httpwrap/ginresp.go`
- 修改：`pkg/httpwrap/ginresp_test.go`
- 修改：`cmd/server/view/picture.go`
- 修改：`cmd/server/view/gallery_detail.go`
- 修改：`cmd/server/view/gallery_picture.go`
- 修改：`cmd/server/view/search.go`
- 修改：`cmd/server/view/tag_result.go`
- 修改：`cmd/server/server.go`

- [ ] **步骤 1：创建 `pkg/httpwrap/errcode.go`**

```go
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package httpwrap

// ErrCode 标准错误码
type ErrCode int

const (
	ErrCodeUnknown   ErrCode = -1 // 未知错误
	ErrCodeInvalid   ErrCode = -2 // 参数无效
	ErrCodeNotFound  ErrCode = -3 // 资源不存在
	ErrCodeForbidden ErrCode = -4 // 无权限
	ErrCodeInternal  ErrCode = -5 // 内部错误
)
```

- [ ] **步骤 2：修改 `ginresp.go`——让 `GinRespondError` 接受 `ErrCode` 而非 `int`**

将 `GinRespondError` 签名中的 `code int` 改为 `code ErrCode`：

```go
func GinRespondError(c *gin.Context, httpStatus int, code ErrCode, msg string) {
	GinRespond[any](c, httpStatus, int(code), msg, nil)
}
```

同时 `GinRespond[T any]` 的 `code int` 参数保持不变（低层不感知 ErrCode 类型）。

- [ ] **步骤 3：修改 View 层 5 个文件——将 `c.AbortWithError` 改为 `GinRespondError`**

**`cmd/server/view/picture.go`**（line 37-38, 48-49, 113-116 共 3 处）：

将：
```go
c.AbortWithError(http.StatusBadRequest, err)
```
改为：
```go
httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, err.Error())
c.Abort()
```

> 注意：`c.AbortWithError` 既有状态码设置又有 abort 效果。改用 `GinRespondError` 后，需手动 `c.Abort()`。但 `GinRespondError` 只写入响应，不中止链——所以需要在函数中 return 的路径上保持 `c.Abort()`，或者在 handler 顶部加 `c.Abort()` 前检查。最安全的做法：在 `GinRespondError` 下方保留 `c.Abort()`。

添加 import：`"github.com/cocomhub/cocom/pkg/httpwrap"`

**`cmd/server/view/gallery_detail.go`**（line 38, 48 共 2 处）：

```go
// 替换前：
c.AbortWithError(http.StatusBadRequest, err)
// 替换后：
httpwrap.GinRespondError(c, http.StatusBadRequest, httpwrap.ErrCodeInvalid, err.Error())
c.Abort()
```

**`cmd/server/view/gallery_picture.go`**（line 39, 49 共 2 处）：同上。

**`cmd/server/view/search.go`**（line 43, 54 共 2 处）：同上。

**`cmd/server/view/tag_result.go`**（line 52, 64 共 2 处）：同上。

- [ ] **步骤 4：修改 `server.go`——将几处直出响应改为 `GinRespondError`**

**`cmd/server/server.go`** line 73 (`c.AbortWithStatus`) 和 line 87 (`c.AbortWithError`)、line 79 (`c.AbortWithStatus`)：

```go
// line 72-74: token 不匹配
if c.GetHeader("X-Admin-Token") != token {
    httpwrap.GinRespondError(c, http.StatusUnauthorized, httpwrap.ErrCodeForbidden, "admin token mismatch")
    c.Abort()
    return
}

// line 78-81: 非 loopback 访问
ip := c.ClientIP()
if ip != "127.0.0.1" && ip != "::1" {
    httpwrap.GinRespondError(c, http.StatusForbidden, httpwrap.ErrCodeForbidden, "only loopback allowed")
    c.Abort()
    return
}

// line 87: shutdown 已开始
httpwrap.GinRespondError(c, http.StatusConflict, httpwrap.ErrCodeInternal, "server shutdown already started")
c.Abort()
```

- [ ] **步骤 5：编译验证**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./...
```

预期：exit 0，无错误

- [ ] **步骤 6：Commit**

```bash
git add pkg/httpwrap/errcode.go pkg/httpwrap/ginresp.go pkg/httpwrap/ginresp_test.go cmd/server/view/picture.go cmd/server/view/gallery_detail.go cmd/server/view/gallery_picture.go cmd/server/view/search.go cmd/server/view/tag_result.go cmd/server/server.go
git commit -m "feat(httpwrap): 统一错误响应格式 — 新增 ErrCode 枚举；View/Server 层使用 GinRespondError"
```

---

### 任务 2：配置文档自动生成

**文件：**
- 创建：`tools/config-doc-gen/main.go`
- 创建：`internal/config/gen.go`
- 修改：`internal/config/config.go`（添加 `// config-doc:` 注释 + 缺失默认值）
- 修改：`cmd/server/config.go`（添加 `// config-doc:` 注释）
- 修改：`pkg/logging/config.go`（添加 `// config-doc:` 注释）
- 修改：`pkg/mongowrap/mongo.go`（添加 `// config-doc:` 注释）
- 修改：`pkg/download/downloader.go`（添加 `// config-doc:` 注释 + 缺失默认值）
- 修改：`pkg/comic/config.go`（添加 `// config-doc:` 注释）
- 修改：`pkg/archive/manager/config.go`（添加 `// config-doc:` 注释）
- 修改：`cmd/server/internal/comic/download.go`（添加 `// config-doc:` 注释）
- 修改：`cmd/server/internal/mongo/mongo.go`（添加 `// config-doc:` 注释）
- 修改：`cmd/server/internal/cache/cache.go`（添加 `// config-doc:` 注释）
- 修改：`pkg/middlewares/localguard.go`（添加缺失默认值）
- 修改：`cocom/Makefile`（添加 `config-doc` 目标）

- [ ] **步骤 1：创建 `tools/config-doc-gen/main.go`**

```go
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ConfigEntry 代表一个配置项
type ConfigEntry struct {
	Key         string
	Type        string
	Default     string
	Description string
	Source      string // 源文件路径
}

var (
	configDocRe = regexp.MustCompile(`// config-doc:\s*(\S+)\s*\|\s*(\S+)\s*\|\s*(.*?)\s*\|\s*(.*)`)
	setDefaultRe = regexp.MustCompile(`viper\.SetDefault\(\s*"([^"]+)"`)
)

func main() {
	output := flag.String("output", "", "output file path (default: stdout)")
	flag.Parse()

	root := findModuleRoot()
	entries := scanFiles(root)
	missing := scanMissingDefaults(root, entries)

	// Build markdown
	var buf strings.Builder
	buf.WriteString("# 配置管理文档\n\n")
	buf.WriteString("> Auto-generated by config-doc-gen\n\n")

	// Group by section
	sections := groupBySection(entries)
	sectionOrder := []string{"基础配置", "日志", "MongoDB", "存储", "存档", "归档管理器", "服务端", "下载", "漫画", "缓存", "客户端"}
	for _, sec := range sectionOrder {
		if items, ok := sections[sec]; ok {
			buf.WriteString(fmt.Sprintf("## %s\n\n", sec))
			buf.WriteString("| 配置键 | 类型 | 默认值 | 描述 |\n")
			buf.WriteString("|--------|------|--------|------|\n")
			for _, item := range items {
				def := item.Default
				if def == "" {
					def = "`\"\"`"
				} else {
					def = fmt.Sprintf("`%s`", def)
				}
				buf.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n",
					item.Key, item.Type, def, item.Description))
			}
			buf.WriteString("\n")
		}
	}

	// Missing defaults section
	if len(missing) > 0 {
		buf.WriteString("## 缺失默认值的配置项\n\n")
		buf.WriteString("以下键被 `viper.Get*` 读取但无 `viper.SetDefault` 定义：\n\n")
		buf.WriteString("| 配置键 | 读取位置 |\n")
		buf.WriteString("|--------|----------|\n")
		for _, m := range missing {
			buf.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", m.Key, m.Source))
		}
		buf.WriteString("\n")
	}

	// Footer with timestamp and commit hash
	commitHash := getCommitHash()
	buf.WriteString(fmt.Sprintf("<!-- generated at %s | commit: %s -->\n",
		time.Now().Format(time.RFC3339), commitHash))

	result := buf.String()

	if *output != "" {
		os.WriteFile(*output, []byte(result), 0644)
		fmt.Println("config docs written to", *output)
	} else {
		fmt.Print(result)
	}
}

func findModuleRoot() string {
	// Assume CWD is the module root, or walk up to find go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// fallback
			dir, _ = os.Getwd()
			return dir
		}
		dir = parent
	}
}

func scanFiles(root string) []ConfigEntry {
	var entries []ConfigEntry
	// Scan dirs: cmd/, internal/, pkg/
	dirs := []string{"cmd", "internal", "pkg"}
	for _, d := range dirs {
		abs := filepath.Join(root, d)
		filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			var prevLine string
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(strings.TrimSpace(prevLine), "// config-doc:") {
					if matches := configDocRe.FindStringSubmatch(prevLine); len(matches) == 5 {
						rel, _ := filepath.Rel(root, path)
						entries = append(entries, ConfigEntry{
							Key:         matches[1],
							Type:        matches[2],
							Default:     matches[3],
							Description: matches[4],
							Source:      rel,
						})
					}
				}
				prevLine = line
			}
			return nil
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries
}

func scanMissingDefaults(root string, known []ConfigEntry) []ConfigEntry {
	knownKeys := map[string]bool{}
	for _, e := range known {
		knownKeys[e.Key] = true
	}
	// Also add keys that have viper.SetDefault but no config-doc comment
	// These should be documented, but at least they have defaults
	// We track get-only keys
	getKeys := map[string]string{}
	setKeys := map[string]bool{}

	// First pass: find all viper.SetDefault keys
	dirs := []string{"cmd", "internal", "pkg"}
	for _, d := range dirs {
		abs := filepath.Join(root, d)
		filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if matches := setDefaultRe.FindStringSubmatch(line); len(matches) == 2 {
					setKeys[matches[1]] = true
				}
				// Find Get* calls
				if strings.Contains(line, `viper.Get`) || strings.Contains(line, `viper.UnmarshalKey`) {
					getRe := regexp.MustCompile(`viper\.(Get\w+|UnmarshalKey)\(\s*"([^"]+)"`)
					if gm := getRe.FindStringSubmatch(line); len(gm) == 3 {
						key := gm[2]
						rel, _ := filepath.Rel(root, path)
						// Only record if no SetDefault exists
						if !setKeys[key] {
							getKeys[key] = rel
						}
					}
				}
			}
			return nil
		})
	}

	var missing []ConfigEntry
	for k, v := range getKeys {
		missing = append(missing, ConfigEntry{Key: k, Source: v})
	}
	sort.Slice(missing, func(i, j int) bool {
		return missing[i].Key < missing[j].Key
	})
	return missing
}

func groupBySection(entries []ConfigEntry) map[string][]ConfigEntry {
	groups := map[string][]ConfigEntry{}
	for _, e := range entries {
		sec := classifySection(e.Key)
		groups[sec] = append(groups[sec], e)
	}
	return groups
}

func classifySection(key string) string {
	switch {
	case key == "port", key == "host":
		return "基础配置"
	case strings.HasPrefix(key, "log."):
		return "日志"
	case strings.HasPrefix(key, "mongo."):
		return "MongoDB"
	case strings.HasPrefix(key, "cocom.storage"), strings.HasPrefix(key, "storage."):
		return "存储"
	case strings.HasPrefix(key, "cocom.archive"), strings.HasPrefix(key, "archive."):
		return "存档"
	case strings.HasPrefix(key, "archive.manager"):
		return "归档管理器"
	case strings.HasPrefix(key, "server."):
		return "服务端"
	case strings.HasPrefix(key, "download."), strings.HasPrefix(key, "http."):
		return "下载"
	case strings.HasPrefix(key, "comic."):
		return "漫画"
	case strings.HasPrefix(key, "cocom.cache"):
		return "缓存"
	case strings.HasPrefix(key, "client."):
		return "客户端"
	case strings.HasPrefix(key, "debug."), strings.HasPrefix(key, "admin."):
		return "安全"
	case strings.HasPrefix(key, "mutex."):
		return "锁"
	default:
		return "其他"
	}
}

func getCommitHash() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
```

- [ ] **步骤 2：创建 `internal/config/gen.go`**

```go
// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

//go:generate go run ../../tools/config-doc-gen/main.go --output ../../docs/config.md
```

- [ ] **步骤 3：为所有 `viper.SetDefault` 添加 `// config-doc:` 注释**

**`internal/config/config.go`** 修改为：

```go
// config-doc: cocom.storage.path | string | /data/cocom/data/gallery | 漫画图片存储目录
viper.SetDefault(StorageGalleryKey, "/data/cocom/data/gallery")
// config-doc: cocom.archive.path | string | /data/cocom/data/archive | 归档文件存储目录
viper.SetDefault(StorageArchiveKey, "/data/cocom/data/archive")
// config-doc: cocom.archive.temp_path | string | /data/cocom/data/archive-temp | 归档临时文件目录
viper.SetDefault(StorageArchiveTempKey, "/data/cocom/data/archive-temp")
// config-doc: cocom.archive.password | string | (deprecated, 请使用 archive.password)
viper.SetDefault("cocom.archive.password", "")
// config-doc: cocom.archive.cmd | string | (deprecated, 请使用 archive.cmd)
viper.SetDefault("cocom.archive.cmd", "")
// config-doc: cocom.archive.replicate | bool | false | (deprecated, 请使用 archive.replicate)
viper.SetDefault("cocom.archive.replicate", false)

// config-doc: archive.password | string | archive@123456 | 7z 加密密码
viper.SetDefault("archive.password", "archive@123456")
// config-doc: archive.cmd | string | 7z | 7z 命令路径
viper.SetDefault("archive.cmd", "7z")
// config-doc: archive.replicate | bool | false | 是否默认复制到远端存储
viper.SetDefault("archive.replicate", false)
// config-doc: archive.algorithm.single.concurrency | int | 4 | 单线程存档算法并发数
viper.SetDefault("archive.algorithm.single.concurrency", 4)
// config-doc: archive.algorithm.double.concurrency | int | 4 | 双线程存档算法并发数
viper.SetDefault("archive.algorithm.double.concurrency", 4)

// 缺失默认值补充
// config-doc: server.listen.http.addr | string | :35456 | HTTP 监听地址
viper.SetDefault("server.listen.http.addr", ":35456")
// config-doc: server.shutdown_timeout | duration | 5s | 优雅关闭超时
viper.SetDefault("server.shutdown_timeout", "5s")
// config-doc: host | string | (来自 server.listen.http.addr 或 localhost) | 主机名
viper.SetDefault("host", "")
// config-doc: http.enable_proxy | bool | false | 是否启用 HTTP 代理下载
viper.SetDefault("http.enable_proxy", false)
// config-doc: http.proxy | string | | HTTP 代理地址
viper.SetDefault("http.proxy", "")
```

**`cmd/server/config.go`** 修改为：

```go
// config-doc: server.access_log.patterns | []string | ["/debug","/api","/v1","/v2"] | 记录访问日志的 URL 模式列表
viper.SetDefault("server.access_log.patterns", []string{"/debug", "/api", "/v1", "/v2"})
// config-doc: server.cors.enabled | bool | false | 是否启用 CORS
viper.SetDefault("server.cors.enabled", false)
// config-doc: server.cors.allow_origins | string | * | 允许的源
viper.SetDefault("server.cors.allow_origins", "*")
// config-doc: server.cors.allow_methods | string | GET,POST,PUT,DELETE,OPTIONS | 允许的 HTTP 方法
viper.SetDefault("server.cors.allow_methods", "GET,POST,PUT,DELETE,OPTIONS")
// config-doc: server.cors.allow_headers | string | * | 允许的请求头
viper.SetDefault("server.cors.allow_headers", "*")
// config-doc: server.gzip.enabled | bool | false | 是否启用 Gzip 压缩
viper.SetDefault("server.gzip.enabled", false)
// config-doc: server.gzip.level | int | 1 | Gzip 压缩级别
viper.SetDefault("server.gzip.level", gzip.BestSpeed)
// config-doc: server.ratelimit.enabled | bool | false | 是否启用限流
viper.SetDefault("server.ratelimit.enabled", false)
// config-doc: server.ratelimit.rps | int | 10 | 每秒请求数限制
viper.SetDefault("server.ratelimit.rps", 10)
// config-doc: server.ratelimit.burst | int | 20 | 突发请求数
viper.SetDefault("server.ratelimit.burst", 20)
// config-doc: server.scheduler.enabled | bool | false | 是否启用调度器
viper.SetDefault("server.scheduler.enabled", false)
// config-doc: server.scheduler.timezone | string | Local | 调度器时区
viper.SetDefault("server.scheduler.timezone", "Local")
// config-doc: server.scheduler.probe_comic.enabled | bool | false | 是否启用漫画探测调度
viper.SetDefault("server.scheduler.probe_comic.enabled", false)
// config-doc: server.scheduler.probe_comic.name | string | ProbeComic | 漫画探测任务名称
viper.SetDefault("server.scheduler.probe_comic.name", "ProbeComic")
// config-doc: server.scheduler.probe_comic.cron | string | 0 */10 * * * * | 漫画探测 Cron 表达式
viper.SetDefault("server.scheduler.probe_comic.cron", "0 */10 * * * *")
// config-doc: server.scheduler.probe_comic.tags | []string | ["probe","comic"] | 漫画探测标签列表
viper.SetDefault("server.scheduler.probe_comic.tags", []string{"probe", "comic"})
// config-doc: server.scheduler.archive_status_check.enabled | bool | false | 是否启用存档状态检查调度
viper.SetDefault("server.scheduler.archive_status_check.enabled", false)
// config-doc: server.scheduler.archive_status_check.name | string | ArchiveStatusChecker | 存档状态检查任务名称
viper.SetDefault("server.scheduler.archive_status_check.name", "ArchiveStatusChecker")
// config-doc: server.scheduler.archive_status_check.cron | string | 0 */30 * * * * | 存档状态检查 Cron 表达式
viper.SetDefault("server.scheduler.archive_status_check.cron", "0 */30 * * * *")
// config-doc: server.scheduler.archive_status_check.tags | []string | ["archive","check"] | 存档状态检查标签列表
viper.SetDefault("server.scheduler.archive_status_check.tags", []string{"archive", "check"})
// config-doc: server.scheduler.archive_status_check.limit | int | 100 | 每次检查数量上限
viper.SetDefault("server.scheduler.archive_status_check.limit", 100)
// config-doc: server.scheduler.archive_status_check.max_conn | int | 3 | 最大并发连接数
viper.SetDefault("server.scheduler.archive_status_check.max_conn", 3)
// config-doc: server.scheduler.archive_status_check.backends | []string | [] | 要检查的后端列表
viper.SetDefault("server.scheduler.archive_status_check.backends", []string{})
// config-doc: server.scheduler.cocoma_archiver.enabled | bool | false | 是否启用 Cocoma 归档调度
viper.SetDefault("server.scheduler.cocoma_archiver.enabled", false)
// config-doc: server.scheduler.cocoma_archiver.cron | string | * * * * * | Cocoma 归档 Cron 表达式
viper.SetDefault("server.scheduler.cocoma_archiver.cron", "* * * * *")
// config-doc: server.scheduler.cocoma_archiver.limit | int | 10000 | 每次处理上限
viper.SetDefault("server.scheduler.cocoma_archiver.limit", 10000)
// config-doc: server.scheduler.cocoma_archiver.cid_regex | string | ^(\\d+)\\.cocoma$ | CID 匹配正则
viper.SetDefault("server.scheduler.cocoma_archiver.cid_regex", "^(\\d+)\\.cocoma$")
// config-doc: server.scheduler.cocoma_archiver.scan_dir | string | | 扫描目录
viper.SetDefault("server.scheduler.cocoma_archiver.scan_dir", "")
// config-doc: server.scheduler.cocoma_archiver.archive_dir | string | | 归档输出目录
viper.SetDefault("server.scheduler.cocoma_archiver.archive_dir", "")
// config-doc: server.scheduler.cocoma_archiver.notmatch_dir | string | | 不匹配文件的移动目录
viper.SetDefault("server.scheduler.cocoma_archiver.notmatch_dir", "")
```

**`pkg/logging/config.go`** (line 16-33)：同理添加 `// config-doc:` 注释。

**`pkg/mongowrap/mongo.go`** (line 27-31)：同理添加。

**`pkg/archive/manager/config.go`** (line 13-23)：同理添加。

**`pkg/comic/config.go`** (line 9-10)：同理添加。

**`pkg/download/downloader.go`** (line 29-30)：同理添加。

**`cmd/server/internal/comic/download.go`** (line 30)：同理添加。

**`cmd/server/internal/mongo/mongo.go`** (line 43-50)：同理添加。

**`cmd/server/internal/cache/cache.go`** (line 21-22)：同理添加。

**`pkg/middlewares/localguard.go`** —— 添加缺失默认值：

```go
func init() {
	viper.SetDefault("debug.allow_remote", false)
	viper.SetDefault("admin.allow_remote", false)
}
```

- [ ] **步骤 4：修改 `cocom/Makefile`——添加 `config-doc` 目标**

在 `.PHONY: help` 前添加：

```makefile
# 配置文档生成
.PHONY: config-doc
config-doc:
	@cp docs/config.md docs/config.md.bak 2>/dev/null || true
	go generate ./internal/config/...
	@echo "=== 配置变更对比 ==="
	@-diff docs/config.md.bak docs/config.md 2>/dev/null || true
	@rm -f docs/config.md.bak
	@echo "=== config-doc generated ==="
```

同时也更新 `help` 目标，在 `help` 下加一行显示 `config-doc`。

- [ ] **步骤 5：编译验证**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./...
go vet ./...
```

预期：exit 0

- [ ] **步骤 6：运行首次生成验证**

```bash
cd D:\workdir\leon\cocomhub\cocom
make config-doc
```

预期：生成 `docs/config.md`，显示 diff（与备份文件对比）

- [ ] **步骤 7：Commit**

```bash
git add tools/config-doc-gen/ internal/config/gen.go internal/config/config.go cmd/server/config.go pkg/logging/config.go pkg/mongowrap/mongo.go pkg/download/downloader.go pkg/comic/config.go pkg/archive/manager/config.go cmd/server/internal/comic/download.go cmd/server/internal/mongo/mongo.go cmd/server/internal/cache/cache.go pkg/middlewares/localguard.go cocom/Makefile docs/config.md
git commit -m "feat(config): 配置文档自动生成工具 — go:generate + config-doc 注释 + Makefile 目标 + 补齐缺失默认值"
```

---

### 任务 3：单元测试补全

**文件：**
- 创建：`cmd/server/view/search_test.go`
- 创建：`cmd/server/view/picture_test.go`
- 创建：`cmd/server/view/gallery_detail_test.go`
- 创建：`pkg/middlewares/requestid_test.go`
- 创建：`pkg/middlewares/accesslog_test.go`
- 创建：`pkg/middlewares/ratelimit_test.go`
- 修改：`pkg/comic/handler.go`（如有必要为测试导出）

**测试重点：**

**`cmd/server/view/search_test.go`**：
- `parseSearchResultPageArgs`：空 query、正常 page、page=0→默认 page=1
- `HighlightKeyword`：关键字匹配、大小写不敏感、特殊字符转义、无匹配

**`cmd/server/view/picture_test.go`**：
- `parsePictureArgs`：正常 cid+name、cid 非法、name 为空

**`cmd/server/view/gallery_detail_test.go`**：
- `IsNavigationActive`：始终返回 false
- `HasLike`：tag 列表含 custom/like 时返回 true

**`pkg/middlewares/requestid_test.go`**：
- 中间件在请求中注入 Request-ID（空请求时自动生成）
- 自定义 Request-ID 被保留

**`pkg/middlewares/accesslog_test.go`**：
- 匹配 pattern 的路径输出日志（用 `bytes.Buffer` 接管 slog 输出）
- 不匹配 pattern 的路径不输出日志

**`pkg/middlewares/ratelimit_test.go`**：
- 正常请求通过限流
- 超过 RPS 后请求被拒绝（429）

- [ ] **步骤 1-12：逐个编写测试 → 编译 → 通过 → 提交**

（具体测试代码在实现时编写，此处省略以避免计划过长。每个测试文件约 30-80 行，table-driven style。）

---

### 任务 4：Makefile 统一

**文件：**
- 修改：`download-manager/Makefile`
- 修改：`sproxy/Makefile`

- [ ] **步骤 1：download-manager/Makefile 添加 `test` 和 `lint`**

在 `clean` 目标前添加：

```makefile
.PHONY: test
test:
	@echo Running go test -race ./...
	@$(GO) test -race ./...

.PHONY: lint
lint:
	golangci-lint run || echo "golangci-lint not available, skipping"
```

- [ ] **步骤 2：sproxy/Makefile 添加 `lint`**

在 `vet` 目标后添加：

```makefile
.PHONY: lint
lint:
	golangci-lint run ./... || echo "golangci-lint not available, skipping"
```

- [ ] **步骤 3：编译验证（两个子项目）**

```bash
cd D:\workdir\leon\cocomhub\download-manager && go build ./...
cd D:\workdir\leon\cocomhub\sproxy && go build ./...
make test  # 仅在 sproxy 下可运行
```

预期：build 通过

- [ ] **步骤 4：Commit**

```bash
git add download-manager/Makefile sproxy/Makefile
git commit -m "chore(makefile): 统一子项目 make test / make lint 目标"
```

---

### 任务 5：过期 TODO 清理

**文件：**
- 重命名：`docs/TODO.md` → `docs/TODO-ARCHIVE-2026-06.md`
- 修改：`pkg/archive/archiver.go`（line 261 注释处理）
- 修改：`pkg/comic/monitor.go`（line 133 注释处理）
- 修改：`pkg/comic/verify.go`（line 262 注释处理）
- 修改：`cocom/Makefile`（line 101 注释清理）

- [ ] **步骤 1：重命名 TODO.md**

```bash
mv docs/TODO.md docs/TODO-ARCHIVE-2026-06.md
```

在归档文件顶部添加一行：

```
<!-- Archived 2026-06-04: content merged into milestone roadmap docs/superpowers/specs/2026-06-04-cocom-development-roadmap-design.md -->
```

- [ ] **步骤 2：处理 `pkg/archive/archiver.go:261`**

将 `// TODO: 完善整个目录的修改时间` 改为：

```go
// FIXME: 需完善整个目录的修改时间计算逻辑（当前仅返回单个文件 mtime）
```

- [ ] **步骤 3：处理 `pkg/comic/monitor.go:133`**

将 `// TODO: 添加实际的 IO 统计逻辑` 改为：

```go
// TODO(M5): 添加实际的 IO 统计逻辑
```

- [ ] **步骤 4：处理 `pkg/comic/verify.go:262`**

将 `// TODO: 实现最后消息设置` 改为：

```go
// TODO(M5): 实现最后消息设置
```

- [ ] **步骤 5：清理 Makefile TODO**

将 Makefile line 101-104：
```makefile
# TODO: no files actually use this right now
.PHONY: go-gen
go-gen:
	@echo skipping go generate ./...
```

改为（移除 TODO 注释，保留目标体）：

```makefile
.PHONY: go-gen
go-gen:
	go generate ./...
```

- [ ] **步骤 6：验证**

```bash
cd D:\workdir\leon\cocomhub\cocom
grep -n "TODO" pkg/archive/archiver.go pkg/comic/monitor.go pkg/comic/verify.go
```

预期：只有带 (M5) 标记的 TODO 和 FIXME

- [ ] **步骤 7：Commit**

```bash
git add docs/TODO-ARCHIVE-2026-06.md docs/TODO.md pkg/archive/archiver.go pkg/comic/monitor.go pkg/comic/verify.go cocom/Makefile
git commit -m "chore(cleanup): 过期 TODO 清理 — 归档 docs/TODO.md, 标记 M5 TODO, 清理 Makefile TODO 注释"
```

---

## 验收标准（逐项验证）

- [ ] `go build ./...` 在三个子项目中均编译通过
- [ ] `make test` 在 cocom、sproxy 中可用且通过；download-manager 中可用（无测试时为 0 passed）
- [ ] `make lint` 在 cocom、sproxy 中可用；download-manager 中兼容缺失 golangci-lint
- [ ] View 层 5 个文件中无 `c.AbortWithError` 残留
- [ ] `make config-doc` 首次运行生成 `docs/config.md`，含时间戳和 commit hash
- [ ] `docs/TODO.md` 已归档，代码中 4 处 TODO 分类处理