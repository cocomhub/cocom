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

func (h *helper) Archive(ctx context.Context, srcDir, destPath string, replicate bool, replicatePrefix string, acfg archive.Config) (*ArchiveMeta, error) {
	if srcDir == "" || destPath == "" {
		return nil, ErrInvalidArgument
	}
	m := h.Manager()
	var recordFileList []string
	originRecordFileList := acfg.RecordFileList
	acfg.RecordFileList = func(ctx context.Context, files []string) error {
		recordFileList = files
		if originRecordFileList != nil {
			return originRecordFileList(ctx, files)
		}
		return nil
	}

	var history []ArchiveMeta
	oldMeta, err := m.Get(ctx, acfg.ID)
	if err != nil && !IsNotFound(err) {
		return nil, err
	} else if err == nil {
		history = oldMeta.History
		oldMeta.History = nil
		history = append(history, *oldMeta)
	}

	if archErr := archive.Get(m.Algorithm()).Archive(ctx, srcDir, destPath, acfg); archErr != nil {
		return nil, archErr
	}
	st, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("获取归档文件大小失败: %w", err)
	}
	md5, err := util.FileMD5(destPath)
	if err != nil {
		return nil, fmt.Errorf("计算归档文件MD5失败: %w", err)
	}
	meta := &ArchiveMeta{
		ID:        acfg.ID,
		Name:      filepath.Base(srcDir),
		Path:      destPath,
		Size:      st.Size(),
		FileCount: len(recordFileList),
		ModTime:   st.ModTime(),
		Version:   archive.ParseArchiveVersion(destPath),
		FileList: func() []string {
			if m.MetaRecordFileList() {
				return recordFileList
			}
			return nil
		}(),
		History: history,
		Checksum: storage.Checksum{
			Algorithm: "md5",
			Value:     md5,
		},
		Locators:      []storage.StorageLocator{},
		ReplicaHealth: storage.NewHealthy(true),
	}
	if err := m.Put(ctx, meta); err != nil {
		return nil, err
	}
	if replicate {
		for _, s := range m.Replicates() {
			if err := h.replicate(ctx, m, s, replicatePrefix, meta); err != nil {
				return nil, err
			}
		}
	}
	return meta, nil
}
