// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/cocomhub/cocom/internal/rootcli"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var migrateFlags struct {
	dryRun bool
	yes    bool
}

// migration 描述一次旧键 → 新键的迁移。
type migration struct {
	oldKey    string
	newKey    string
	value     any
	sensitive bool // 敏感键（口令等），diff 输出时脱敏
}

func init() {
	configMigrateCmd.Flags().BoolVar(&migrateFlags.dryRun, "dry-run", false, "只输出迁移 diff，不写盘")
	configMigrateCmd.Flags().BoolVarP(&migrateFlags.yes, "yes", "y", false, "跳过 host/port 迁移确认")
	configCmd.AddCommand(configMigrateCmd)
	rootCmd.AddCommand(configCmd)
}

// configCmd 是配置相关子命令的父命令。
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理（迁移旧版配置键等）",
}

var configMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "迁移旧版配置键到新版键",
	Long: `一次性迁移旧版配置键到新版键，并输出迁移前后对照。

支持的迁移映射：
  archive.password/cmd/replicate/algorithm.*  ->  cocom.archive.*
  storage.backends                            ->  cocom.storage.backends
  admin.token / admin.allow_remote           ->  server.admin.*
  debug.allow_remote                        ->  server.admin.allow_remote
  http.enable_proxy / http.proxy            ->  download.enableProxy / download.proxyURL
  host / port                                 ->  server.listen.http.addr（需确认拼接方式）

注意：
  - 已用旧口令归档的文件，迁移配置后不会自动可解；
    需在迁移后显式确认 cocom.archive.password 与历史口令一致。
  - 迁移会重写配置文件（注释将丢失），建议先备份原文件。

示例：
  cocom config migrate --dry-run
  cocom config migrate --config ./cocom.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigMigrate(cmd)
	},
}

func runConfigMigrate(cmd *cobra.Command) error {
	cfgFile := rootcli.ConfigFile()
	if cfgFile == "" {
		return errors.New("未指定配置文件，请通过 --config 指定")
	}
	raw, err := os.ReadFile(cfgFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败：%w", err)
	}
	var data map[string]any
	if umErr := yaml.Unmarshal(raw, &data); umErr != nil {
		return fmt.Errorf("解析配置文件失败：%w", umErr)
	}

	var migrations []migration
	var overwroteKeys []string
	collect := func(oldPath, newPath string, v any, overwrote bool) {
		migrations = append(migrations, migration{oldKey: oldPath, newKey: newPath, value: v, sensitive: isSensitiveKey(oldPath)})
		if overwrote {
			overwroteKeys = append(overwroteKeys, newPath)
		}
	}

	// archive.* -> cocom.archive.*
	for _, key := range []string{"password", "cmd"} {
		oldPath := "archive." + key
		newPath := "cocom.archive." + key
		if v, ok := getPath(data, strings.Split(oldPath, ".")...); ok {
			// 铁律 3：新旧键冲突（两者都显式配置且不同）→ 迁移失败，需人工决策。
			existing, hasNew := getPath(data, strings.Split(newPath, ".")...)
			if hasNew && !isZeroValue(existing) && !reflect.DeepEqual(existing, v) {
				return conflictError(oldPath, newPath, v, existing)
			}
			overwrote := hasNew
			setPath(data, v, strings.Split(newPath, ".")...)
			deletePath(data, strings.Split(oldPath, ".")...)
			collect(oldPath, newPath, v, overwrote)
		}
	}
	if v, ok := getPath(data, "archive", "replicate"); ok {
		existing, hasNew := getPath(data, "cocom", "archive", "replicate")
		if hasNew && existing != v {
			return conflictError("archive.replicate", "cocom.archive.replicate", v, existing)
		}
		overwrote := hasNew
		setPath(data, v, "cocom", "archive", "replicate")
		deletePath(data, "archive", "replicate")
		collect("archive.replicate", "cocom.archive.replicate", v, overwrote)
	}
	if v, ok := getPath(data, "archive", "algorithm"); ok {
		existing, hasNew := getPath(data, "cocom", "archive", "algorithm")
		if hasNew && !reflect.DeepEqual(existing, v) {
			return conflictError("archive.algorithm", "cocom.archive.algorithm", v, existing)
		}
		overwrote := hasNew
		setPath(data, v, "cocom", "archive", "algorithm")
		deletePath(data, "archive", "algorithm")
		collect("archive.algorithm", "cocom.archive.algorithm", v, overwrote)
	}

	// storage.backends -> cocom.storage.backends
	if v, ok := getPath(data, "storage", "backends"); ok {
		existing, hasNew := getPath(data, "cocom", "storage", "backends")
		if hasNew && !reflect.DeepEqual(existing, v) {
			return conflictError("storage.backends", "cocom.storage.backends", v, existing)
		}
		overwrote := hasNew
		setPath(data, v, "cocom", "storage", "backends")
		deletePath(data, "storage", "backends")
		collect("storage.backends", "cocom.storage.backends", v, overwrote)
	}

	// admin.token -> server.admin.token（敏感键，diff 自动脱敏为 ***）
	// debug.allow_remote / admin.allow_remote -> server.admin.allow_remote
	// 两旧 allow_remote 同落目标时按顺序迁移；后迁移者与目标不同值 → conflictError 需人工决策。
	if v, ok := getPath(data, "admin", "token"); ok {
		existing, hasNew := getPath(data, "server", "admin", "token")
		if hasNew && existing != v {
			return conflictError("admin.token", "server.admin.token", v, existing)
		}
		overwrote := hasNew
		setPath(data, v, "server", "admin", "token")
		deletePath(data, "admin", "token")
		collect("admin.token", "server.admin.token", v, overwrote)
	}
	for _, oldPath := range []string{"admin.allow_remote", "debug.allow_remote"} {
		if v, ok := getPath(data, strings.Split(oldPath, ".")...); ok {
			existing, hasNew := getPath(data, "server", "admin", "allow_remote")
			if hasNew && existing != v {
				return conflictError(oldPath, "server.admin.allow_remote", v, existing)
			}
			overwrote := hasNew
			setPath(data, v, "server", "admin", "allow_remote")
			deletePath(data, strings.Split(oldPath, ".")...)
			collect(oldPath, "server.admin.allow_remote", v, overwrote)
		}
	}

	// http.* -> download.*
	if v, ok := getPath(data, "http", "enable_proxy"); ok {
		existing, hasNew := getPath(data, "download", "enableProxy")
		if hasNew && existing != v {
			return conflictError("http.enable_proxy", "download.enableProxy", v, existing)
		}
		overwrote := hasNew
		setPath(data, v, "download", "enableProxy")
		deletePath(data, "http", "enable_proxy")
		collect("http.enable_proxy", "download.enableProxy", v, overwrote)
	}
	if v, ok := getPath(data, "http", "proxy"); ok {
		existing, hasNew := getPath(data, "download", "proxyURL")
		if hasNew && existing != v {
			return conflictError("http.proxy", "download.proxyURL", v, existing)
		}
		overwrote := hasNew
		setPath(data, v, "download", "proxyURL")
		deletePath(data, "http", "proxy")
		collect("http.proxy", "download.proxyURL", v, overwrote)
	}

	// host/port -> server.listen.http.addr（仅当新键未设置时才迁移，避免覆盖现有 addr）
	existingAddr, hasAddr := getPath(data, "server", "listen", "http", "addr")
	if !hasAddr || strings.TrimSpace(toStringOr(existingAddr, "")) == "" {
		hostVal, hasHost := getPath(data, "host")
		portVal, hasPort := getPath(data, "port")
		if hasHost || hasPort {
			addr, composeErr := composeListenAddr(hostVal, hasHost, portVal, hasPort)
			if composeErr != nil {
				return composeErr
			}
			if strings.HasPrefix(addr, "0.0.0.0:") || strings.HasPrefix(addr, "0.0.0.0") {
				slog.Warn("migrate 组合出 0.0.0.0 监听地址——将对所有网络接口开放；建议显式配置为具体 IP 或 127.0.0.1",
					slog.String("key", "server.listen.http.addr"), slog.String("addr", addr))
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "检测到旧版 host/port，将迁移为 server.listen.http.addr: %q\n", addr)
			if !migrateFlags.yes && !confirm(cmd, "确认迁移 host/port 到 server.listen.http.addr？") {
				fmt.Fprintln(cmd.ErrOrStderr(), "已跳过 host/port 迁移（其余迁移仍会执行）")
			} else {
				setPath(data, addr, "server", "listen", "http", "addr")
				if hasHost {
					deletePath(data, "host")
				}
				if hasPort {
					deletePath(data, "port")
				}
				collect("host/port", "server.listen.http.addr", addr, hasAddr)
			}
		}
	}

	// 清理迁移后遗留的空父节点（archive/http/storage 等只剩空 map 的中间键）
	pruneEmpty(data)

	if len(migrations) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "没有发现需要迁移的旧版配置键")
		return nil
	}

	// 输出迁移前后 diff（排序稳定输出；敏感键值脱敏）
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].oldKey < migrations[j].oldKey })
	fmt.Fprintln(cmd.OutOrStdout(), "=== 迁移 diff ===")
	for _, m := range migrations {
		val := m.value
		if m.sensitive || valueHasCredentials(m.value) {
			val = "***"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %-44s ->  %-44s = %v\n", m.oldKey, m.newKey, val)
	}

	// 写盘前整体校验新键冲突（铁律 3）：已迁移键中若有本次尝试覆盖的（新键原值非零）。
	if overwriteErr := validateMigratedOverwrites(cmd, overwroteKeys); overwriteErr != nil {
		return overwriteErr
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化配置失败：%w", err)
	}

	if migrateFlags.dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "（--dry-run：未写盘）")
		return nil
	}

	// 实际写盘：原子替换（临时文件 + rename；失败时清除临时文件，原文件保留）。
	if err := writeFileAtomic(cfgFile, out); err != nil {
		return fmt.Errorf("写入配置文件失败：%w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "已写入 %s，共迁移 %d 个键。\n", cfgFile, len(migrations))
	fmt.Fprintln(cmd.ErrOrStderr(), "注意：已用旧口令归档的文件不会自动可解，请确认 cocom.archive.password 与历史口令一致。")
	return nil
}

// validateMigratedOverwrites 检查迁移汇总中覆盖非零新键值的情况并给出提示。
// 真实的冲突（新旧同时显式配置且不同）已在各迁移分支 fail-fast（铁律 3）；
// 此处仅对「新键有值但不冲突」的覆盖做展示性提示，帮助用户审阅。
func validateMigratedOverwrites(cmd *cobra.Command, overwroteKeys []string) error {
	for _, k := range overwroteKeys {
		fmt.Fprintf(cmd.ErrOrStderr(), "注意：新键 %s 已存在非默认值，config migrate 已按旧键值覆盖（新旧不冲突时允许覆盖）。若要保留原新键值，请先手动编辑配置文件。\n", k)
	}
	return nil
}

// composeListenAddr 拼接 host/port 为 server.listen.http.addr。
// 铁律 4：host 缺失默认 127.0.0.1（非 0.0.0.0）；仅显式 host 为 0.0.0.0 时保留，
// 由调用方负责记录 0.0.0.0 的 WARN 日志。
func composeListenAddr(hostVal any, hasHost bool, portVal any, hasPort bool) (string, error) {
	host := "127.0.0.1"
	if hasHost {
		h, ok := hostVal.(string)
		if !ok {
			return "", fmt.Errorf("config: invalid key %q: %v (want string)", "host", hostVal)
		}
		if strings.TrimSpace(h) != "" {
			host = h
		}
	}
	port := "8080"
	if hasPort {
		p, ok := toInt(portVal)
		if !ok {
			return "", fmt.Errorf("config: invalid key %q: %v (want int)", "port", portVal)
		}
		port = fmt.Sprintf("%d", p)
	}
	return fmt.Sprintf("%s:%s", host, port), nil
}

// confirm 交互确认。非交互输入（EOF）视为拒绝。
func confirm(cmd *cobra.Command, msg string) bool {
	fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N] ", msg)
	var ans string
	_, _ = fmt.Fscanln(os.Stdin, &ans)
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes"
}

// writeFileAtomic 原子写盘：把内容写入同目录临时文件（0o600/原权限），再 os.Rename
// 覆盖目标。Rename 失败时清除临时文件，原文件保持原样，避免写一半的配置文件（铁律 1/3）。
func writeFileAtomic(target string, out []byte) error {
	mode := os.FileMode(0o600)
	if fi, statErr := os.Stat(target); statErr == nil {
		if fi.Mode().Perm()&0o077 == 0 {
			mode = fi.Mode().Perm()
		}
	}
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".config-migrate-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	dispose := func(err error) error { tmp.Close(); cleanup(); return err }
	if _, err := tmp.Write(out); err != nil {
		return dispose(err)
	}
	if err := tmp.Sync(); err != nil {
		return dispose(err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		cleanup()
		return err
	}
	return nil
}

// isZeroValue 判断 v 是否为对应 Go 类型的零值（用于「新键是否显式配置」判断）。
// map/slice/string/bool/数值类型覆盖；其余类型按非 nil 视为非零。
func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return !rv.IsValid() || rv.IsZero()
}

// conflictError 构造新旧键冲突的错误信息（铁律 3）。
func conflictError(oldPath, newPath string, oldVal, newVal any) error {
	return fmt.Errorf("迁移冲突：%s=%q 与 %s=%q 同时显式配置且值不同——请人工决策后手动统一配置（新旧冲突，需人工决策），config migrate 不再自动覆盖",
		oldPath, fmt.Sprint(oldVal), newPath, fmt.Sprint(newVal))
}

// getPath 从嵌套 map 中读取 path 指定的值。
func getPath(m map[string]any, path ...string) (any, bool) {
	cur := any(m)
	for i, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := mm[p]
		if !ok {
			return nil, false
		}
		if i == len(path)-1 {
			return v, true
		}
		cur = v
	}
	return nil, false
}

// setPath 在嵌套 map 中写入 path 指定的值（中间节点自动创建）。
func setPath(m map[string]any, value any, path ...string) {
	cur := m
	for _, p := range path[:len(path)-1] {
		child, ok := cur[p].(map[string]any)
		if !ok {
			child = map[string]any{}
			cur[p] = child
		}
		cur = child
	}
	cur[path[len(path)-1]] = value
}

// deletePath 从嵌套 map 中删除 path 指定的键（空中间节点保留，不做清理）。
func deletePath(m map[string]any, path ...string) {
	cur := m
	for _, p := range path[:len(path)-1] {
		child, ok := cur[p].(map[string]any)
		if !ok {
			return
		}
		cur = child
	}
	delete(cur, path[len(path)-1])
}

// toInt 将 yaml.v3 解码出的任意整数类型统一转为 int。
func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case uint64:
		return int(x), true
	case float64:
		return int(x), true
	}
	return 0, false
}

// toStringOr 将值转为字符串；非字符串时返回 fallback。
func toStringOr(v any, fallback string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fallback
}

// isSensitiveKey 判断配置键是否为凭据类（口令/token/密钥等），diff 输出时应对其脱敏。
// 大小写不敏感，覆盖常见凭据后缀。
func isSensitiveKey(k string) bool {
	lk := strings.ToLower(k)
	for _, s := range []string{"password", "passwd", "pwd", "token", "secret", "key", "auth", "credential", "private_key"} {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

// valueHasCredentials 判断配置值是否含内嵌凭据的 URL（如 http://user:secret@proxy:8080），
// 是则 diff 输出也应脱敏（键名未必含敏感后缀，如 download.proxyURL）。
// 兼容无 scheme 形式（如 "user:pass@proxy:8080"）：先用 url.Parse 判定，
// 无 scheme 时再手工按首个 @ 之前是否含冒号探测。
func valueHasCredentials(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if u, err := url.Parse(s); err == nil && u.User != nil {
		return true
	}
	// 无 scheme 形式：截取 @ 前部分，含冒号即视为 userinfo。
	if at := strings.LastIndex(s, "@"); at > 0 {
		authority := s[:at]
		if strings.Contains(authority, ":") {
			// 排除无凭据的 host:port（如 "127.0.0.1:8080@x" 罕见；贪心冒号探测对
			// 形如 "a:b@c" 的判定视为凭据，可接受漏报而非误报）。
			return true
		}
	}
	return false
}

// pruneEmpty 递归删除值为空 map 的中间节点（migrate 删除叶键后留下的空父节点）。
func pruneEmpty(m map[string]any) {
	for k, v := range m {
		if child, ok := v.(map[string]any); ok {
			pruneEmpty(child)
			if len(child) == 0 {
				delete(m, k)
			}
		}
	}
}
