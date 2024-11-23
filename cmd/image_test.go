package cmd

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suixibing/cocom/pkg/imaging"
)

func TestImageCommands(t *testing.T) {
	// 准备测试目录和文件
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "test.jpg")
	dstPath := filepath.Join(tmpDir, "output.jpg")

	// 创建测试图片
	err := createTestImage(srcPath)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "resize",
			args:    []string{"image", "resize", srcPath, dstPath, "800", "600"},
			wantErr: false,
		},
		{
			name:    "crop",
			args:    []string{"image", "crop", srcPath, dstPath, "100", "100", "400", "300"},
			wantErr: false,
		},
		{
			name:    "rotate",
			args:    []string{"image", "rotate", srcPath, dstPath, "90"},
			wantErr: false,
		},
		{
			name:    "verify",
			args:    []string{"image", "verify", srcPath},
			wantErr: false,
		},
		{
			name:    "missing source",
			args:    []string{"image", "resize", "800", "600"},
			wantErr: true,
		},
		{
			name:    "missing width height",
			args:    []string{"image", "resize", srcPath, dstPath},
			wantErr: true,
		},
		{
			name:    "invalid width",
			args:    []string{"image", "resize", srcPath, dstPath, "invalid", "600"},
			wantErr: true,
		},
		{
			name:    "invalid source",
			args:    []string{"image", "verify", "nonexistent.jpg"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.args[1] != "verify" {
					assert.FileExists(t, dstPath)
				}
			}
			// 清理输出文件
			os.Remove(dstPath)
		})
	}
}

func TestBatchProcessing(t *testing.T) {
	// 准备测试目录
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	err := os.MkdirAll(srcDir, 0o755)
	assert.NoError(t, err)

	// 创建测试图片
	for i := 0; i < 3; i++ {
		srcPath := filepath.Join(srcDir, fmt.Sprintf("test%d.jpg", i))
		err := createTestImage(srcPath)
		assert.NoError(t, err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "batch resize",
			args: []string{
				"image", "resize",
				filepath.Join(srcDir, "*.jpg"),
				dstDir, "800", "600",
				"--batch",
			},
			wantErr: false,
		},
		{
			name: "batch resize multiple sources",
			args: []string{
				"image", "resize",
				filepath.Join(srcDir, "*.jpg"),
				filepath.Join(srcDir, "*.png"),
				dstDir, "800", "600",
				"--batch",
			},
			wantErr: false,
		},
		{
			name: "batch verify",
			args: []string{
				"image", "verify",
				filepath.Join(srcDir, "*.jpg"),
				"--batch",
			},
			wantErr: false,
		},
		{
			name: "batch missing dst",
			args: []string{
				"image", "resize",
				filepath.Join(srcDir, "*.jpg"),
				"800", "600",
				"--batch",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确保目标目录存在
			err = os.MkdirAll(dstDir, 0o755)
			assert.NoError(t, err)

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !strings.Contains(tt.name, "verify") {
					files, err := filepath.Glob(filepath.Join(dstDir, "*.jpg"))
					assert.NoError(t, err)
					assert.Len(t, files, 3)
				}
			}
			// 清理输出目录
			os.RemoveAll(dstDir)
		})
	}
}

// 创建测试图片
func createTestImage(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	return imaging.CreateImage(path, img)
}

func TestImageCommands_OutputNaming(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "input.jpg")
	dstDir := filepath.Join(tmpDir, "output")

	// 创建测试图片和目标目录
	err := createTestImage(srcPath)
	assert.NoError(t, err)
	err = os.MkdirAll(dstDir, 0o755)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name: "resize to dir",
			args: []string{
				"image", "resize", srcPath, dstDir, "800", "600",
			},
			expected: filepath.Join(dstDir, "input_resize_w800h600.jpg"),
		},
		{
			name: "crop to dir",
			args: []string{
				"image", "crop", srcPath, dstDir, "100", "200", "300", "400",
			},
			expected: filepath.Join(dstDir, "input_crop_x100y200w300h400.jpg"),
		},
		{
			name: "rotate to dir",
			args: []string{
				"image", "rotate", srcPath, dstDir, "90",
			},
			expected: filepath.Join(dstDir, "input_rotate_angle90.jpg"),
		},
		{
			name: "adjust to dir",
			args: []string{
				"image", "adjust", srcPath, dstDir, "0.5", "1.5",
			},
			expected: filepath.Join(dstDir, "input_adjust_brightness0.5contrast1.5.jpg"),
		},
		{
			name: "blur to dir",
			args: []string{
				"image", "blur", srcPath, dstDir, "0.5",
			},
			expected: filepath.Join(dstDir, "input_blur_sigma0.5.jpg"),
		},
		{
			name: "sharpen to dir",
			args: []string{
				"image", "sharpen", srcPath, dstDir, "0.5",
			},
			expected: filepath.Join(dstDir, "input_sharpen_sigma0.5.jpg"),
		},
		{
			name: "flip to dir",
			args: []string{
				"image", "flip", srcPath, dstDir,
			},
			expected: filepath.Join(dstDir, "input_flip.jpg"),
		},
		{
			name: "flop to dir",
			args: []string{
				"image", "flop", srcPath, dstDir,
			},
			expected: filepath.Join(dstDir, "input_flop.jpg"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			assert.NoError(t, err)
			assert.FileExists(t, tt.expected)
		})
	}
}

func TestBatchProcessing_AllFormats(t *testing.T) {
	// 准备测试目录
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	err := os.MkdirAll(srcDir, 0o755)
	assert.NoError(t, err)

	// 创建不同格式的测试图片
	formats := []string{".jpg", ".png", ".gif", ".bmp", ".tiff"}
	for i, format := range formats {
		srcPath := filepath.Join(srcDir, fmt.Sprintf("test%d%s", i, format))
		err := createTestImage(srcPath)
		if err != nil {
			t.Logf("跳过不支持的格式: %s", format)
			continue
		}
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "batch resize all formats",
			args: []string{
				"image", "resize",
				filepath.Join(srcDir, "*.*"),
				dstDir, "800", "600",
				"--batch",
			},
			wantErr: false,
		},
		{
			name: "batch verify all formats",
			args: []string{
				"image", "verify",
				filepath.Join(srcDir, "*.*"),
				"--batch",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !strings.Contains(tt.name, "verify") {
					// 验证每种格式的输出文件
					for _, format := range formats {
						pattern := filepath.Join(dstDir, "*"+format)
						files, err := filepath.Glob(pattern)
						assert.NoError(t, err)
						if len(files) == 0 {
							t.Logf("跳过不支持的格式: %s", format)
							continue
						}
						assert.Len(t, files, 1, "格式 %s 的输出文件数量不正确", format)
					}
				}
			}
			// 清理输出目录
			os.RemoveAll(dstDir)
		})
	}
}

func TestImageFormats(t *testing.T) {
	// 准备测试目录
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")
	resultFile := filepath.Join(tmpDir, "results.json")

	err := os.MkdirAll(srcDir, 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(dstDir, 0o755)
	assert.NoError(t, err)

	// 创建不同格式的测试图片
	formats := map[string][]string{
		"jpeg": {".jpg", ".jpeg"},
		"png":  {".png"},
		"gif":  {".gif"},
		"bmp":  {".bmp"},
		"tiff": {".tif", ".tiff"},
		"webp": {".webp"},
	}

	for _, exts := range formats {
		for _, ext := range exts {
			srcPath := filepath.Join(srcDir, "test"+strings.ReplaceAll(ext, ".", "_")+ext)
			err := createTestImage(srcPath)
			assert.NoError(t, err)
		}
	}

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		checkExt string
	}{
		{
			name: "convert jpg to png",
			args: []string{
				"image", "convert",
				filepath.Join(srcDir, "test_jpg.jpg"),
				dstDir, "png",
			},
			checkExt: ".png",
		},
		{
			name: "batch convert to webp",
			args: []string{
				"image", "convert",
				filepath.Join(srcDir, "*.*"),
				dstDir, "webp",
				"--batch",
			},
			checkExt: ".webp",
		},
		{
			name: "verify all formats",
			args: []string{
				"image", "verify",
				filepath.Join(srcDir, "*.*"),
				"--batch",
				"-o", resultFile,
			},
		},
		{
			name: "invalid format",
			args: []string{
				"image", "convert",
				filepath.Join(srcDir, "test_jpg.jpg"),
				dstDir, "xyz",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkExt != "" {
					files, err := filepath.Glob(filepath.Join(dstDir, "*"+tt.checkExt))
					assert.NoError(t, err)
					if strings.Contains(tt.name, "batch") {
						assert.NotEmpty(t, files)
					} else {
						assert.Len(t, files, 1)
					}
				}
				if strings.Contains(tt.name, "verify") {
					assert.FileExists(t, resultFile)
					data, err := os.ReadFile(resultFile)
					assert.NoError(t, err)
					var results []*imaging.BatchResult
					err = json.Unmarshal(data, &results)
					assert.NoError(t, err)
					assert.NotEmpty(t, results)
					for _, result := range results {
						assert.NotNil(t, result.Image)
						assert.False(t, result.Image.Invalid)
						assert.Empty(t, result.Error)
					}
				}
			}
			// 清理输出目录
			os.RemoveAll(dstDir)
		})
	}
}
