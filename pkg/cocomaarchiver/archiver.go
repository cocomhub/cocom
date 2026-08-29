// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cocomaarchiver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cocomhub/cocom/pkg/util"
)

type Options struct {
	ScanDir     string
	ArchiveDir  string
	NotMatchDir string
	Limit       int
	CIDRegex    *regexp.Regexp
	LookupMD5   func(ctx context.Context, cid int) (string, error)
}

type Stats struct {
	Scanned   int
	Processed int
	Archived  int
	NotMatch  int
	Errors    int
}

func defaultCIDRegex() *regexp.Regexp {
	return regexp.MustCompile(`^(\d+)\.cocoma$`)
}

func RunOnce(ctx context.Context, opts Options) (Stats, error) {
	var stats Stats
	if strings.TrimSpace(opts.ScanDir) == "" ||
		strings.TrimSpace(opts.ArchiveDir) == "" ||
		strings.TrimSpace(opts.NotMatchDir) == "" {
		return stats, errors.New("ScanDir/ArchiveDir/NotMatchDir 必填")
	}
	if opts.LookupMD5 == nil {
		return stats, errors.New("LookupMD5 必须提供")
	}
	if opts.CIDRegex == nil {
		opts.CIDRegex = defaultCIDRegex()
	}
	if opts.Limit <= 0 {
		opts.Limit = 10000
	}
	if err := os.MkdirAll(opts.ArchiveDir, 0o755); err != nil {
		return stats, err
	}
	if err := os.MkdirAll(opts.NotMatchDir, 0o755); err != nil {
		return stats, err
	}

	candidates := make([]string, 0, opts.Limit)
	err := filepath.WalkDir(opts.ScanDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if len(candidates) >= opts.Limit {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".cocoma" {
			return nil
		}
		candidates = append(candidates, path)
		return nil
	})
	if err != nil {
		return stats, err
	}
	stats.Scanned = len(candidates)
	for _, path := range candidates {
		if err := processOne(ctx, path, &opts, &stats); err != nil {
			stats.Errors++
			slog.WarnContext(ctx, "处理 cocoma 文件失败", slog.String("path", path), slog.String("err", err.Error()))
		}
	}
	return stats, nil
}

func processOne(ctx context.Context, src string, opts *Options, stats *Stats) error {
	stats.Processed++
	base := filepath.Base(src)
	m := opts.CIDRegex.FindStringSubmatch(base)
	if len(m) < 2 {
		return moveToNotMatch(ctx, src, 0, opts, "regex_no_match")
	}
	cid, err := strconv.Atoi(m[1])
	if err != nil {
		return moveToNotMatch(ctx, src, 0, opts, "cid_parse_failed")
	}

	fileMD5, err := util.FileMD5(src)
	if err != nil {
		return moveToNotMatch(ctx, src, cid, opts, "calc_md5_failed")
	}
	expectMD5, err := opts.LookupMD5(ctx, cid)
	if err != nil || strings.TrimSpace(expectMD5) == "" {
		if err != nil {
			slog.DebugContext(ctx, "查询 md5 失败", slog.Int("cid", cid), slog.String("err", err.Error()))
		}
		return moveToNotMatch(ctx, src, cid, opts, "md5_absent")
	}
	if !strings.EqualFold(strings.TrimSpace(fileMD5), strings.TrimSpace(expectMD5)) {
		return moveToNotMatch(ctx, src, cid, opts, "md5_not_match")
	}
	dest := buildArchivePath(opts.ArchiveDir, cid)
	if err := ensureDir(filepath.Dir(dest)); err != nil {
		return moveToNotMatch(ctx, src, cid, opts, "ensure_archive_dir_failed")
	}
	if exists(dest) {
		return moveToNotMatch(ctx, src, cid, opts, "archive_exists")
	}
	if err := moveFileAtomic(src, dest); err != nil {
		return moveToNotMatch(ctx, src, cid, opts, "archive_move_failed")
	}
	stats.Archived++
	slog.InfoContext(ctx, "归档成功", slog.Int("cid", cid), slog.String("src", src), slog.String("dest", dest))
	return nil
}

func moveToNotMatch(ctx context.Context, src string, cid int, opts *Options, reason string) error {
	dest := buildArchivePath(opts.NotMatchDir, cid)
	if cid == 0 {
		// 未能解析 cid，落在 notmatch 根目录
		dest = filepath.Join(opts.NotMatchDir, filepath.Base(src))
	}
	dir := filepath.Dir(dest)
	if err := ensureDir(dir); err != nil {
		return err
	}
	if exists(dest) {
		dest = uniquePath(dest)
	}
	if err := moveFileAtomic(src, dest); err != nil {
		return err
	}
	slog.InfoContext(ctx, "归档不匹配", slog.String("reason", reason), slog.String("src", src), slog.String("dest", dest))
	return nil
}

func buildArchivePath(root string, cid int) string {
	if cid <= 0 {
		return filepath.Join(root, "unknown", time.Now().Format("20060102_150405")+".cocoma")
	}
	prefix := strings.Join(util.SplitStrRightBySize(fmt.Sprintf("%04d", cid/100), 2), "/")
	return filepath.Join(root, prefix, fmt.Sprintf("%d.cocoma", cid))
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func uniquePath(dst string) string {
	ext := filepath.Ext(dst)
	name := strings.TrimSuffix(filepath.Base(dst), ext)
	dir := filepath.Dir(dst)
	for i := 1; i < 1000; i++ {
		c := filepath.Join(dir, fmt.Sprintf("%s.dup-%d%s", name, i, ext))
		if !exists(c) {
			return c
		}
	}
	return fmt.Sprintf("%s.%d", dst, time.Now().UnixNano())
}

func moveFileAtomic(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else {
		var linkErr *os.LinkError
		if errors.As(err, &linkErr) && linkErr.Err == syscall.EXDEV {
			return copyThenSwap(src, dst)
		}
		// 其他错误也尝试回退
		if errors.Is(err, os.ErrNotExist) {
			return err
		}
		return copyThenSwap(src, dst)
	}
}

func copyThenSwap(src, dst string) error {
	tmp := dst + ".tmp-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err = io.CopyBuffer(df, sf, make([]byte, 1024*1024)); err != nil {
		df.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err = df.Sync(); err != nil {
		df.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err = df.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = os.Remove(src); err != nil {
		return err
	}
	return nil
}
