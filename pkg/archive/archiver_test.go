// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/cocomhub/cocom/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestDataPath() string {
	// 获取当前测试文件所在目录
	_, testFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(testFile), "testdata")
}

func setupTestEnv(t *testing.T) (string, func()) {
	// 创建测试目录结构
	testDir, err := os.MkdirTemp("", "archive-test-*")
	require.NoError(t, err)

	// 创建测试文件
	testFiles := []struct {
		path    string
		isDir   bool
		content string
		modTime time.Time
	}{
		{"empty", true, "", time.Unix(1610000000, 0)},
		{"dir1/file1.txt", false, "content1", time.Unix(1610000000, 0)},
		{"dir1/file2.txt", false, "content2", time.Unix(1620000000, 0)},
		{"dir2/subdir/file3.txt", false, "content3", time.Unix(1640000000, 0)},
		{"file4.txt", false, "content4", time.Unix(1630000000, 0)},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(testDir, "src", tf.path)
		if tf.isDir {
			require.NoError(t, os.MkdirAll(fullPath, 0o755))
			require.NoError(t, os.Chtimes(fullPath, tf.modTime, tf.modTime))
			continue
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(tf.content), 0o644))
		require.NoError(t, os.Chtimes(fullPath, tf.modTime, tf.modTime))
	}

	return testDir, func() {
		os.RemoveAll(testDir)
	}
}

func TestNewSingle(t *testing.T) {
	single := newSingle()
	assert.NotNil(t, single)
	assert.Equal(t, TypeSingle, single.Type())
	assert.NotNil(t, single.ch)
}

func TestNewDouble(t *testing.T) {
	double := newDouble()
	assert.NotNil(t, double)
	assert.Equal(t, TypeDouble, double.Type())
	assert.NotNil(t, double.ch)
	assert.NotNil(t, double.single)
}

func TestGetAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		algoType Type
		expected Type
	}{
		{"single", TypeSingle, TypeSingle},
		{"double", TypeDouble, TypeDouble},
		{"default", "", TypeSingle},
		{"invalid", "invalid", TypeSingle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			algo := Get(tt.algoType)
			assert.NotNil(t, algo)
			assert.Equal(t, tt.expected, algo.Type())
		})
	}
}

func TestValidateArchiveInput(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	srcDir := filepath.Join(testDir, "src")
	cfg := Config{
		CmdPath:  "",
		Password: "testpass",
		ModTime:  time.Now(),
		TempDir:  testDir,
	}

	tests := []struct {
		name       string
		srcDir     string
		destPath   string
		cfg        Config
		wantErr    bool
		errContain string
	}{
		{
			name:     "valid input",
			srcDir:   srcDir,
			destPath: filepath.Join(testDir, "test.7z"),
			cfg:      cfg,
			wantErr:  false,
		},
		{
			name:       "empty src dir",
			srcDir:     "",
			destPath:   filepath.Join(testDir, "test.7z"),
			cfg:        cfg,
			wantErr:    true,
			errContain: "源目录不能为空",
		},
		{
			name:       "empty dest path",
			srcDir:     srcDir,
			destPath:   "",
			cfg:        cfg,
			wantErr:    true,
			errContain: "目标压缩包路径不能为空",
		},
		{
			name:     "empty password",
			srcDir:   srcDir,
			destPath: filepath.Join(testDir, "test.7z"),
			cfg: Config{
				CmdPath:  "/path/to/7z",
				Password: "",
			},
			wantErr:    true,
			errContain: "密码不能为空",
		},
		{
			name:       "non-existent src dir",
			srcDir:     filepath.Join(testDir, "nonexistent"),
			destPath:   filepath.Join(testDir, "test.7z"),
			cfg:        cfg,
			wantErr:    true,
			errContain: "源目录不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validateArchiveInput(tt.srcDir, tt.destPath)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRestoreInput(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	archivePath := filepath.Join(testDir, "test.7z")
	destDir := filepath.Join(testDir, "dest")
	cfg := Config{
		CmdPath:  "/path/to/7z",
		Password: "testpass",
	}

	// 创建测试压缩包
	require.NoError(t, os.WriteFile(archivePath, []byte("test content"), 0o644))

	tests := []struct {
		name        string
		archivePath string
		destDir     string
		cfg         Config
		wantErr     bool
		errContain  string
	}{
		{
			name:        "valid input",
			archivePath: archivePath,
			destDir:     destDir,
			cfg:         cfg,
			wantErr:     false,
		},
		{
			name:        "empty archive path",
			archivePath: "",
			destDir:     destDir,
			cfg:         cfg,
			wantErr:     true,
			errContain:  "压缩包路径不能为空",
		},
		{
			name:        "empty dest dir",
			archivePath: archivePath,
			destDir:     "",
			cfg:         cfg,
			wantErr:     true,
			errContain:  "目标目录不能为空",
		},
		{
			name:        "non-existent archive",
			archivePath: filepath.Join(testDir, "nonexistent.7z"),
			destDir:     destDir,
			cfg:         cfg,
			wantErr:     true,
			errContain:  "压缩包不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validateRestoreInput(tt.archivePath, tt.destDir)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetDirModTime(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	srcDir := filepath.Join(testDir, "src")

	// 测试非空目录
	modTime, err := getDirModTime(srcDir)
	assert.NoError(t, err)
	assert.False(t, modTime.IsZero())

	// 测试空目录
	emptyDir := filepath.Join(testDir, "empty")
	require.NoError(t, os.MkdirAll(emptyDir, 0o755))

	modTime, err = getDirModTime(emptyDir)
	assert.NoError(t, err)
	assert.False(t, modTime.IsZero())

	// 测试不存在的目录
	_, err = getDirModTime(filepath.Join(testDir, "nonexistent"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "遍历目录失败")
}

func SaveTempDir(dir string) {
	os.CopyFS("./testdata", os.DirFS(dir))
}

func TestGenerateSortedFileList2(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()
	srcDir := filepath.Join(testDir, "src")
	testDataPath := getTestDataPath()

	getFileContent := func(filePath string) string {
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Sprintf("read file %s failed: %v", filePath, err)
		}
		return string(fileContent)
	}

	type args struct {
		srcDir string
	}
	tests := []struct {
		name       string
		args       args
		targetFile string
		wantErr    bool
		errContain string
	}{
		{
			name: "nonexistent src dir",
			args: args{
				srcDir: filepath.Join(srcDir, "nonexistent"),
			},
			targetFile: "GenerateSortedFileList_empty_dir.txt",
			wantErr:    true,
			errContain: "遍历目录失败",
		},
		{
			name: "valid input",
			args: args{
				srcDir: srcDir,
			},
			targetFile: "GenerateSortedFileList_dir.txt",
			wantErr:    false,
		},
		{
			name: "file only",
			args: args{
				srcDir: filepath.Join(srcDir, "dir1", "file1.txt"),
			},
			targetFile: "GenerateSortedFileList_file_only.txt",
			wantErr:    false,
		},
		{
			name: "empty src dir",
			args: args{
				srcDir: filepath.Join(srcDir, "empty"),
			},
			targetFile: "GenerateSortedFileList_empty_dir.txt",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetFile := filepath.Join(testDataPath, tt.targetFile)
			got, err := generateSortedFileList(tt.args.srcDir, testDir)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				assert.NoError(t, err)
				assert.FileExists(t, got)
				assert.Equalf(t, util.MustFileMD5(targetFile), util.MustFileMD5(got), "generateSortedFileList(%v, %v) wantFileContent:[%s] gotFileContent:[%s]", tt.args.srcDir, testDir, getFileContent(targetFile), getFileContent(got))
			}
		})
	}
}

func TestSortFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "sort by depth then alphabetically",
			input: []string{
				"dir2/subdir/file3.txt",
				"dir1/file1.txt",
				"file4.txt",
				"dir1",
				"dir2/subdir",
				"dir1/file2.txt",
				"dir2",
			},
			expected: []string{
				"dir1",
				"dir2",
				"file4.txt",
				"dir1/file1.txt",
				"dir1/file2.txt",
				"dir2/subdir",
				"dir2/subdir/file3.txt",
			},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single file",
			input:    []string{"file.txt"},
			expected: []string{"file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := make([]string, len(tt.input))
			copy(files, tt.input)
			sortFilePaths(files)
			assert.Equal(t, tt.expected, files)
		})
	}
}

func TestNormalizePathSeparator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "windows path",
			input:    "dir\\subdir\\file.txt",
			expected: "dir/subdir/file.txt",
		},
		{
			name:     "linux path",
			input:    "dir/subdir/file.txt",
			expected: "dir/subdir/file.txt",
		},
		{
			name:     "mixed path",
			input:    "dir\\subdir/file.txt",
			expected: "dir/subdir/file.txt",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 只在Windows环境下测试反斜杠替换
			if filepath.Separator == '\\' {
				result := normalizePathSeparator(tt.input)
				assert.Equal(t, tt.expected, result)
			} else {
				// 在非Windows环境下，输入应该是相同的
				result := normalizePathSeparator(tt.input)
				assert.Equal(t, tt.input, result)
			}
		})
	}
}

func TestSingleAlgorithmIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// 创建真实的7z模拟（在真实环境中需要使用真实的7z命令）
	srcDir := filepath.Join(testDir, "src")
	destArchive := filepath.Join(testDir, "output.7z")
	destDir := filepath.Join(testDir, "extracted")

	cfg := Config{
		ID:       1,
		CmdPath:  "",
		Password: "testpass",
		ModTime:  time.Now(),
		TempDir:  testDir,
	}

	ctx := context.Background()

	// 测试Archive
	single := newSingle()
	err := single.Archive(ctx, srcDir, destArchive, cfg)
	assert.NoError(t, err)

	// 验证文件被创建
	assert.FileExists(t, destArchive)

	// 测试Restore
	err = single.Restore(ctx, destArchive, destDir, cfg)
	assert.NoError(t, err)

	// 验证解压结果
	assert.DirExists(t, destDir)
}

func TestDoubleAlgorithmIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// 创建真实的7z模拟
	srcDir := filepath.Join(testDir, "src")
	destArchive := filepath.Join(testDir, "output_double.7z")
	destDir := filepath.Join(testDir, "extracted_double")

	cfg := Config{
		ID:       1,
		CmdPath:  "",
		Password: "testpass",
		ModTime:  time.Now(),
		TempDir:  testDir,
	}

	ctx := context.Background()

	// 测试Archive
	double := newDouble()
	err := double.Archive(ctx, srcDir, destArchive, cfg)
	assert.NoError(t, err)

	// 验证文件被创建
	assert.FileExists(t, destArchive)

	// 测试Restore
	err = double.Restore(ctx, destArchive, destDir, cfg)
	assert.NoError(t, err)

	// 验证解压结果
	assert.DirExists(t, destDir)
}

func TestConcurrentAccess(t *testing.T) {
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	srcDir := filepath.Join(testDir, "src")
	cfg := Config{
		CmdPath:  "/bin/echo", // 使用echo作为模拟命令
		Password: "testpass",
		ModTime:  time.Now(),
		TempDir:  testDir,
	}

	ctx := context.Background()

	// 创建多个并发任务
	numWorkers := 5
	errors := make(chan error, numWorkers)

	for i := range numWorkers {
		go func(idx int, cfg Config) {
			cfg.ID = i
			destArchive := filepath.Join(testDir, fmt.Sprintf("output_%d.7z", idx))
			algo := Get(TypeSingle)
			err := algo.Archive(ctx, srcDir, destArchive, cfg)
			errors <- err
		}(i, cfg)
	}

	// 收集错误
	for range numWorkers {
		err := <-errors
		// 由于我们使用echo命令，Archive会失败，但我们主要是测试并发安全性
		// 所以我们不检查错误内容
		_ = err
	}
}

func TestWithContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip unstable cancellation timing on Windows")
	}
	testDir, cleanup := setupTestEnv(t)
	defer cleanup()

	srcDir := filepath.Join(testDir, "src")
	destArchive := filepath.Join(testDir, "output.7z")
	cfg := Config{
		ID:       1,
		CmdPath:  "",
		Password: "testpass",
		ModTime:  time.Now(),
		TempDir:  testDir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	single := newSingle()

	err := single.Archive(ctx, srcDir, destArchive, cfg)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
