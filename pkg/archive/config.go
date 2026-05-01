// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var default7zCmdPath, _ = exec.LookPath("7z")

type Config struct {
	ID             int
	CmdPath        string
	Password       string
	TempDir        string
	RecordFileList func(context.Context, []string) error

	// ModTime 压缩时使用的时间，默认使用源目录下第一个文件的修改时间
	ModTime time.Time
}

func (c *Config) validateArchiveInput(srcDir, destArchivePath string) error {
	if srcDir == "" {
		return errors.New("源目录不能为空")
	}
	if destArchivePath == "" {
		return errors.New("目标压缩包路径不能为空")
	}

	if c.CmdPath == "" {
		if default7zCmdPath == "" {
			return errors.New("7z命令未找到")
		}
		c.CmdPath = default7zCmdPath
	}

	if c.Password == "" {
		return errors.New("密码不能为空")
	}

	if c.TempDir == "" {
		c.TempDir = os.TempDir()
	}
	if !strings.Contains(c.TempDir, "archive-data/archive") {
		c.TempDir = filepath.Join(c.TempDir, "archive-data", "archive")
	}
	if err := os.MkdirAll(c.TempDir, 0o755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 检查源目录是否存在
	if _, err := os.Stat(srcDir); err != nil {
		return fmt.Errorf("源目录不存在: %w", err)
	}

	if c.ModTime.IsZero() {
		var err error
		c.ModTime, err = getFirstFileModTime(srcDir)
		if err != nil {
			return err
		}
	}

	err := os.Chtimes(srcDir, c.ModTime, c.ModTime)
	if err != nil {
		return fmt.Errorf("update srcDir[%s] time[%s] failed: %w", srcDir, c.ModTime, err)
	}
	return nil
}

func (c *Config) validateRestoreInput(archivePath, destDir string) error {
	if archivePath == "" {
		return errors.New("压缩包路径不能为空")
	}
	if destDir == "" {
		return errors.New("目标目录不能为空")
	}

	if c.CmdPath == "" {
		if default7zCmdPath == "" {
			return errors.New("7z命令未找到")
		}
		c.CmdPath = default7zCmdPath
	}

	if c.Password == "" {
		return errors.New("密码不能为空")
	}

	if c.TempDir == "" {
		c.TempDir = os.TempDir()
	}
	if !strings.Contains(c.TempDir, "archive-data/restore") {
		c.TempDir = filepath.Join(c.TempDir, "archive-data", "restore")
	}
	if err := os.MkdirAll(c.TempDir, 0o755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 检查压缩包是否存在
	if _, err := os.Stat(archivePath); err != nil {
		return fmt.Errorf("压缩包不存在: %w", err)
	}

	return nil
}
