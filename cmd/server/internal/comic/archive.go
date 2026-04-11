// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/config"
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
	algo := config.GetArchiveAlgorithm()
	var t archive.Type
	switch algo {
	case string(archive.TypeDouble):
		t = archive.TypeDouble
	default:
		t = archive.TypeSingle
	}
	cfg := archive.Config{
		ID:       info.CID,
		CmdPath:  cmdPath,
		Password: password,
		TempDir:  tempDir,
	}
	if err := archivemanager.ArchiveAndRegister(ctx, info.SaveDir(), tempArchivePath, cfg); err != nil {
		return err
	}

	if tempArchivePath != archivePath {
		err := os.Rename(tempArchivePath, archivePath)
		if err != nil {
			return err
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
	info.Archive = &api.ArchiveInfo{
		Path:      archivePath,
		MD5:       md5,
		Size:      stat.Size(),
		CreatedAt: time.Now(),
		Algorithm: string(t),
		ByForce:   !info.VerifyInfo.IsValid(),
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
		algo := config.GetArchiveAlgorithm()
		if algo == string(archive.TypeDouble) {
			t = archive.TypeDouble
		} else {
			t = archive.TypeSingle
		}
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
		if err = util.CopyDir(tempSaveDir, saveDirParent); err != nil {
			return fmt.Errorf("failed to copy temp directory[%s] to save parent directory[%s]: %v",
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
	info := &api.ComicInfo{}
	if err := GetComicInfo(ctx, cid, info); err != nil {
		return err
	}
	return restoreComic(ctx, info)
}
