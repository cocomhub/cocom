// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"os"
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
  http.enable_proxy / http.proxy              ->  download.enableProxy / download.proxyURL
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
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("解析配置文件失败：%w", err)
	}

	var migrations []migration
	collect := func(oldPath, newPath string, v any) {
		migrations = append(migrations, migration{oldKey: oldPath, newKey: newPath, value: v, sensitive: isSensitiveKey(oldPath)})
	}

	// archive.* -> cocom.archive.*
	for _, key := range []string{"password", "cmd"} {
		oldPath := "archive." + key
		newPath := "cocom.archive." + key
		if v, ok := getPath(data, strings.Split(oldPath, ".")...); ok {
			setPath(data, v, strings.Split(newPath, ".")...)
			deletePath(data, strings.Split(oldPath, ".")...)
			collect(oldPath, newPath, v)
		}
	}
	if v, ok := getPath(data, "archive", "replicate"); ok {
		setPath(data, v, "cocom", "archive", "replicate")
		deletePath(data, "archive", "replicate")
		collect("archive.replicate", "cocom.archive.replicate", v)
	}
	if v, ok := getPath(data, "archive", "algorithm"); ok {
		setPath(data, v, "cocom", "archive", "algorithm")
		deletePath(data, "archive", "algorithm")
		collect("archive.algorithm", "cocom.archive.algorithm", v)
	}

	// storage.backends -> cocom.storage.backends
	if v, ok := getPath(data, "storage", "backends"); ok {
		setPath(data, v, "cocom", "storage", "backends")
		deletePath(data, "storage", "backends")
		collect("storage.backends", "cocom.storage.backends", v)
	}

	// http.* -> download.*
	if v, ok := getPath(data, "http", "enable_proxy"); ok {
		setPath(data, v, "download", "enableProxy")
		deletePath(data, "http", "enable_proxy")
		collect("http.enable_proxy", "download.enableProxy", v)
	}
	if v, ok := getPath(data, "http", "proxy"); ok {
		setPath(data, v, "download", "proxyURL")
		deletePath(data, "http", "proxy")
		collect("http.proxy", "download.proxyURL", v)
	}

	// host/port -> server.listen.http.addr（需确认拼接；仅当新键未设置时才迁移，避免覆盖现有 addr）
	existingAddr, hasAddr := getPath(data, "server", "listen", "http", "addr")
	if !hasAddr || strings.TrimSpace(toStringOr(existingAddr, "")) == "" {
		hostVal, hasHost := getPath(data, "host")
		portVal, hasPort := getPath(data, "port")
		if hasHost || hasPort {
			addr, composeErr := composeListenAddr(hostVal, hasHost, portVal, hasPort)
			if composeErr != nil {
				return composeErr
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
				collect("host/port", "server.listen.http.addr", addr)
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
		if m.sensitive {
			val = "***"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %-44s ->  %-44s = %v\n", m.oldKey, m.newKey, val)
	}

	if migrateFlags.dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "（--dry-run：未写盘）")
		return nil
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化配置失败：%w", err)
	}
	// 保留原文件权限；若文件不存在则用 0o600（配置文件含口令/token 等敏感值，仅属主可读写）
	mode := os.FileMode(0o600)
	if fi, statErr := os.Stat(cfgFile); statErr == nil {
		mode = fi.Mode().Perm()
	}
	if err := os.WriteFile(cfgFile, out, mode); err != nil {
		return fmt.Errorf("写入配置文件失败：%w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "已写入 %s，共迁移 %d 个键。\n", cfgFile, len(migrations))
	fmt.Fprintln(cmd.ErrOrStderr(), "注意：已用旧口令归档的文件不会自动可解，请确认 cocom.archive.password 与历史口令一致。")
	return nil
}

// composeListenAddr 拼接 host/port 为 server.listen.http.addr。
func composeListenAddr(hostVal any, hasHost bool, portVal any, hasPort bool) (string, error) {
	host := "0.0.0.0"
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
