package imaging

import (
	"context"
	_ "image/gif"  // 注册 GIF 格式
	_ "image/jpeg" // 注册 JPEG 格式
	_ "image/png"  // 注册 PNG 格式

	_ "github.com/suixibing/cocom/pkg/imaging/webp" // 注册 WebP 格式
)

// ImageInfo 图片信息
type ImageInfo struct {
	Path    string `json:"path"`
	Format  string `json:"format"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Size    int64  `json:"size"`
	Invalid bool   `json:"invalid"`
}

// VerifyImage 验证图片完整性
func VerifyImage(ctx context.Context, path string) (*ImageInfo, error) {
	handler, err := NewImageHandler(ctx, path, "")
	if err != nil {
		return &ImageInfo{
			Path:    path,
			Invalid: true,
		}, err
	}
	return handler.GetInfo(), handler.Verify()
}

// ProcessVerify 处理验证结果
func ProcessVerify(ctx context.Context, patterns []string, opts *BatchOptions) error {
	return ProcessBatch(ctx, patterns, opts, func(h *ImageHandler) error {
		return h.Verify()
	})
}
