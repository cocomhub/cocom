// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultArchiveSuffix = ".cocoma"
)

// RedactCmd 控制归档错误/日志中是否对命令行做密码脱敏（默认脱敏）。
// 对应配置字段 cocom.archive.redact_cmd；由 cmd/root.go 在启动期一次性注入。
// 注意：脱敏只覆盖错误/日志/HTTP 响应，argv 在 `ps` 等进程列表中的可见性属于
// 7z `-p` 机制的固有限制，无法在此层消除。
//
// 测试约定：redactCmdString 显式接收 redact 参数，测试直接传参即可，
// **不得**改写本包级变量做注入 seam（否则并发/乱序下即为 data race）。
var RedactCmd = true

// redactCmdString 返回 cmd.String() 的脱敏版本：将明文密码替换为 "***"。
// 直接替换密码本身而非 "-p"+password，可同时覆盖 cmd.String() 对含空格/特殊字符
// 参数加引号（如 "-pmy secret"）的情况。password 为空或 RedactCmd 为 false 时返回原始字符串。
// 短口令（<3 字符）整串替换会污染命令行中同字符的其它位置，改为仅替换 "-p" 后的口令片段。
func redactCmdString(cmd *exec.Cmd, password string, redact bool) string {
	raw := cmd.String()
	if password == "" || !redact {
		return raw
	}
	if len(password) < 3 {
		return redactShortPassword(raw, password)
	}
	if strings.Contains(raw, password) {
		return strings.ReplaceAll(raw, password, "***")
	}
	return raw
}

// redactShortPassword 对短口令仅替换 "-p<password>" 形态，避免整串替换误伤命令行其他字符。
// cmd.String() 对含特殊字符的密码可能加引号（如 "-pmy secret"），此处同时匹配引号内形态。
func redactShortPassword(raw, password string) string {
	// 匹配 "-p<password>" 与 -p<password> 两种形态
	for _, marker := range []string{`"-p` + password + `"`, `"-p` + password, `-p` + password} {
		if idx := strings.Index(raw, marker); idx >= 0 {
			return raw[:idx] + strings.ReplaceAll(raw[idx:], marker, "-p***")
		}
	}
	return raw
}

var regexArchiveVersion = regexp.MustCompile(`(.*)-v(\d+)\.(.*)$`)

func ParseArchiveVersion(archiveFilePath string) int {
	matches := regexArchiveVersion.FindStringSubmatch(filepath.Base(archiveFilePath))
	if len(matches) == 4 {
		version, _ := strconv.Atoi(matches[2])
		if version > 0 {
			return version
		}
	}
	return 1
}

type Type string

const (
	TypeSingle Type = "single"
	TypeDouble Type = "double"
)

type Algorithm interface {
	Type() Type
	Archive(ctx context.Context, srcDir string, destArchivePath string, cfg Config) error
	Restore(ctx context.Context, archivePath string, destDir string, cfg Config) error
}

var (
	onceSingle = sync.OnceValue(newSingle)
	onceDouble = sync.OnceValue(newDouble)

	singleAlgoConcurrency = &atomic.Int64{}
	doubleAlgoConcurrency = &atomic.Int64{}
)

func init() {
	singleAlgoConcurrency.Store(1)
	doubleAlgoConcurrency.Store(1)
}

// InitConcurrency 设置归档算法的并发数，必须在首次调用 Get() 前执行。
func InitConcurrency(single, double int) {
	if single > 0 {
		singleAlgoConcurrency.Store(int64(single))
	}
	if double > 0 {
		doubleAlgoConcurrency.Store(int64(double))
	}
}

func Get(t Type) Algorithm {
	switch t {
	case TypeDouble:
		return onceDouble()
	default:
		return onceSingle()
	}
}

func newSingle() *single {
	return &single{ch: make(chan struct{}, int(singleAlgoConcurrency.Load()))}
}

type single struct {
	ch chan struct{}
}

func (s *single) Type() Type { return TypeSingle }

func (s *single) Archive(ctx context.Context, srcDir string, destArchivePath string, cfg Config) error {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()

	// 验证输入参数
	if err := cfg.validateArchiveInput(srcDir, destArchivePath); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 生成排序后的文件列表文件
	fileListPath, err := generateSortedFileList(ctx, srcDir, cfg.TempDir, cfg.RecordFileList)
	if err != nil {
		return fmt.Errorf("生成文件列表失败: %w", err)
	}
	defer os.Remove(fileListPath) // 清理临时文件

	destArchivePathAbs, err := filepath.Abs(destArchivePath)
	if err != nil {
		return fmt.Errorf("destArchivePath[%s] 转换为绝对路径失败: %w", destArchivePath, err)
	}

	args := []string{"a", "-mhe=on", "-p" + cfg.Password}
	// 使用文件列表控制文件顺序
	args = append(args, destArchivePathAbs)
	args = append(args, "@"+fileListPath) // 使用@符号指定文件列表

	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	// 设置工作目录为源目录的父目录，以确保相对路径正确
	cmd.Dir = filepath.Dir(srcDir)
	// 7z 进度/错误回显不进服务日志（密码可能随回显暴露）
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if cmdErr := cmd.Run(); cmdErr != nil {
		return fmt.Errorf("single archive cmd[%s] err:%w", redactCmdString(cmd, cfg.Password, RedactCmd), cmdErr)
	}
	slog.DebugContext(ctx, "single archive success", slog.String("cmd", redactCmdString(cmd, cfg.Password, RedactCmd)), slog.String("dir", cmd.Dir))

	err = os.Chtimes(destArchivePath, cfg.ModTime, cfg.ModTime)
	if err != nil {
		return fmt.Errorf("update destArchivePath[%s] time[%s] failed: %w", destArchivePath, cfg.ModTime, err)
	}
	return nil
}

func (s *single) Restore(ctx context.Context, archivePath string, destDir string, cfg Config) error {
	s.ch <- struct{}{}
	defer func() { <-s.ch }()

	// 验证输入参数
	if err := cfg.validateRestoreInput(archivePath, destDir); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	args := []string{"x", "-y", "-p" + cfg.Password, "-o" + destDir, archivePath}
	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("single restore cmd[%s] err:%w", redactCmdString(cmd, cfg.Password, RedactCmd), err)
	}
	slog.DebugContext(ctx, "single restore success", slog.String("cmd", redactCmdString(cmd, cfg.Password, RedactCmd)), slog.String("dir", cmd.Dir))
	return nil
}

func newDouble() *double {
	return &double{
		single: onceSingle(),
		ch:     make(chan struct{}, int(doubleAlgoConcurrency.Load())),
	}
}

type double struct {
	ch     chan struct{}
	single *single
}

func (d *double) Type() Type { return TypeDouble }

func (d *double) Archive(ctx context.Context, srcDir string, destArchivePath string, cfg Config) error {
	d.ch <- struct{}{}
	defer func() { <-d.ch }()

	// 验证输入参数
	if err := cfg.validateArchiveInput(srcDir, destArchivePath); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	stage := destArchivePath + ".stage1"
	if err := d.single.Archive(ctx, srcDir, stage, cfg); err != nil {
		_ = os.Remove(stage) // strip 失败清理残留 stage，避免下次被误认为有效中间产物/磁盘泄漏
		return err
	}
	nestedDir := filepath.Join(filepath.Dir(destArchivePath), fmt.Sprintf("%d", cfg.ID))
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		_ = os.Remove(stage)
		return err
	}
	// 从这开始的任一失败都必须清理 nestedDir 与可能残留的 stage
	defer os.RemoveAll(nestedDir)

	nestedFile := filepath.Join(nestedDir, filepath.Base(destArchivePath))
	if err := os.Rename(stage, nestedFile); err != nil {
		_ = os.Remove(stage)
		return err
	}
	err := os.Chtimes(nestedFile, cfg.ModTime, cfg.ModTime)
	if err != nil {
		return fmt.Errorf("update nestedFile[%s] time[%s] failed: %w", nestedFile, cfg.ModTime, err)
	}
	err = os.Chtimes(nestedDir, cfg.ModTime, cfg.ModTime)
	if err != nil {
		return fmt.Errorf("update nestedDir[%s] time[%s] failed: %w", nestedDir, cfg.ModTime, err)
	}

	// 生成排序后的文件列表文件
	fileListPath, err := generateSortedFileList(ctx, nestedDir, cfg.TempDir, nil)
	if err != nil {
		return fmt.Errorf("生成文件列表失败: %w", err)
	}
	defer os.Remove(fileListPath) // 清理临时文件

	destArchivePathAbs, err := filepath.Abs(destArchivePath)
	if err != nil {
		return fmt.Errorf("destArchivePath[%s] 转换为绝对路径失败: %w", destArchivePath, err)
	}

	args := []string{"a", "-mhe=on", "-p" + cfg.Password}
	// 使用文件列表控制文件顺序
	args = append(args, destArchivePathAbs)
	args = append(args, "@"+fileListPath) // 使用@符号指定文件列表

	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	// 设置工作目录为源目录的父目录，以确保相对路径正确
	cmd.Dir = filepath.Dir(nestedDir)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if cmdErr := cmd.Run(); cmdErr != nil {
		return fmt.Errorf("double archive cmd[%s] err:%w", redactCmdString(cmd, cfg.Password, RedactCmd), cmdErr)
	}
	slog.DebugContext(ctx, "double archive success", slog.String("cmd", redactCmdString(cmd, cfg.Password, RedactCmd)), slog.String("dir", cmd.Dir))

	err = os.Chtimes(destArchivePath, cfg.ModTime, cfg.ModTime)
	if err != nil {
		return fmt.Errorf("update destArchivePath[%s] time[%s] failed: %w", destArchivePath, cfg.ModTime, err)
	}
	return os.RemoveAll(nestedDir)
}

func (d *double) Restore(ctx context.Context, archivePath string, destDir string, cfg Config) error {
	d.ch <- struct{}{}
	defer func() { <-d.ch }()

	// 验证输入参数
	if err := cfg.validateRestoreInput(archivePath, destDir); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	tmpDir, err := os.MkdirTemp(cfg.TempDir, "restore-*")
	if err != nil {
		return err
	}
	args := []string{"x", "-y", "-p" + cfg.Password, "-o" + tmpDir, archivePath}
	cmd := exec.CommandContext(ctx, cfg.CmdPath, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return fmt.Errorf("double restore cmd[%s] err:%w", redactCmdString(cmd, cfg.Password, RedactCmd), err)
	}
	slog.DebugContext(ctx, "double restore success", slog.String("cmd", redactCmdString(cmd, cfg.Password, RedactCmd)), slog.String("dir", cmd.Dir))
	nestedFile := filepath.Join(tmpDir, fmt.Sprintf("%d", cfg.ID), filepath.Base(archivePath))
	if err := d.single.Restore(ctx, nestedFile, destDir, cfg); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	return os.RemoveAll(tmpDir)
}

func getFirstFileModTime(name string) (time.Time, error) {
	stat, err := os.Stat(name)
	if err != nil {
		return time.Time{}, fmt.Errorf("os stat name[%s] err:%w", name, err)
	}
	if !stat.IsDir() {
		return stat.ModTime(), nil
	}
	files, err := os.ReadDir(name)
	if err != nil {
		return time.Time{}, fmt.Errorf("os read dir[%s] err:%w", name, err)
	}
	if len(files) == 0 {
		// 空目录不再视为错误：回退为该目录自身的修改时间。
		// （此前直接报错，只要源目录首个条目是空目录就导致归档整体失败。）
		return stat.ModTime(), nil
	}
	// 优先返回第一个「普通文件」的修改时间；若首个条目是目录（可能为空）则继续往后找，
	// 全部子目录都为空时退化为其中第一个目录的返回值，仍不可得则用本目录时间。
	var dirFallback time.Time
	for _, e := range files {
		child := filepath.Join(name, e.Name())
		childStat, statErr := os.Stat(child)
		if statErr != nil {
			return time.Time{}, fmt.Errorf("os stat name[%s] err:%w", child, statErr)
		}
		if !childStat.IsDir() {
			return childStat.ModTime(), nil
		}
		t, walkErr := getFirstFileModTime(child)
		if walkErr != nil {
			return time.Time{}, walkErr
		}
		if dirFallback.IsZero() {
			dirFallback = t
		}
	}
	if !dirFallback.IsZero() {
		return dirFallback, nil
	}
	return stat.ModTime(), nil
}

// getDirModTime TODO 完善整个目录的修改时间
func getDirModTime(dirPath string) (time.Time, error) {
	var firstModTime time.Time
	var found bool

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过符号链接
		info, err := d.Info()
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// 获取修改时间
		modTime := info.ModTime()
		if !found || modTime.After(firstModTime) {
			firstModTime = modTime
			found = true
		}
		return nil
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("遍历目录失败[%s]: %w", dirPath, err)
	}

	if !found {
		// 如果是空目录，返回当前时间
		return time.Now(), nil
	}

	return firstModTime, nil
}

// generateSortedFileList 生成排序后的文件列表
func generateSortedFileList(ctx context.Context, srcDir, tempDir string, recordFileList func(context.Context, []string) error) (string, error) {
	// 收集所有文件路径
	var files []string
	baseDir := filepath.Dir(srcDir)

	isFilePath := map[string]bool{}
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// 获取相对于源目录父目录的路径
			relPath, relErr := filepath.Rel(baseDir, path)
			if relErr != nil {
				return fmt.Errorf("获取相对路径失败: %w", relErr)
			}

			files = append(files, relPath)
			return nil
		}

		// 跳过符号链接
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// 获取相对于源目录父目录的路径
		relPath, relErr := filepath.Rel(baseDir, path)
		if relErr != nil {
			return fmt.Errorf("获取相对路径失败: %w", relErr)
		}
		// 统一使用 / 分隔符作为 map 键，与 sortFilePaths 的 \→/ 规范化保持一致，
		// 避免 Windows 下 filepath.Rel 产出的反斜杠与查询侧正斜杠失配导致 FileList 丢失。
		relPathNorm := normalizePathSeparator(relPath)
		isFilePath[relPathNorm] = true
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("遍历目录失败: %w", err)
	}

	// 排序文件路径，确保跨平台一致性
	sortFilePaths(files)

	// 创建临时文件
	tmpDir := tempDir
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}

	tmpFile, err := os.CreateTemp(tmpDir, "filelist-*.txt")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer tmpFile.Close()

	normalizedFiles := make([]string, 0, len(files))
	// 写入文件列表
	for _, file := range files {
		// 使用统一的路径分隔符，确保跨平台兼容性
		normalizedPath := normalizePathSeparator(file)
		if isFilePath[normalizedPath] {
			normalizedFiles = append(normalizedFiles, normalizedPath)
		}
		_, writeErr := tmpFile.WriteString(normalizedPath + "\n")
		if writeErr != nil {
			return "", fmt.Errorf("写入文件列表失败: %w", writeErr)
		}
	}
	if recordFileList != nil {
		if recordErr := recordFileList(ctx, normalizedFiles); recordErr != nil {
			return "", fmt.Errorf("记录文件列表失败: %w", recordErr)
		}
	}

	name := tmpFile.Name()
	name, err = filepath.Abs(name)
	if err != nil {
		return "", fmt.Errorf("转换为绝对路径失败: %w", err)
	}
	return name, nil
}

// sortFilePaths 排序文件路径并统一分隔符为 /，确保跨平台一致性。
// 仅 Windows 下将 \ 替换为 /，避免破坏 Linux 上文件名中合法的反斜杠。
func sortFilePaths(files []string) {
	if filepath.Separator == '\\' {
		for i, p := range files {
			files[i] = strings.ReplaceAll(p, "\\", "/")
		}
	}
	sort.Slice(files, func(i, j int) bool {
		a, b := files[i], files[j]

		depthA := strings.Count(a, "/")
		depthB := strings.Count(b, "/")

		if depthA != depthB {
			return depthA < depthB
		}
		return a < b
	})
}

// normalizePathSeparator 规范化路径分隔符
func normalizePathSeparator(path string) string {
	// 将路径分隔符统一为Linux风格的斜杠
	// 7z在所有平台上都能正确处理这种格式
	return strings.ReplaceAll(path, string(filepath.Separator), "/")
}
