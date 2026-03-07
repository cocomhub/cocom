// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package imaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"

	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
	"github.com/stretchr/testify/assert"
)

// createImageBytes 创建指定格式的测试图片字节流
func createImageBytes(format string) ([]byte, error) {
	var err error
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := new(bytes.Buffer)

	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(buf, img, nil)
	case "png":
		err = png.Encode(buf, img)
	case "gif":
		err = gif.Encode(buf, img, nil)
	case "bmp":
		err = bmp.Encode(buf, img)
	case "tiff":
		err = tiff.Encode(buf, img, nil)
	case "webp":
		return webp.ConvertWebP(context.Background(), img)
	default:
		err = fmt.Errorf("unsupported format: %s", format)
	}
	return buf.Bytes(), err
}

func TestVerifyImage(t *testing.T) {
	// 创建各种格式的有效图片
	formats := []string{"jpeg", "png", "gif", "bmp", "tiff", "webp"}
	validImages := make(map[string][]byte)
	for _, format := range formats {
		data, err := createImageBytes(format)
		if err != nil {
			t.Logf("跳过不支持的格式 %s: %v", format, err)
			continue
		}
		validImages[format] = data
	}

	tests := []struct {
		name     string
		filename string
		content  []byte
		wantErr  bool
	}{
		{
			name:     "valid jpeg",
			filename: "test.jpg",
			content:  validImages["jpeg"],
			wantErr:  false,
		},
		{
			name:     "valid png",
			filename: "test.png",
			content:  validImages["png"],
			wantErr:  false,
		},
		{
			name:     "valid gif",
			filename: "test.gif",
			content:  validImages["gif"],
			wantErr:  false,
		},
		{
			name:     "valid bmp",
			filename: "test.bmp",
			content:  validImages["bmp"],
			wantErr:  false,
		},
		{
			name:     "valid tiff",
			filename: "test.tiff",
			content:  validImages["tiff"],
			wantErr:  false,
		},
		{
			name:     "valid webp",
			filename: "test.webp",
			content:  validImages["webp"],
			wantErr:  false,
		},
		{
			name:     "invalid image",
			filename: "invalid.jpg",
			content:  []byte("invalid image data"),
			wantErr:  true,
		},
		{
			name:     "non-existent file",
			filename: "nonexistent.jpg",
			content:  nil,
			wantErr:  true,
		},
		{
			name:     "empty file",
			filename: "empty.jpg",
			content:  []byte{},
			wantErr:  true,
		},
		{
			name:     "wrong extension but file valid",
			filename: "test.xyz",
			content:  validImages["jpeg"],
			wantErr:  false,
		},
		{
			name:     "wrong extension and file format unknown",
			filename: "test.xyz",
			content:  []byte("unknown file format"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.HasPrefix(tt.name, "valid ") && tt.content == nil {
				t.Skipf("跳过不支持的格式测试: %s", tt.name)
			}

			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, tt.filename)

			if tt.content != nil {
				err := os.WriteFile(path, tt.content, 0o644)
				assert.NoError(t, err)
			}

			ctx := clog.NewTraceCtx("test")
			verifyResult, err := VerifyImage(ctx, path)
			t.Log("verify result:", verifyResult)
			if tt.wantErr {
				assert.Error(t, err)
				if strings.HasPrefix(tt.name, "invalid") {
					assert.Contains(t, err.Error(), "image format")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessVerify(t *testing.T) {
	// 准备测试目录
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	resultFile := filepath.Join(tmpDir, "results.json")

	err := os.MkdirAll(srcDir, 0o755)
	assert.NoError(t, err)

	// 创建测试图片
	validJPEG, err := createImageBytes("jpeg")
	assert.NoError(t, err)
	validPNG, err := createImageBytes("png")
	assert.NoError(t, err)
	invalidImage := []byte("invalid image data")

	files := map[string][]byte{
		"valid1.jpg":   validJPEG,
		"valid2.png":   validPNG,
		"invalid.jpg":  invalidImage,
		"valid3.jpg":   validJPEG,
		"invalid2.jpg": invalidImage,
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(srcDir, name), content, 0o644)
		assert.NoError(t, err)
	}

	ctx := clog.NewTraceCtx("test")
	opts := &BatchOptions{
		Workers:    2,
		Op:         "verify",
		ResultFile: resultFile,
	}

	err = ProcessVerify(ctx, []string{filepath.Join(srcDir, "*.*")}, opts)
	assert.Error(t, err) // 应该返回错误，因为有无效图片

	// 验证结果文件
	assert.FileExists(t, resultFile)
	data, err := os.ReadFile(resultFile)
	assert.NoError(t, err)

	var results []*BatchResult
	err = json.Unmarshal(data, &results)
	assert.NoError(t, err)
	assert.Len(t, results, len(files))

	// 检查结果
	validCount := 0
	invalidCount := 0
	for _, result := range results {
		if strings.Contains(result.Path, "invalid") {
			invalidCount++
			if result.Image != nil {
				assert.True(t, result.Image.Invalid)
			}
			assert.NotEmpty(t, result.Error)
		} else {
			validCount++
			assert.NotNil(t, result.Image)
			assert.False(t, result.Image.Invalid)
			assert.Empty(t, result.Error)
		}
	}

	// 验证有效和无效图片的数量
	assert.Equal(t, 3, validCount, "有效图片数量不正确")
	assert.Equal(t, 2, invalidCount, "无效图片数量不正确")
}
