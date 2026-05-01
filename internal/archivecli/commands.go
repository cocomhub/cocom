// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package archivecli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cocomhub/cocom/cmd/server/config"
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/storage"
	_ "github.com/cocomhub/cocom/pkg/storage/baidupcs"
	_ "github.com/cocomhub/cocom/pkg/storage/localfs"
	"github.com/cocomhub/cocom/pkg/util"
	"github.com/spf13/cobra"
)

type Options struct {
	GetArchiveID       func(id int) (int, error)
	OutputMode         func() string
	ReplicatePrefix    func(id int) string
	RootDir            func() string
	ArchiveSuffix      func() string
	GetSourceDir       func(ctx context.Context, id int) (string, error)
	GetArchiveFilePath func(ctx context.Context, id int) (string, error)
}

func Attach(root *cobra.Command, opts Options) {
	if opts.GetArchiveID == nil {
		opts.GetArchiveID = func(id int) (int, error) {
			if id > 0 {
				return id, nil
			}
			return 0, errors.New("缺少必要参数：--id")
		}
	}
	if opts.RootDir == nil {
		opts.RootDir = func() string {
			return ""
		}
	}
	if opts.ArchiveSuffix == nil {
		opts.ArchiveSuffix = func() string {
			return archive.DefaultArchiveSuffix
		}
	}
	if opts.GetSourceDir == nil {
		opts.GetSourceDir = func(ctx context.Context, id int) (string, error) {
			var replicatePrefix string
			if opts.ReplicatePrefix != nil {
				replicatePrefix = opts.ReplicatePrefix(id)
				if len(replicatePrefix) > 0 && replicatePrefix[0] == '/' {
					replicatePrefix = replicatePrefix[1:]
				}
			}

			return filepath.Join(opts.RootDir(), replicatePrefix, fmt.Sprintf("%d", id)), nil
		}
	}
	if opts.GetArchiveFilePath == nil {
		opts.GetArchiveFilePath = func(ctx context.Context, id int) (string, error) {
			suffix := opts.ArchiveSuffix()
			if suffix == "" {
				suffix = archive.DefaultArchiveSuffix
			}
			if !strings.HasPrefix(suffix, ".") {
				suffix = "." + suffix
			}

			meta, err := manager.Get().Get(ctx, id)
			if err != nil && !manager.IsNotFound(err) {
				return "", err
			} else if err == nil {
				archiveFilePath, err := archivePathFromMeta(meta)
				if err == nil {
					version := archive.ParseArchiveVersion(archiveFilePath)
					newArchiveFilePath := filepath.Join(filepath.Dir(archiveFilePath), fmt.Sprintf("%d-v%d%s", id, version+1, suffix))
					slog.InfoContext(ctx, "存档记录存在，基于存档文件路径生成新版本路径", "prev", archiveFilePath, "archive_path", newArchiveFilePath, "version", version+1)
					return newArchiveFilePath, nil
				}
			}

			var replicatePrefix string
			if opts.ReplicatePrefix != nil {
				replicatePrefix = opts.ReplicatePrefix(id)
			}

			archiveFilePath := filepath.Join(opts.RootDir(), "archive", replicatePrefix, fmt.Sprintf("%d%s", id, suffix))
			slog.InfoContext(ctx, "存档记录不存在，使用默认存档文件路径", "archive_path", archiveFilePath)
			return archiveFilePath, nil
		}
	}
	set := commandSet{opts: &opts}
	root.AddCommand(set.newPackCmd())
	root.AddCommand(set.newUnpackCmd())
	root.AddCommand(set.newQueryCmd())
	root.AddCommand(set.newBackupCmd())
	root.AddCommand(set.newCheckCmd())
}

func EmitError(textWriter, jsonWriter io.Writer, mode string, err error) {
	if strings.EqualFold(normalizeMode(mode), "json") {
		_ = json.NewEncoder(jsonWriter).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
			"time":  time.Now().Format(time.RFC3339),
		})
		return
	}
	_, _ = fmt.Fprintln(textWriter, "错误：", err.Error())
}

func EmitOK(writer io.Writer, mode string, value any) {
	if strings.EqualFold(normalizeMode(mode), "json") {
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"ok":   true,
			"data": value,
			"time": time.Now().Format(time.RFC3339),
		})
		return
	}
	switch typed := value.(type) {
	case manager.ArchiveMeta:
		renderArchiveMeta(writer, typed)
	case []manager.ArchiveMeta:
		tab := tabwriter.NewWriter(writer, 0, 8, 0, '\t', 0)
		_, _ = fmt.Fprintln(tab, "ID\tName\tPath\tSize\tChecksum\tHealthy")
		for _, item := range typed {
			_, _ = fmt.Fprintf(
				tab,
				"%d\t%s\t%s\t%d\t%s:%s\t%t\n",
				item.ID,
				item.Name,
				item.Path,
				item.Size,
				item.Checksum.Algorithm,
				item.Checksum.Value,
				item.ReplicaHealth.Healthy,
			)
		}
		_ = tab.Flush()
	default:
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(value)
	}
}

type commandSet struct {
	opts *Options
}

func (c commandSet) newPackCmd() *cobra.Command {
	var replicate bool
	var replicatePrefix string
	var id int

	cmd := &cobra.Command{
		Use:   "pack",
		Short: "打包归档源目录到存档文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			archiveID, err := c.opts.GetArchiveID(id)
			if err != nil {
				return fmt.Errorf("获取存档 ID 失败: %w", err)
			}
			srcDir, err := c.opts.GetSourceDir(ctx, archiveID)
			if err != nil {
				return fmt.Errorf("无法获取归档源目录: %w", err)
			}
			archiveFilePath, err := c.opts.GetArchiveFilePath(ctx, archiveID)
			if err != nil {
				return fmt.Errorf("无法获取存档文件路径: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(archiveFilePath), 0o755); err != nil {
				return err
			}
			cfg, err := archiveConfig(archiveID)
			if err != nil {
				return fmt.Errorf("无法获取归档配置: %w", err)
			}
			if replicatePrefix == "" && c.opts.ReplicatePrefix != nil {
				replicatePrefix = c.opts.ReplicatePrefix(archiveID)
			}
			meta, err := manager.Archive(ctx, srcDir, archiveFilePath, replicate, replicatePrefix, cfg)
			if err != nil {
				return fmt.Errorf("归档失败: %w", err)
			}
			EmitOK(cmd.OutOrStdout(), c.outputMode(), meta)
			return nil
		},
	}
	cmd.Flags().BoolVar(&replicate, "replicate", false, "是否复制到存储")
	cmd.Flags().StringVar(&replicatePrefix, "replicate-prefix", "", "复制到存储时的前缀")
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	return cmd
}

func (c commandSet) newUnpackCmd() *cobra.Command {
	var id int

	cmd := &cobra.Command{
		Use:   "unpack",
		Short: "解包存档文件到归档源目录",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			archiveID, err := c.opts.GetArchiveID(id)
			if err != nil {
				return fmt.Errorf("获取存档 ID 失败: %w", err)
			}
			srcDir, err := c.opts.GetSourceDir(ctx, archiveID)
			if err != nil {
				return fmt.Errorf("无法获取归档源目录: %w", err)
			}
			archiveFilePath, err := c.opts.GetArchiveFilePath(ctx, archiveID)
			if err != nil {
				return fmt.Errorf("无法获取存档文件路径: %w", err)
			}
			algorithm := manager.Get().Algorithm()
			if strings.TrimSpace(archiveFilePath) == "" {
				meta, getErr := manager.Get().Get(ctx, archiveID)
				if getErr != nil {
					return getErr
				}
				archiveFilePath, getErr = archivePathFromMeta(meta)
				if getErr != nil {
					return getErr
				}
				if meta.Type != "" {
					algorithm = meta.Type
				}
			}
			if err := os.MkdirAll(srcDir, 0o755); err != nil {
				return err
			}
			cfg, err := archiveConfig(archiveID)
			if err != nil {
				return fmt.Errorf("无法获取归档配置: %w", err)
			}
			if err := archive.Get(algorithm).Restore(ctx, archiveFilePath, filepath.Dir(srcDir), cfg); err != nil {
				return fmt.Errorf("解包失败: %w", err)
			}
			EmitOK(cmd.OutOrStdout(), c.outputMode(), map[string]any{
				"id":              archiveID,
				"srcDir":          srcDir,
				"archiveFilePath": archiveFilePath,
				"algorithm":       string(algorithm),
			})
			return nil
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	return cmd
}

func (c commandSet) newQueryCmd() *cobra.Command {
	var id int
	var name string
	var limit int

	cmd := &cobra.Command{
		Use:   "query",
		Short: "查询单个 archive 记录或过滤结果",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			archiveID, err := c.opts.GetArchiveID(id)
			if err == nil {
				meta, err := manager.Get().Get(ctx, archiveID)
				if err != nil {
					if manager.IsNotFound(err) {
						slog.InfoContext(ctx, "存档记录不存在", "id", archiveID)
						return nil
					}
					return err
				}
				EmitOK(cmd.OutOrStdout(), c.outputMode(), meta)
				return nil
			}
			items, listErr := manager.Get().List(ctx, manager.IndexFilter{Name: name})
			if listErr != nil {
				return listErr
			}
			if limit > 0 && len(items) > limit {
				items = items[:limit]
			}
			EmitOK(cmd.OutOrStdout(), c.outputMode(), items)
			return nil
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	cmd.Flags().StringVar(&name, "name", "", "名称过滤")
	cmd.Flags().IntVar(&limit, "limit", 0, "最大返回数量")
	return cmd
}

func (c commandSet) newBackupCmd() *cobra.Command {
	var id int
	var backend string
	var prefix string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "复制单个归档到目标存储并更新位置",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			archiveID, err := c.opts.GetArchiveID(id)
			if err != nil {
				return fmt.Errorf("获取存档 ID 失败: %w", err)
			}
			resolvedBackend := backend
			if strings.TrimSpace(resolvedBackend) == "" {
				resolvedBackend = "default-backup"
			}
			resolvedPrefix := prefix
			if strings.TrimSpace(resolvedPrefix) == "archive/data" {
				if c.opts.ReplicatePrefix != nil {
					resolvedPrefix = c.opts.ReplicatePrefix(archiveID)
				}
			}
			dst, ok := storage.Get(resolvedBackend)
			if !ok {
				return fmt.Errorf("目标后端 %q 未配置", resolvedBackend)
			}
			metas, err := manager.ReplicateMore(ctx, dst, resolvedPrefix, manager.IndexFilter{ID: archiveID})
			if err != nil {
				return err
			}
			meta, err := manager.Get().Get(ctx, archiveID)
			if err != nil {
				return err
			}
			EmitOK(cmd.OutOrStdout(), c.outputMode(), map[string]any{
				"replicated": metas,
				"backend":    resolvedBackend,
				"prefix":     resolvedPrefix,
				"meta":       meta,
			})
			return nil
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	cmd.Flags().StringVar(&backend, "backend", "default-backup", "目标后端标识")
	cmd.Flags().StringVar(&prefix, "prefix", "archive/data", "目标前缀路径")
	return cmd
}

func (c commandSet) newCheckCmd() *cobra.Command {
	var id int
	var force bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "校验单个归档及副本健康状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			archiveID, err := c.opts.GetArchiveID(id)
			if err != nil {
				return fmt.Errorf("获取存档 ID 失败: %w", err)
			}
			meta, err := manager.Check(ctx, archiveID, force)
			if err != nil {
				return err
			}
			EmitOK(cmd.OutOrStdout(), c.outputMode(), meta)
			return nil
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	cmd.Flags().BoolVar(&force, "force", false, "是否强制校验")
	return cmd
}

func (c commandSet) outputMode() string {
	if c.opts.OutputMode == nil {
		return "text"
	}
	return normalizeMode(c.opts.OutputMode())
}

func normalizeMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "json") {
		return "json"
	}
	return "text"
}

func archiveConfig(id int) (archive.Config, error) {
	password := strings.TrimSpace(config.GetArchivePassword())
	if password == "" {
		return archive.Config{}, errors.New("归档密码未配置：cocom.archive.password 为空")
	}
	return archive.Config{
		ID:       id,
		CmdPath:  util.FirstNonEmpty(config.GetArchiveCmd(), "7z"),
		Password: password,
		TempDir:  config.GetArchiveTempRoot(),
	}, nil
}

func archivePathFromMeta(meta *manager.ArchiveMeta) (string, error) {
	if err := meta.Validate(); err != nil {
		return "", err
	}
	if strings.TrimSpace(meta.Path) != "" {
		return meta.Path, nil
	}
	return "", errors.New("索引缺少归档路径信息")
}

func renderArchiveMeta(writer io.Writer, meta manager.ArchiveMeta) {
	_, _ = fmt.Fprintf(writer, "ID: %d\n", meta.ID)
	_, _ = fmt.Fprintf(writer, "CID: %d\n", meta.ID)
	_, _ = fmt.Fprintf(writer, "Name: %s\n", meta.Name)
	_, _ = fmt.Fprintf(writer, "Path: %s\n", meta.Path)
	_, _ = fmt.Fprintf(writer, "Size: %d\n", meta.Size)
	_, _ = fmt.Fprintf(writer, "FileCount: %d\n", meta.FileCount)
	_, _ = fmt.Fprintf(writer, "Version: %d\n", meta.Version)
	_, _ = fmt.Fprintf(writer, "Algorithm: %s\n", meta.Type)
	_, _ = fmt.Fprintf(writer, "Checksum: %s:%s\n", meta.Checksum.Algorithm, meta.Checksum.Value)
	_, _ = fmt.Fprintf(writer, "Healthy: %t\n", meta.ReplicaHealth.Healthy)
	if !meta.ReplicaHealth.CheckedAt.IsZero() {
		_, _ = fmt.Fprintf(writer, "CheckedAt: %s\n", meta.ReplicaHealth.CheckedAt.Format(time.RFC3339))
	}
	if len(meta.Locators) == 0 {
		return
	}
	tab := tabwriter.NewWriter(writer, 0, 8, 0, '\t', 0)
	_, _ = fmt.Fprintln(tab, "Backend\tKey\tHealthy\tCheckedAt")
	for _, locator := range meta.Locators {
		checkedAt := ""
		if !locator.CheckedAt.IsZero() {
			checkedAt = locator.CheckedAt.Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(
			tab,
			"%s\t%s\t%t\t%s\n",
			locator.Backend,
			locator.Key,
			locator.Healthy,
			checkedAt,
		)
	}
	_ = tab.Flush()
}
