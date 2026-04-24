// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package archivecli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/config"
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/cocomhub/cocom/pkg/storage"
	_ "github.com/cocomhub/cocom/pkg/storage/baidupcs"
	_ "github.com/cocomhub/cocom/pkg/storage/localfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Options struct {
	OutputMode      func() string
	ReplicatePrefix func(id int) string
}

func Attach(root *cobra.Command, opts Options) {
	set := commandSet{opts: opts}
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
	opts Options
}

func (c commandSet) newPackCmd() *cobra.Command {
	var srcDir string
	var destPath string
	var replicate bool
	var replicatePrefix string
	var id int
	var cid int

	cmd := &cobra.Command{
		Use:   "pack",
		Short: "打包单个 cid 并写入 archive manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveID, err := resolveArchiveID(id, cid)
			if err != nil {
				return err
			}
			resolvedSrcDir := srcDir
			if strings.TrimSpace(resolvedSrcDir) == "" {
				info, resolveErr := resolveComicInfo(cmd.Context(), archiveID)
				if resolveErr != nil {
					return fmt.Errorf("缺少必要参数：--src-dir，且无法通过 comicInfo 推导源目录: %w", resolveErr)
				}
				resolvedSrcDir = info.SaveDir()
			}
			resolvedDestPath := destPath
			if strings.TrimSpace(resolvedDestPath) == "" {
				resolvedDestPath = inferredArchivePath(archiveID)
			}
			if err := os.MkdirAll(filepath.Dir(resolvedDestPath), 0o755); err != nil {
				return err
			}
			cfg, err := archiveConfig(archiveID)
			if err != nil {
				return err
			}
			if replicatePrefix == "" && c.opts.ReplicatePrefix != nil {
				replicatePrefix = c.opts.ReplicatePrefix(archiveID)
			}
			meta, err := manager.Archive(cmd.Context(), resolvedSrcDir, resolvedDestPath, replicate, replicatePrefix, cfg)
			if err != nil {
				return err
			}
			EmitOK(cmd.OutOrStdout(), c.outputMode(), meta)
			return nil
		},
	}
	cmd.Flags().StringVar(&srcDir, "src-dir", "", "源目录")
	cmd.Flags().StringVar(&destPath, "dest-path", "", "目标归档文件路径")
	cmd.Flags().BoolVar(&replicate, "replicate", false, "是否复制到存储")
	cmd.Flags().StringVar(&replicatePrefix, "replicate-prefix", "", "复制到存储时的前缀")
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	cmd.Flags().IntVar(&cid, "cid", 0, "漫画 CID")
	return cmd
}

func (c commandSet) newUnpackCmd() *cobra.Command {
	var src string
	var out string
	var id int
	var cid int

	cmd := &cobra.Command{
		Use:   "unpack",
		Short: "解包单个归档到目标目录",
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveID, err := resolveArchiveID(id, cid)
			if err != nil && strings.TrimSpace(src) == "" {
				return errors.New("必须提供 --cid、--id 或 --src 之一")
			}
			resolvedOut := out
			if strings.TrimSpace(resolvedOut) == "" {
				if cid == 0 {
					return errors.New("缺少必要参数：--out")
				}
				info, resolveErr := resolveComicInfo(cmd.Context(), cid)
				if resolveErr != nil {
					return fmt.Errorf("缺少必要参数：--out，且无法通过 comicInfo 推导输出目录: %w", resolveErr)
				}
				resolvedOut = info.SaveDir()
			}
			archivePath := src
			algorithm := manager.Get().Algorithm()
			if strings.TrimSpace(archivePath) == "" {
				meta, getErr := manager.Get().Get(cmd.Context(), archiveID)
				if getErr != nil {
					return getErr
				}
				archivePath, getErr = archivePathFromMeta(meta)
				if getErr != nil {
					return getErr
				}
				if meta.Type != "" {
					algorithm = meta.Type
				}
			}
			if err := os.MkdirAll(resolvedOut, 0o755); err != nil {
				return err
			}
			cfg, err := archiveConfig(archiveID)
			if err != nil {
				return err
			}
			if err := archive.Get(algorithm).Restore(cmd.Context(), archivePath, resolvedOut, cfg); err != nil {
				return err
			}
			EmitOK(cmd.OutOrStdout(), c.outputMode(), map[string]any{
				"id":        archiveID,
				"cid":       archiveID,
				"src":       archivePath,
				"out":       resolvedOut,
				"algorithm": string(algorithm),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&src, "src", "", "归档文件路径")
	cmd.Flags().StringVar(&out, "out", "", "输出目录")
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	cmd.Flags().IntVar(&cid, "cid", 0, "漫画 CID")
	return cmd
}

func (c commandSet) newQueryCmd() *cobra.Command {
	var id int
	var cid int
	var name string
	var limit int

	cmd := &cobra.Command{
		Use:   "query",
		Short: "查询单个 archive 记录或过滤结果",
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveID, err := resolveArchiveID(id, cid)
			if err == nil {
				meta, getErr := manager.Get().Get(cmd.Context(), archiveID)
				if getErr != nil {
					return getErr
				}
				EmitOK(cmd.OutOrStdout(), c.outputMode(), meta)
				return nil
			}
			items, listErr := manager.Get().List(cmd.Context(), manager.IndexFilter{Name: name})
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
	cmd.Flags().IntVar(&cid, "cid", 0, "漫画 CID")
	cmd.Flags().StringVar(&name, "name", "", "名称过滤")
	cmd.Flags().IntVar(&limit, "limit", 0, "最大返回数量")
	return cmd
}

func (c commandSet) newBackupCmd() *cobra.Command {
	var id int
	var cid int
	var backend string
	var prefix string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "复制单个归档到目标存储并更新位置",
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveID, err := resolveArchiveID(id, cid)
			if err != nil {
				return err
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
			metas, err := manager.ReplicateMore(cmd.Context(), dst, resolvedPrefix, manager.IndexFilter{ID: archiveID})
			if err != nil {
				return err
			}
			meta, err := manager.Get().Get(cmd.Context(), archiveID)
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
	cmd.Flags().IntVar(&cid, "cid", 0, "漫画 CID")
	cmd.Flags().StringVar(&backend, "backend", "default-backup", "目标后端标识")
	cmd.Flags().StringVar(&prefix, "prefix", "archive/data", "目标前缀路径")
	return cmd
}

func (c commandSet) newCheckCmd() *cobra.Command {
	var id int
	var cid int
	var force bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "校验单个归档及副本健康状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			archiveID, err := resolveArchiveID(id, cid)
			if err != nil {
				return err
			}
			meta, err := manager.Check(cmd.Context(), archiveID, force)
			if err != nil {
				return err
			}
			EmitOK(cmd.OutOrStdout(), c.outputMode(), meta)
			return nil
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "归档 ID")
	cmd.Flags().IntVar(&cid, "cid", 0, "漫画 CID")
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

func resolveArchiveID(id, cid int) (int, error) {
	switch {
	case cid > 0:
		return cid, nil
	case id > 0:
		return id, nil
	default:
		return 0, errors.New("缺少必要参数：--cid 或 --id")
	}
}

func archiveConfig(id int) (archive.Config, error) {
	password := strings.TrimSpace(config.GetArchivePassword())
	if password == "" {
		return archive.Config{}, errors.New("归档密码未配置：cocom.archive.password 为空")
	}
	return archive.Config{
		ID:       id,
		CmdPath:  firstNonEmpty(config.GetArchiveCmd(), "7z"),
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
	for _, locator := range meta.Locators {
		if strings.TrimSpace(locator.Key) != "" {
			return locator.Key, nil
		}
	}
	return "", errors.New("索引缺少归档路径信息")
}

func inferredArchivePath(cid int) string {
	info := &api.ComicInfo{CID: cid}
	return filepath.Join(info.ArchiveDir(), info.ArchiveName())
}

func resolveComicInfo(ctx context.Context, cid int) (*api.ComicInfo, error) {
	if cid == 0 {
		return nil, errors.New("cid 不能为空")
	}
	coll := comicInfoCollection()
	var info api.ComicInfo
	if err := coll.FindOne(ctx, bson.M{"cid": cid}).Decode(&info); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("cid=%d 的 comicInfo 不存在", cid)
		}
		return nil, err
	}
	return &info, nil
}

func comicInfoCollection() *mongo.Collection {
	return mongowrap.DB(firstNonEmpty(
		strings.TrimSpace(viper.GetString("comic.mongo.database")),
		strings.TrimSpace(viper.GetString("mongo.database")),
		"cocom",
	)).Collection(firstNonEmpty(
		strings.TrimSpace(viper.GetString("comic.mongo.collections.comicInfo")),
		"comicInfo",
	))
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
