// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/archive"
	archivemanager "github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/comic"
	"github.com/cocomhub/cocom/pkg/util"
)

func archiveComic(ctx context.Context, info *api.ComicInfo, force bool) error {
	if info == nil {
		return nil
	}
	if !force && !info.VerifyInfo.IsValid() {
		return nil
	}

	// 幂等：若目标已存在且 MD5 一致，直接返回并补全元信息
	if info.Archive != nil && info.Archive.Path != "" && info.Archive.MD5 != "" {
		if st, err := os.Stat(info.Archive.Path); err == nil && !st.IsDir() {
			if md5, mErr := util.FileMD5(info.Archive.Path); mErr == nil && strings.EqualFold(md5, info.Archive.MD5) {
				return nil
			}
		}
	}

	password := config.GetArchivePassword()
	if password == "" {
		return fmt.Errorf("archive password is empty")
	}

	if err := os.MkdirAll(info.ArchiveDir(), 0o755); err != nil {
		return err
	}

	tempDir := info.ArchiveTempDir()
	archivePath := filepath.Join(info.ArchiveDir(), info.ArchiveName())
	tempArchivePath := filepath.Join(tempDir, info.ArchiveName())
	cmdPath := config.GetArchiveCmd()
	cfg := archive.Config{
		ID:       info.CID,
		CmdPath:  cmdPath,
		Password: password,
		TempDir:  tempDir,
	}
	meta, err := archivemanager.Archive(ctx, info.SaveDir(), tempArchivePath, config.GetArchiveReplicate(), info.StoragePrefix(), cfg)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("archive meta is nil")
	}

	if tempArchivePath != archivePath {
		if err = util.Move(tempArchivePath, archivePath); err != nil {
			return fmt.Errorf("failed to move archive directory[%s] to [%s]: %v",
				tempArchivePath, archivePath, err)
		}
	}

	stat, err := os.Stat(archivePath)
	if err != nil {
		return err
	}
	md5, err := util.FileMD5(archivePath)
	if err != nil {
		return err
	}
	info.Archive, err = archivemanager.ArchiveMeta2CocomArchiveInfo(meta)
	if err != nil {
		return err
	}
	if info.Archive.Size != stat.Size() {
		slog.Error("archive size mismatch", "expected", meta.Size, "actual", stat.Size())
		info.Archive.ReplicaHealth.Healthy = false
	} else {
		info.Archive.Path = archivePath
	}
	info.Archive.ByForce = !info.VerifyInfo.IsValid()

	if meta.Checksum.Algorithm == "md5" && meta.Checksum.Value != md5 {
		slog.Error("archive md5 mismatch", "expected", meta.Checksum.Value, "actual", md5)
		info.Archive.ReplicaHealth.Healthy = false
	} else {
		info.Archive.MD5 = md5
	}
	return nil
}

func restoreComic(ctx context.Context, info *api.ComicInfo) error {
	if info == nil {
		return nil
	}
	if info.Archive == nil || info.Archive.Path == "" {
		return nil
	}

	if info.Archive != nil && info.Archive.Path != "" && info.Archive.MD5 != "" {
		if st, err := os.Stat(info.Archive.Path); err == nil && !st.IsDir() {
			if md5, mErr := util.FileMD5(info.Archive.Path); mErr == nil && !strings.EqualFold(md5, info.Archive.MD5) {
				return &comic.ArchiveMD5MismatchError{Expected: info.Archive.MD5, Actual: md5}
			}
		}
	}

	password := config.GetArchivePassword()
	if password == "" {
		return fmt.Errorf("archive password not found")
	}

	saveDir := info.SaveDir()
	saveDirParent := filepath.Dir(saveDir)
	if err := os.MkdirAll(saveDirParent, 0o755); err != nil {
		return err
	}
	cmdPath := config.GetArchiveCmd()
	var t archive.Type
	if info.Archive.Algorithm == string(archive.TypeDouble) {
		t = archive.TypeDouble
	} else if info.Archive.Algorithm == string(archive.TypeSingle) {
		t = archive.TypeSingle
	} else {
		return fmt.Errorf("archive algorithm %s not supported", info.Archive.Algorithm)
	}
	tempDir := info.ArchiveTempDir()
	cfg := archive.Config{
		ID:       info.CID,
		CmdPath:  cmdPath,
		Password: password,
		TempDir:  tempDir,
	}
	err := archive.Get(t).Restore(ctx, info.Archive.Path, tempDir, cfg)
	if err != nil {
		return err
	}
	if tempDir != saveDirParent {
		tempSaveDir := filepath.Join(tempDir, filepath.Base(saveDir))
		if err = util.Move(tempSaveDir, saveDirParent); err != nil {
			return fmt.Errorf("failed to move temp directory[%s] to save parent directory[%s]: %v",
				tempSaveDir, saveDirParent, err)
		}
		if err := os.RemoveAll(tempSaveDir); err != nil {
			slog.ErrorContext(ctx, "restoreComic RemoveAll dir err", slog.String("dir", tempSaveDir), slog.String("err", err.Error()))
		} else {
			slog.DebugContext(ctx, "restoreComic RemoveAll dir succ", slog.String("dir", tempSaveDir))
		}
	}
	return nil
}

func RestoreComicByID(ctx context.Context, cid int) error {
	if s := GetDefaultStorage(); s != nil {
		return s.RestoreByID(ctx, strconv.Itoa(cid))
	}
	info := &api.ComicInfo{}
	if err := GetComicInfo(ctx, cid, info); err != nil {
		return err
	}
	return restoreComic(ctx, info)
}
