// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package imaging

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
	"github.com/stretchr/testify/assert"
)

func TestImageHandler(t *testing.T) {
	// 准备测试目录和文件
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "test.jpg")
	dstPath := filepath.Join(tmpDir, "test_out.jpg")

	// 创建测试图片
	err := createTestImage(srcPath)
	assert.NoError(t, err)

	ctx := clog.NewTraceCtx("test")
	handler, err := NewImageHandler(ctx, srcPath, dstPath)
	assert.NoError(t, err)
	assert.NotNil(t, handler)

	// 测试调整大小
	err = handler.Resize(100, 100)
	assert.NoError(t, err)

	// 测试裁剪
	err = handler.Crop(0, 0, 50, 50)
	assert.NoError(t, err)

	// 测试旋转
	err = handler.Rotate(90)
	assert.NoError(t, err)

	// 测试亮度和对比度
	err = handler.Adjust(0.5, 1.5)
	assert.NoError(t, err)

	// 测试保存
	err = handler.Save(dstPath)
	assert.NoError(t, err)
	assert.FileExists(t, dstPath)
}

func TestBatchProcessor(t *testing.T) {
	// 准备测试目录
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	err := os.MkdirAll(srcDir, 0o755)
	assert.NoError(t, err)

	err = os.MkdirAll(dstDir, 0o755)
	assert.NoError(t, err)

	// 创建测试图片
	var files []string
	for i := range 3 {
		srcPath := filepath.Join(srcDir, fmt.Sprintf("test%d.jpg", i))
		files = append(files, srcPath)
		err := createTestImage(srcPath)
		assert.NoError(t, err)
	}

	ctx := clog.NewTraceCtx("test")
	batch := NewBatchProcessor(ctx, &BatchOptions{
		DstDir:  dstDir,
		Workers: 2,
		Op:      "resize",
		Params: map[string]string{
			"w": "100",
			"h": "200",
		},
	})
	batch.files = files

	// 测试批量调整大小
	err = batch.Process(func(h *ImageHandler) error {
		if err := h.Resize(100, 200); err != nil {
			return err
		}
		return h.Save(h.DstPath)
	})
	assert.NoError(t, err)

	// 验证输出文件
	files, err = filepath.Glob(filepath.Join(dstDir, "*.jpg"))
	assert.NoError(t, err)
	assert.Len(t, files, 3)
}

// 创建测试图片
func createTestImage(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	return CreateImage(path, img)
}

func TestImageHandler_AllFormats(t *testing.T) {
	tmpDir := t.TempDir()
	formats := map[string][]string{
		"jpeg": {".jpg", ".jpeg"},
		"png":  {".png"},
		"gif":  {".gif"},
		"bmp":  {".bmp"},
		"tiff": {".tif", ".tiff"},
		"webp": {".webp"},
	}

	ctx := clog.NewTraceCtx("test")
	for format, exts := range formats {
		for _, ext := range exts {
			t.Run(format+ext, func(t *testing.T) {
				srcPath := filepath.Join(tmpDir, "test"+ext)
				dstDir := filepath.Join(tmpDir, "output")
				err := os.MkdirAll(dstDir, 0o755)
				assert.NoError(t, err)

				// 创建源图片
				err = createTestImage(srcPath)
				if err != nil {
					t.Skipf("跳过不支持的格式: %s", ext)
					return
				}

				// 测试处理和保存
				handler, err := NewImageHandler(ctx, srcPath, GenerateOutputPath(srcPath, dstDir, "resize", map[string]string{
					"w": "100",
					"h": "100",
				}))
				if err != nil {
					t.Skipf("跳过不支持的格式: %s", ext)
					return
				}

				err = handler.Resize(100, 100)
				assert.NoError(t, err)

				err = handler.Save(handler.DstPath)
				assert.NoError(t, err)

				// 验证输出文件
				expectedPath := filepath.Join(dstDir, fmt.Sprintf("test_resize_w100h100%s", ext))
				assert.FileExists(t, expectedPath)
			})
		}
	}
}

func TestImageHandler_OutputNaming(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "input.jpg")
	dstDir := filepath.Join(tmpDir, "output")

	// 创建源图片
	err := createTestImage(srcPath)
	assert.NoError(t, err)

	err = os.MkdirAll(dstDir, 0o755)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		op       string
		params   map[string]string
		expected string
	}{
		{
			name: "resize",
			op:   "resize",
			params: map[string]string{
				"w": "800",
				"h": "600",
			},
			expected: "input_resize_w800h600.jpg",
		},
		{
			name: "crop",
			op:   "crop",
			params: map[string]string{
				"x": "100",
				"y": "100",
				"w": "400",
				"h": "300",
			},
			expected: "input_crop_x100y100w400h300.jpg",
		},
		{
			name: "rotate",
			op:   "rotate",
			params: map[string]string{
				"angle": "90",
			},
			expected: "input_rotate_angle90.jpg",
		},
		{
			name: "adjust",
			op:   "adjust",
			params: map[string]string{
				"brightness": "0.5",
				"contrast":   "1.5",
			},
			expected: "input_adjust_brightness0.5contrast1.5.jpg",
		},
		{
			name: "blur",
			op:   "blur",
			params: map[string]string{
				"sigma": "0.5",
			},
			expected: "input_blur_sigma0.5.jpg",
		},
		{
			name: "sharpen",
			op:   "sharpen",
			params: map[string]string{
				"sigma": "0.5",
			},
			expected: "input_sharpen_sigma0.5.jpg",
		},
		{
			name:     "flip",
			op:       "flip",
			params:   nil,
			expected: "input_flip.jpg",
		},
		{
			name:     "flop",
			op:       "flop",
			params:   nil,
			expected: "input_flop.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outPath := GenerateOutputPath(srcPath, dstDir, tt.op, tt.params)
			assert.Equal(t, filepath.Join(dstDir, tt.expected), outPath)
		})
	}
}

func TestBatchProcessor_MultipleFormats(t *testing.T) {
	// 准备测试目录
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	err := os.MkdirAll(srcDir, 0o755)
	assert.NoError(t, err)

	// 创建不同格式的测试图片
	formats := []string{".jpg", ".png", ".gif", ".bmp", ".tiff"}

	// 检查是否安装了 WebP 工具
	if webp.HasWebPUtil() {
		formats = append(formats, ".webp")
	}

	var files []string
	for i, format := range formats {
		srcPath := filepath.Join(srcDir, fmt.Sprintf("test%d%s", i, format))
		files = append(files, srcPath)
		err := createTestImage(srcPath)
		assert.NoError(t, err)
	}

	ctx := clog.NewTraceCtx("test")
	batch := NewBatchProcessor(ctx, &BatchOptions{
		DstDir:  dstDir,
		Workers: 2,
		Op:      "resize",
	})
	batch.files = files

	// 测试批量调整大小
	err = batch.Process(func(h *ImageHandler) error {
		if err := h.Resize(100, 100); err != nil {
			return err
		}
		return h.Save(h.DstPath)
	})
	assert.NoError(t, err)

	// 验证每种格式的输出文件
	for _, format := range formats {
		pattern := filepath.Join(dstDir, "*"+format)
		files, err := filepath.Glob(pattern)
		assert.NoError(t, err)
		assert.Len(t, files, 1, "格式 %s 的输出文件数量不正确", format)
	}
}

func TestGenerateOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "input.jpg")
	dstDir := filepath.Join(tmpDir, "output")

	err := os.MkdirAll(dstDir, 0o755)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		op       string
		params   map[string]string
		expected string
	}{
		{
			name: "resize",
			op:   "resize",
			params: map[string]string{
				"w": "800",
				"h": "600",
			},
			expected: "input_resize_w800h600.jpg",
		},
		{
			name: "crop",
			op:   "crop",
			params: map[string]string{
				"x": "100",
				"y": "100",
				"w": "400",
				"h": "300",
			},
			expected: "input_crop_x100y100w400h300.jpg",
		},
		{
			name: "rotate",
			op:   "rotate",
			params: map[string]string{
				"angle": "90",
			},
			expected: "input_rotate_angle90.jpg",
		},
		{
			name: "adjust",
			op:   "adjust",
			params: map[string]string{
				"brightness": "0.5",
				"contrast":   "1.5",
			},
			expected: "input_adjust_brightness0.5contrast1.5.jpg",
		},
		{
			name: "blur",
			op:   "blur",
			params: map[string]string{
				"sigma": "0.5",
			},
			expected: "input_blur_sigma0.5.jpg",
		},
		{
			name: "sharpen",
			op:   "sharpen",
			params: map[string]string{
				"sigma": "0.5",
			},
			expected: "input_sharpen_sigma0.5.jpg",
		},
		{
			name:     "flip",
			op:       "flip",
			params:   nil,
			expected: "input_flip.jpg",
		},
		{
			name:     "flop",
			op:       "flop",
			params:   nil,
			expected: "input_flop.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outPath := GenerateOutputPath(srcPath, dstDir, tt.op, tt.params)
			assert.Equal(t, filepath.Join(dstDir, tt.expected), outPath)
		})
	}
}

func TestImageHandler_ConvertFormat(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "test.jpg")
	formats := []string{
		"jpg", "jpeg", "png", "gif", "bmp", "tiff",
	}

	// 检查是否安装了 WebP 工具
	if webp.HasWebPUtil() {
		formats = append(formats, "webp")
	}

	// 创建源图片
	err := createTestImage(srcPath)
	assert.NoError(t, err)

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			dstPath := filepath.Join(tmpDir, "out."+format)
			ctx := clog.NewTraceCtx("test")

			handler, err := NewImageHandler(ctx, srcPath, dstPath)
			if err != nil {
				t.Skipf("跳过不支持的格式: %s", format)
				return
			}

			err = handler.ConvertFormat(format)
			assert.NoError(t, err)

			err = handler.Save(handler.DstPath)
			if err != nil {
				t.Skipf("跳过不支持的格式: %s", format)
				return
			}

			// 验证输出文件
			assert.FileExists(t, dstPath)
			_, err = VerifyImage(ctx, dstPath)
			assert.NoError(t, err)
		})
	}

	// 测试无效格式
	t.Run("invalid format", func(t *testing.T) {
		handler, err := NewImageHandler(context.Background(), srcPath, "out.xyz")
		assert.NoError(t, err)
		err = handler.ConvertFormat("xyz")
		assert.Error(t, err)
	})
}

func TestImageHandler_AllOperations_AllFormats(t *testing.T) {
	// 准备测试目录
	tmpDir := t.TempDir()
	formats := []struct {
		name string
		exts []string
	}{
		{"JPEG", []string{".jpg", ".jpeg"}},
		{"PNG", []string{".png"}},
		{"GIF", []string{".gif"}},
		{"BMP", []string{".bmp"}},
		{"TIFF", []string{".tif", ".tiff"}},
		{"WebP", []string{".webp"}},
	}

	operations := []struct {
		name     string
		process  func(*ImageHandler) error
		validate func(*testing.T, *ImageInfo)
	}{
		{
			name: "resize",
			process: func(h *ImageHandler) error {
				return h.Resize(100, 100)
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.Equal(t, 100, info.Width)
				assert.Equal(t, 100, info.Height)
			},
		},
		{
			name: "crop",
			process: func(h *ImageHandler) error {
				return h.Crop(10, 10, 50, 50)
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.Equal(t, 50, info.Width)
				assert.Equal(t, 50, info.Height)
			},
		},
		{
			name: "rotate",
			process: func(h *ImageHandler) error {
				return h.Rotate(90)
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.Equal(t, info.Height, info.Width)
			},
		},
		{
			name: "adjust",
			process: func(h *ImageHandler) error {
				return h.Adjust(0.5, 1.5)
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.False(t, info.Invalid)
			},
		},
		{
			name: "flip",
			process: func(h *ImageHandler) error {
				return h.Flip()
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.False(t, info.Invalid)
			},
		},
		{
			name: "flop",
			process: func(h *ImageHandler) error {
				return h.Flop()
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.False(t, info.Invalid)
			},
		},
		{
			name: "blur",
			process: func(h *ImageHandler) error {
				return h.Blur(0.5)
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.False(t, info.Invalid)
			},
		},
		{
			name: "sharpen",
			process: func(h *ImageHandler) error {
				return h.Sharpen(0.5)
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.False(t, info.Invalid)
			},
		},
		{
			name: "convert_to_jpg",
			process: func(h *ImageHandler) error {
				return h.ConvertFormat("jpg")
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.Equal(t, "jpeg", info.Format)
			},
		},
		{
			name: "convert_to_png",
			process: func(h *ImageHandler) error {
				return h.ConvertFormat("png")
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.Equal(t, "png", info.Format)
			},
		},
		{
			name: "convert_to_webp",
			process: func(h *ImageHandler) error {
				return h.ConvertFormat("webp")
			},
			validate: func(t *testing.T, info *ImageInfo) {
				assert.Equal(t, "webp", info.Format)
			},
		},
	}

	ctx := clog.NewTraceCtx("test")

	// 对每种格式测试所有操作
	for _, format := range formats {
		for _, ext := range format.exts {
			for _, op := range operations {
				t.Run(fmt.Sprintf("%s_%s_%s", format.name, ext, op.name), func(t *testing.T) {
					srcPath := filepath.Join(tmpDir, "test"+ext)
					dstPath := filepath.Join(tmpDir, "out"+ext)

					// 创建源图片
					err := createTestImage(srcPath)
					if err != nil {
						t.Skipf("跳过不支持的格式: %s", ext)
						return
					}

					// 创建处理器
					handler, err := NewImageHandler(ctx, srcPath, dstPath)
					if err != nil {
						t.Skipf("跳过不支持的格式: %s", ext)
						return
					}

					// 执行操作
					err = op.process(handler)
					assert.NoError(t, err)

					// 保存结果
					err = handler.Save(handler.DstPath)
					assert.NoError(t, err)

					// 验证结果
					info, err := VerifyImage(ctx, handler.DstPath)
					assert.NoError(t, err)
					op.validate(t, info)

					// 验证结果文件可以正常打开和解码
					_, err = NewImageHandler(ctx, handler.DstPath, "")
					assert.NoError(t, err)
				})
			}
		}
	}
}
