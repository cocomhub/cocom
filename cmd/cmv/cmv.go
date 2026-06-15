// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmv

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/util"
)

var DefaultConfig = &ComicMoveManagerConfig{}

type ComicMoveManagerConfig struct {
	ComicRegexRuleRaw string
	SrcPath           string
	DstRootPath       string
	Output            string
	SkipDirs          []string
	IgnoreNotMatch    bool
	SkipFail          bool
}

func NewComicMoveManager(cfg ...*ComicMoveManagerConfig) *ComicMoveManager {
	if len(cfg) == 0 {
		cfg = append(cfg, DefaultConfig)
	}
	return &ComicMoveManager{
		ComicMoveManagerConfig: cfg[0],
	}
}

type ComicMoveManager struct {
	*ComicMoveManagerConfig

	ComicRegexRule *regexp.Regexp
}

func (m *ComicMoveManager) Handle(ctx context.Context) error {
	dirs, err := m.FindComicDirs(ctx, m.SrcPath)
	if err != nil {
		return err
	}

	return m.GenScript(dirs)
}

func (m *ComicMoveManager) ParseCID(raw string) (int64, error) {
	if m.ComicRegexRule == nil {
		var err error
		m.ComicRegexRule, err = regexp.Compile(m.ComicRegexRuleRaw)
		if err != nil {
			return 0, errwrap.New(-1, "comic rule not match").SetIErr(err)
		}
	}

	result := m.ComicRegexRule.FindStringSubmatch(raw)
	if len(result) < 2 {
		return 0, errwrap.New(-1, "comic rule not match").SetIErrF("comic rule(%s) raw(%s)", m.ComicRegexRuleRaw, raw)
	}

	cid, err := strconv.ParseInt(result[1], 10, 64)
	if err != nil {
		return 0, errwrap.New(-1, "invalid comic id").SetIErr(err)
	}
	return cid, nil
}

func (m *ComicMoveManager) FindComicDirs(ctx context.Context, src string) ([]*ComicDir, error) {
	dirs := make([]*ComicDir, 0)
	err := filepath.Walk(src, func(path string, info fs.FileInfo, walkErr error) error {
		if !info.IsDir() || info.Name() == m.SrcPath {
			return nil
		}

		slog.DebugContext(ctx, "start directory", slog.String("path", path))
		if slices.Contains(m.SkipDirs, info.Name()) {
			return filepath.SkipDir
		}

		cid, err := m.ParseCID(info.Name())
		if err != nil {
			if m.IgnoreNotMatch || m.SkipFail {
				slog.WarnContext(ctx, "skip match failed", slog.String("dir", info.Name()), slog.String("err", err.Error()))
				return nil
			}
			return err
		}

		dir := &ComicDir{
			CID:         cid,
			Name:        info.Name(),
			FullPath:    path,
			DstRootPath: m.DstRootPath,
		}

		err = dir.CheckDst()
		if err != nil {
			if m.SkipFail {
				slog.WarnContext(ctx, "skip check dir failed", slog.String("dir", info.Name()), slog.String("err", err.Error()))
				return nil
			}
			return err
		}

		dirs = append(dirs, dir)
		return nil
	})
	if err != nil {
		return nil, errwrap.New(-1, "filepath walk dir failed").SetIErr(err)
	}
	return dirs, nil
}

func (m *ComicMoveManager) GenScript(dirs []*ComicDir) error {
	var w io.Writer

	switch m.Output {
	case "stdout":
		w = os.Stdout
	case "stderr":
		w = os.Stderr
	default:
		f, err := os.OpenFile(m.Output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o777)
		if err != nil {
			return errwrap.New(-1, "open output file failed").SetIErr(err)
		}
		defer f.Close()
		w = f
	}

	return m.WriteScript(w, dirs)
}

func (m *ComicMoveManager) WriteScript(w io.Writer, dirs []*ComicDir) error {
	buf := bufio.NewWriter(w)
	_, _ = buf.WriteString("#!/bin/bash\n\nset -ex\n\n")
	for i := len(dirs) - 1; i >= 0; i-- {
		_, _ = dirs[i].WriteTo(buf)
	}
	return buf.Flush()
}

type ComicDir struct {
	CID          int64
	Name         string
	FullPath     string
	DstDir       string
	DstPath      string
	DstRootPath  string
	FlagDstExist bool
}

func (d *ComicDir) WriteTo(w io.Writer) (n int64, err error) {
	var n2 int
	if d.FlagDstExist {
		n2, err = fmt.Fprintf(w, "# exist same dir src(%s) dst(%s)\n", d.FullPath, d.DstDir)
	} else {
		fullPath := strings.ReplaceAll(d.FullPath, "\"", "\\\"")
		n2, err = fmt.Fprintf(w, "mkdir -p %s\nmv \"%s\" %s\n", d.DstDir, fullPath, d.DstDir)
	}
	n = int64(n2)
	return
}

func (d *ComicDir) dstDir() string {
	if len(d.DstPath) == 0 {
		d.DstPath = strings.Join(util.SplitStrRightBySize(fmt.Sprintf("%04d", d.CID/100), 2), "/")
	}
	return filepath.Join(d.DstRootPath, d.DstPath)
}

func (d *ComicDir) CheckDst() error {
	d.DstDir = d.dstDir()

	dstFullPath := filepath.Join(d.DstDir, d.Name)
	_, err := os.Stat(dstFullPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	err = util.IsDirSame(d.FullPath, dstFullPath)
	if err != nil {
		return err
	}
	d.FlagDstExist = true
	return nil
}
