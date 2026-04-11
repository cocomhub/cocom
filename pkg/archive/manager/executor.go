// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/util"
)

func (h *helper) ArchiveAndRegister(ctx context.Context, srcDir, destPath string, acfg archive.Config) error {
	if srcDir == "" || destPath == "" {
		return ErrInvalidArgument
	}
	m := h.Manager()
	if err := archive.Get(m.Algorithm()).Archive(ctx, srcDir, destPath, acfg); err != nil {
		return err
	}
	st, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("获取归档文件大小失败: %w", err)
	}
	fc, err := countFiles(srcDir)
	if err != nil {
		return fmt.Errorf("统计文件数量失败: %w", err)
	}
	md5, _ := util.FileMD5(destPath)
	meta := ArchiveMeta{
		ID:        acfg.ID,
		Name:      filepath.Base(srcDir),
		Path:      destPath,
		Size:      st.Size(),
		FileCount: fc,
		ModTime:   st.ModTime(),
		Version:   1,
		Checksum: storage.Checksum{
			Algorithm: "md5",
			Value:     md5,
		},
		Locators: []storage.StorageLocator{},
		Health:   storage.NewHealthy(true),
	}
	return m.Put(ctx, meta)
}

func countFiles(dir string) (int, error) {
	c := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if !info.IsDir() {
			c++
		}
		return nil
	})
	return c, err
}
