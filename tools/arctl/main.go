// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagConfig    string
	flagIndexRoot string
	flagOutput    string
	flagVerbose   bool
)

func main() {
	root := newRootCmd()
	// 统一控制错误与 usage 输出
	root.SilenceUsage = true
	root.SilenceErrors = true
	if err := root.Execute(); err != nil {
		emitError(err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arctl",
		Short: "归档管理命令行工具",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return fmt.Errorf("初始化配置失败: %w", err)
			}
			if err := initArchiveManager(); err != nil {
				return fmt.Errorf("初始化归档管理器失败: %w", err)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&flagConfig, "config", "", "配置文件路径")
	cmd.PersistentFlags().StringVar(&flagOutput, "output", "text", "输出格式：text|json")
	cmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "启用详细日志")
	_ = viper.BindPFlag("arctl.output", cmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("arctl.verbose", cmd.PersistentFlags().Lookup("verbose"))

	cmd.AddCommand(newPackCmd())
	cmd.AddCommand(newUnpackCmd())
	cmd.AddCommand(newQueryCmd())
	cmd.AddCommand(newBackupCmd())
	cmd.AddCommand(newCheckCmd())
	return cmd
}

func initConfig() error {
	c := manager.DefaultConfig()
	viper.SetDefault("arctl.archive.manager.rootDir", c.RootDir)
	viper.SetDefault("arctl.archive.manager.algorithm", string(c.Algorithm))
	viper.SetDefault("arctl.archive.manager.index.type", "file")
	viper.SetDefault("arctl.archive.manager.index.fileStoreName", c.Index.FileStoreName)
	viper.SetDefault("arctl.archive.manager.index.fileStorePrefix", c.Index.FileStorePrefix)

	if strings.TrimSpace(flagConfig) != "" {
		viper.SetConfigFile(flagConfig)
	} else {
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv()
	return viper.ReadInConfig()
}

func initArchiveManager() error {
	if err := storage.SetFromViper(); err != nil {
		return err
	}
	if err := manager.SetFromViper("arctl.archive.manager"); err != nil {
		return err
	}
	return nil
}

func newPackCmd() *cobra.Command {
	var src string
	var dest string
	var id int
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "打包目录并注册索引",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(src) == "" || strings.TrimSpace(dest) == "" || id == 0 {
				return errors.New("缺少必要参数：--src、--dest、--id")
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			ctx := cmd.Context()
			acfg := archive.Config{
				ID:       id,
				CmdPath:  firstNonEmpty(viper.GetString("cocom.archive.cmd"), "7z"),
				Password: viper.GetString("cocom.archive.password"),
				TempDir:  viper.GetString("cocom.archive.temp_path"),
			}
			if strings.TrimSpace(acfg.Password) == "" {
				return errors.New("归档密码未配置：cocom.archive.password 为空")
			}
			if err := manager.ArchiveAndRegister(ctx, src, dest, acfg); err != nil {
				return err
			}
			m := manager.Get()
			meta, err := m.Get(ctx, id)
			if err != nil {
				return err
			}
			emitOK(meta)
			return nil
		},
	}
	cmd.Flags().StringVar(&src, "src-dir", "", "源目录（必填）")
	cmd.Flags().StringVar(&dest, "dest-path", "", "目标归档文件路径（必填）")
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID（必填）")
	return cmd
}

func newUnpackCmd() *cobra.Command {
	var src string
	var id int
	var out string
	cmd := &cobra.Command{
		Use:   "unpack",
		Short: "解包归档到目标目录",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(out) == "" {
				return errors.New("缺少必要参数：--out")
			}
			if strings.TrimSpace(src) == "" && id == 0 {
				return errors.New("必须提供 --src 或 --id 之一")
			}
			ctx := cmd.Context()
			var archivePath string
			var algo archive.Algorithm
			m := manager.Get()
			if id != 0 {
				meta, err := m.Get(ctx, id)
				if err != nil {
					return err
				}
				if strings.TrimSpace(meta.Path) != "" {
					archivePath = meta.Path
				} else if len(meta.Locators) > 0 {
					archivePath = meta.Locators[0].Key
				} else {
					return errors.New("索引缺少归档路径信息")
				}
				algo = archive.Get(m.Algorithm())
			} else {
				archivePath = src
				algo = archive.Get(m.Algorithm())
			}
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
			cfg := archive.Config{
				ID:       id,
				CmdPath:  firstNonEmpty(viper.GetString("cocom.archive.cmd"), "7z"),
				Password: viper.GetString("cocom.archive.password"),
				TempDir:  viper.GetString("cocom.archive.temp_path"),
			}
			if strings.TrimSpace(cfg.Password) == "" {
				return errors.New("归档密码未配置：cocom.archive.password 为空")
			}
			if err := algo.Restore(ctx, archivePath, out, cfg); err != nil {
				return err
			}
			emitOK(map[string]any{
				"id":        id,
				"src":       archivePath,
				"out":       out,
				"algorithm": string(m.Algorithm()),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&src, "src", "", "归档文件路径")
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	cmd.Flags().StringVar(&out, "out", "", "输出目录（必填）")
	return cmd
}

func newQueryCmd() *cobra.Command {
	var id int
	var name string
	var limit int
	cmd := &cobra.Command{
		Use:   "query",
		Short: "查询索引元数据",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			m := manager.Get()
			if id != 0 {
				meta, err := m.Get(ctx, id)
				if err != nil {
					return err
				}
				emitOK(meta)
				return nil
			}
			items, err := m.List(ctx, manager.IndexFilter{Name: name})
			if err != nil {
				return err
			}
			if limit > 0 && len(items) > limit {
				items = items[:limit]
			}
			emitOK(items)
			return nil
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	cmd.Flags().StringVar(&name, "name", "", "名称过滤")
	cmd.Flags().IntVar(&limit, "limit", 0, "最大返回数量")
	return cmd
}

func newBackupCmd() *cobra.Command {
	var id int
	var backend string
	var prefix string
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "复制到目标存储并更新索引位置",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == 0 {
				return errors.New("缺少必要参数：--id")
			}
			if strings.TrimSpace(backend) == "" {
				backend = "default-backup"
			}
			if strings.TrimSpace(prefix) == "" {
				prefix = "archive/data"
			}
			ctx := cmd.Context()
			dst, ok := storage.Get(backend)
			if !ok {
				return fmt.Errorf("目标后端 %q 未配置", backend)
			}
			n, err := manager.ReplicateToStorage(ctx, dst, prefix, manager.IndexFilter{ID: id})
			if err != nil {
				return err
			}
			emitOK(map[string]any{"replicated": n, "backend": backend, "prefix": prefix})
			return nil
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID（必填）")
	cmd.Flags().StringVar(&backend, "backend", "default-backup", "目标后端标识")
	cmd.Flags().StringVar(&prefix, "prefix", "archive/data", "目标前缀路径")
	return cmd
}

func newCheckCmd() *cobra.Command {
	var id int
	cmd := &cobra.Command{
		Use:   "check",
		Short: "校验归档一致性并更新索引健康状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == 0 {
				return errors.New("缺少必要参数：--id")
			}
			ctx := cmd.Context()
			rep, err := manager.CheckAndUpdate(ctx, id)
			if err != nil {
				return err
			}
			emitOK(rep)
			return nil
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID（必填）")
	return cmd
}

func emitError(err error) {
	mode := firstNonEmpty(flagOutput, viper.GetString("arctl.output"))
	if strings.EqualFold(mode, "json") {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
			"time":  time.Now().Format(time.RFC3339),
		})
		return
	}
	fmt.Fprintln(os.Stderr, "错误：", err.Error())
}

func emitOK(v any) {
	mode := firstNonEmpty(flagOutput, viper.GetString("arctl.output"))
	if strings.EqualFold(mode, "json") {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"ok":   true,
			"data": v,
			"time": time.Now().Format(time.RFC3339),
		})
		return
	}
	switch val := v.(type) {
	case manager.ArchiveMeta:
		fmt.Println("ID:", val.ID)
		fmt.Println("Name:", val.Name)
		fmt.Println("Path:", val.Path)
		fmt.Println("Size:", val.Size)
		fmt.Println("Version:", val.Version)
	case []manager.ArchiveMeta:
		tab := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
		defer tab.Flush()
		fmt.Fprintf(tab, "ID\tName\tPath\tSize\tVersion\n")
		for _, it := range val {
			fmt.Fprintf(tab, "%d\t%s\t%s\t%d\t%d\n", it.ID, it.Name, it.Path, it.Size, it.Version)
		}
	default:
		fmt.Printf("%v\n", v)
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
