// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package imaging

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/imaging/webp"
	"github.com/disintegration/imaging"
)

// 数值参数的安全上限，防止 NaN/Inf/负值/超大输入击穿底层库触发 panic 或 OOM。
const (
	// maxImageDimension 限制单个缩放/裁剪目标边长（像素）。
	maxImageDimension = 20000
	// maxImagePixels 限制缩放输出像素总数（约 1e8），防止 OOM。
	maxImagePixels = int64(100000000)
	// maxBlurSigma 限制模糊/锐化 sigma 上界，避免病态大半径分配。
	maxBlurSigma = 1000.0
)

// ImageProcessor 处理图片的接口
type ImageProcessor interface {
	Adjust(brightness float64, contrast float64) error
	Blur(sigma float64) error
	ConvertFormat(format string) error
	Crop(x int, y int, width int, height int) error
	Flip() error
	Flop() error
	GetInfo() *ImageInfo
	Resize(width int, height int) error
	Rotate(angle float64) error
	Save(dstPath string) error
	Sharpen(sigma float64) error
	Verify() error
}

// ImageHandler 图片处理器
type ImageHandler struct {
	ctx      context.Context
	SrcPath  string
	DstPath  string
	img      image.Image
	info     *ImageInfo
	modified bool
}

// NewImageHandler 创建新的图片处理器
func NewImageHandler(ctx context.Context, srcPath, dstPath string) (*ImageHandler, error) {
	return NewImageHandlerV2(ctx, srcPath, dstPath)
}

func NewImageHandlerV1(ctx context.Context, srcPath, dstPath string) (*ImageHandler, error) {
	// 打开文件
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, errwrap.ErrImageOpen.SetIErr(err)
	}
	defer f.Close()

	// 获取文件大小
	info, err := f.Stat()
	if err != nil {
		return nil, errwrap.ErrImageOpen.SetIErr(err)
	}

	// 解码图片配置
	config, format, err := image.DecodeConfig(f)
	if err != nil {
		return nil, errwrap.ErrImageFormat.SetIErr(err)
	}

	// 重置文件指针
	if _, seekErr := f.Seek(0, 0); seekErr != nil {
		return nil, errwrap.ErrImageOpen.SetIErr(seekErr)
	}

	// 解码图片
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, errwrap.ErrImageOpen.SetIErr(err)
	}

	imgInfo := &ImageInfo{
		Path:    srcPath,
		Format:  format,
		Width:   config.Width,
		Height:  config.Height,
		Size:    info.Size(),
		Invalid: false,
	}

	return &ImageHandler{
		ctx:     ctx,
		SrcPath: srcPath,
		DstPath: dstPath,
		img:     img,
		info:    imgInfo,
	}, nil
}

func NewImageHandlerV2(ctx context.Context, srcPath, dstPath string) (*ImageHandler, error) {
	// 打开文件并读取全部内容到内存
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, errwrap.ErrImageOpen.SetIErr(err)
	}

	// 解码图片配置
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		if strings.Contains(err.Error(), "luma/chroma subsampling ratio") {
			return nil, errwrap.ErrImageSubsampling.SetIErr(err)
		}
		return nil, errwrap.ErrImageFormat.SetIErr(err)
	}

	// 解码前先做尺寸/像素上限检查（解压炸弹防护）：DecodeConfig 只解析头部，不
	// 分配像素缓冲；若在 Decode 之前拒绝超大图像，可避免 image.Decode 全量解码
	// 造成的 OOM。min 像素上限与 Resize 输出上限一致。
	if config.Width > maxImageDimension || config.Height > maxImageDimension {
		return nil, errwrap.ErrImageFormat.SetIErrF("图像尺寸超出上限 %d: w=%d h=%d", maxImageDimension, config.Width, config.Height)
	}
	if int64(config.Width)*int64(config.Height) > maxImagePixels {
		return nil, errwrap.ErrImageFormat.SetIErrF("图像像素数超出上限 %d: w=%d h=%d", maxImagePixels, config.Width, config.Height)
	}

	// 解码图像
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errwrap.ErrImageOpen.SetIErr(err)
	}

	imgInfo := &ImageInfo{
		Path:    srcPath,
		Format:  format,
		Width:   config.Width,
		Height:  config.Height,
		Size:    int64(len(data)),
		Invalid: false,
	}

	return &ImageHandler{
		ctx:     ctx,
		SrcPath: srcPath,
		DstPath: dstPath,
		img:     img,
		info:    imgInfo,
	}, nil
}

// Resize 调整图片大小
func (h *ImageHandler) Resize(width, height int) error {
	if width < 0 || height < 0 {
		return errwrap.ErrInvalidArgs.SetIErrF("resize 宽高不能为负数: w=%d h=%d", width, height)
	}
	if width == 0 && height == 0 {
		return errwrap.ErrInvalidArgs.SetIErrF("resize 宽高不能同时为 0")
	}
	if width > maxImageDimension || height > maxImageDimension {
		return errwrap.ErrInvalidArgs.SetIErrF("resize 尺寸超出上限 %d: w=%d h=%d", maxImageDimension, width, height)
	}
	if width > 0 && height > 0 && int64(width)*int64(height) > maxImagePixels {
		return errwrap.ErrInvalidArgs.SetIErrF("resize 输出像素数超出上限 %d: w=%d h=%d", maxImagePixels, width, height)
	}
	// 等比模式（单边为 0）：由非零边推导另一边，再按推导后输出像素总数做上限
	// 检查。像素上限目的是防止缩放输出过大导致 OOM；非等比模式已在上述条件覆盖。
	// 推导用 int64 乘法，避免 int 溢出。
	var w64, h64 int64
	if width == 0 {
		w64 = int64(height) * int64(h.img.Bounds().Dx()) / int64(h.img.Bounds().Dy())
		h64 = int64(height)
	} else {
		w64 = int64(width)
		h64 = int64(width) * int64(h.img.Bounds().Dy()) / int64(h.img.Bounds().Dx())
	}
	if w64*h64 > maxImagePixels {
		return errwrap.ErrInvalidArgs.SetIErrF("resize 等比模式输出像素数超出上限 %d: w=%d h=%d", maxImagePixels, w64, h64)
	}

	h.img = imaging.Resize(h.img, width, height, imaging.Lanczos)
	h.modified = true
	slog.DebugContext(h.ctx, "调整图片大小", slog.Int("width", width), slog.Int("height", height), slog.String("path", h.SrcPath))
	return nil
}

// Crop 裁剪图片
func (h *ImageHandler) Crop(x, y, width, height int) error {
	if width <= 0 || height <= 0 {
		return errwrap.ErrInvalidArgs.SetIErrF("crop 宽高必须为正数: w=%d h=%d", width, height)
	}
	if width > maxImageDimension || height > maxImageDimension {
		return errwrap.ErrInvalidArgs.SetIErrF("crop 尺寸超出上限 %d: w=%d h=%d", maxImageDimension, width, height)
	}
	// 裁剪区域必须落在图像范围内，且 x/y 不能为负；x+width / y+height 用 int64
	// 运算避免 int 溢出回绕。越界时 imaging.Crop 不会报错而是静默裁出 0×0 区域，
	// 这里显式返回错误，避免静默生成的无效图。
	if x < 0 || y < 0 {
		return errwrap.ErrInvalidArgs.SetIErrF("crop 原点不能为负: x=%d y=%d", x, y)
	}
	bounds := h.img.Bounds()
	if int64(x)+int64(width) > int64(bounds.Dx()) || int64(y)+int64(height) > int64(bounds.Dy()) {
		return errwrap.ErrInvalidArgs.SetIErrF("crop 超出图像范围: x=%d y=%d w=%d h=%d, 图像 %dx%d", x, y, width, height, bounds.Dx(), bounds.Dy())
	}

	rect := image.Rect(x, y, x+width, y+height)
	h.img = imaging.Crop(h.img, rect)
	h.modified = true
	slog.DebugContext(h.ctx, "裁剪图片", slog.Any("rect", rect), slog.String("path", h.SrcPath))
	return nil
}

// Rotate 旋转图片
func (h *ImageHandler) Rotate(angle float64) error {
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return errwrap.ErrInvalidArgs.SetIErrF("rotate 角度必须为有限数值: %v", angle)
	}

	h.img = imaging.Rotate(h.img, angle, image.Black)
	h.modified = true
	slog.DebugContext(h.ctx, "旋转图片", slog.Float64("angle", angle), slog.String("path", h.SrcPath))
	return nil
}

// Adjust 调整亮度和对比度
func (h *ImageHandler) Adjust(brightness, contrast float64) error {
	if math.IsNaN(brightness) || math.IsInf(brightness, 0) ||
		math.IsNaN(contrast) || math.IsInf(contrast, 0) {
		return errwrap.ErrInvalidArgs.SetIErrF("adjust 亮度/对比度必须为有限数值: brightness=%v contrast=%v", brightness, contrast)
	}

	h.img = imaging.AdjustBrightness(h.img, brightness)
	h.img = imaging.AdjustContrast(h.img, contrast)
	h.modified = true
	slog.DebugContext(h.ctx, "调整亮度/对比度", slog.Float64("brightness", brightness), slog.Float64("contrast", contrast), slog.String("path", h.SrcPath))
	return nil
}

// Flip 垂直翻转
func (h *ImageHandler) Flip() error {
	h.img = imaging.FlipV(h.img)
	h.modified = true
	slog.DebugContext(h.ctx, "垂直翻转图片", slog.String("path", h.SrcPath))
	return nil
}

// Flop 水平翻转
func (h *ImageHandler) Flop() error {
	h.img = imaging.FlipH(h.img)
	h.modified = true
	slog.DebugContext(h.ctx, "水平翻转图片", slog.String("path", h.SrcPath))
	return nil
}

// Blur 模糊处理
func (h *ImageHandler) Blur(sigma float64) error {
	if sigma <= 0 || math.IsNaN(sigma) || math.IsInf(sigma, 0) {
		return errwrap.ErrInvalidArgs.SetIErrF("blur sigma 必须为 (0, %v] 内的有限数值: %v", maxBlurSigma, sigma)
	}
	if sigma > maxBlurSigma {
		return errwrap.ErrInvalidArgs.SetIErrF("blur sigma 超出上限 %v: %v", maxBlurSigma, sigma)
	}

	h.img = imaging.Blur(h.img, sigma)
	h.modified = true
	slog.DebugContext(h.ctx, "模糊处理", slog.Float64("sigma", sigma), slog.String("path", h.SrcPath))
	return nil
}

// Sharpen 锐化处理
func (h *ImageHandler) Sharpen(sigma float64) error {
	if sigma <= 0 || math.IsNaN(sigma) || math.IsInf(sigma, 0) {
		return errwrap.ErrInvalidArgs.SetIErrF("sharpen sigma 必须为 (0, %v] 内的有限数值: %v", maxBlurSigma, sigma)
	}
	if sigma > maxBlurSigma {
		return errwrap.ErrInvalidArgs.SetIErrF("sharpen sigma 超出上限 %v: %v", maxBlurSigma, sigma)
	}

	h.img = imaging.Sharpen(h.img, sigma)
	h.modified = true
	slog.DebugContext(h.ctx, "锐化处理", slog.Float64("sigma", sigma), slog.String("path", h.SrcPath))
	return nil
}

// Save 保存图片到指定路径
func (h *ImageHandler) Save(dstPath string) error {
	if !h.modified {
		slog.DebugContext(h.ctx, "图片未修改，跳过保存", slog.String("path", h.SrcPath))
		return nil
	}

	if dstPath != "" {
		h.DstPath = dstPath
	}

	ext := strings.ToLower(filepath.Ext(h.DstPath))

	// WebP 格式使用专门的处理函数
	if ext == ".webp" {
		return webp.SaveWebP(h.ctx, h.img, h.DstPath)
	}

	// 其他格式使用 imaging 库保存
	var opts []imaging.EncodeOption
	switch ext {
	case ".jpg", ".jpeg":
		opts = append(opts, imaging.JPEGQuality(100))
	case ".png":
		opts = append(opts, imaging.PNGCompressionLevel(png.DefaultCompression))
	case ".gif", ".bmp", ".tif", ".tiff":
		// 这些格式没有特殊的编码选项
	default:
		return errwrap.ErrImageFormat.SetIErrF("不支持的图片格式: %s", ext)
	}

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(h.DstPath), 0o755); err != nil {
		return errwrap.ErrImageDir.SetIErrF("创建目标目录失败: %v", err)
	}

	err := imaging.Save(h.img, h.DstPath, opts...)
	if err != nil {
		return errwrap.ErrImageSave.SetIErr(err)
	}

	// 更新图片信息
	h.info.Path = h.DstPath
	h.info.Format = strings.TrimPrefix(ext, ".")
	bounds := h.img.Bounds()
	h.info.Width = bounds.Dx()
	h.info.Height = bounds.Dy()
	if info, err := os.Stat(h.DstPath); err == nil {
		h.info.Size = info.Size()
	}

	slog.DebugContext(h.ctx, "保存图片", slog.String("path", h.DstPath), slog.String("format", ext))
	return nil
}

// ConvertFormat 转换图片格式
func (h *ImageHandler) ConvertFormat(format string) error {
	// 验证格式
	format = strings.ToLower(strings.TrimPrefix(format, "."))
	switch format {
	case "jpg", "jpeg", "png", "gif", "bmp", "tiff", "tif", "webp":
		// 更新目标文件扩展名
		ext := "." + format
		h.DstPath = strings.TrimSuffix(h.DstPath, filepath.Ext(h.DstPath)) + ext
		h.modified = true
		slog.DebugContext(h.ctx, "转换格式为", slog.String("format", format), slog.String("path", h.SrcPath))
		return nil
	default:
		return errwrap.ErrImageFormat.SetIErrF("不支持的目标格式: %s", format)
	}
}

// GetInfo 获取图片信息
func (h *ImageHandler) GetInfo() *ImageInfo {
	return h.info
}

// Verify 验证图片完整性
func (h *ImageHandler) Verify() error {
	return nil
}

// GenerateOutputPath 生成输出文件路径
func GenerateOutputPath(src, dst string, op string, params map[string]string) string {
	if dst == "" {
		dst = "."
	}

	// 如果 dst 是目录
	if info, err := os.Stat(dst); err == nil && info.IsDir() {
		base := filepath.Base(src)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)

		if params["format"] != "" {
			ext = "." + params["format"]
		}

		// 构建操作参数字符串
		var opParams []string
		if params != nil {
			// 按固定顺序处理参数
			switch op {
			case "resize", "crop":
				if x, ok := params["x"]; ok {
					opParams = append(opParams, "x"+x)
				}
				if y, ok := params["y"]; ok {
					opParams = append(opParams, "y"+y)
				}
				if w, ok := params["w"]; ok {
					opParams = append(opParams, "w"+w)
				}
				if h, ok := params["h"]; ok {
					opParams = append(opParams, "h"+h)
				}
			case "rotate":
				if angle, ok := params["angle"]; ok {
					opParams = append(opParams, "angle"+angle)
				}
			case "adjust":
				if brightness, ok := params["brightness"]; ok {
					opParams = append(opParams, "brightness"+brightness)
				}
				if contrast, ok := params["contrast"]; ok {
					opParams = append(opParams, "contrast"+contrast)
				}
			case "blur", "sharpen":
				if sigma, ok := params["sigma"]; ok {
					opParams = append(opParams, "sigma"+sigma)
				}
			}
		}

		// 生成新文件名：原名_操作_参数.扩展名
		var suffix string
		if len(opParams) > 0 {
			suffix = "_" + strings.Join(opParams, "")
		}
		newName := fmt.Sprintf("%s_%s%s%s", name, op, suffix, ext)
		return filepath.Join(dst, newName)
	}

	return dst
}

func ProcessImage(ctx context.Context, src string, dst string, op string, params map[string]string, processor func(*ImageHandler) error) (*ImageInfo, error) {
	outPath := GenerateOutputPath(src, dst, op, params)
	handler, err := NewImageHandler(ctx, src, outPath)
	if err != nil {
		return nil, err
	}
	if err := processor(handler); err != nil {
		return nil, err
	}
	if err := handler.Save(handler.DstPath); err != nil {
		return nil, err
	}
	return handler.GetInfo(), nil
}

func CreateImage(path string, img image.Image) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if filepath.Ext(path) == ".webp" {
		return webp.SaveWebP(context.Background(), img, path)
	}
	return imaging.Save(img, path)
}
