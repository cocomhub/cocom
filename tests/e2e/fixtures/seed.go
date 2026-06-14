// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/cocomhub/cocom/cmd/server/api"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
)

// SeedE2EData 填充 E2E 测试需要的 mock 图片文件
func SeedE2EData(ctx context.Context, store *comicpkg.MemoryStorage, galleryRoot string) error {
	// 侧边栏漫画（3001, 3002, 3003）——每页 2 张图
	for _, cid := range []int{3001, 3002, 3003} {
		if err := generateComicImages(store, cid, galleryRoot, 2, func(i int) byte {
			return byte(i * 50)
		}); err != nil {
			return fmt.Errorf("sidebar cid %d: %w", cid, err)
		}
	}

	// 比对漫画（2001, 2002）——5 页
	// 前 2 页相同（MD5 匹配），后 3 页不同
	for i := 1; i <= 5; i++ {
		defaultSeed := byte(i * 50)
		if err := generateSingleImage(galleryRoot, 2001, i, defaultSeed); err != nil {
			return fmt.Errorf("cid 2001 page %d: %w", i, err)
		}
		var seed2 byte
		if i <= 2 {
			seed2 = defaultSeed
		} else {
			seed2 = byte(i*50 + 2002&0xff)
		}
		if err := generateSingleImage(galleryRoot, 2002, i, seed2); err != nil {
			return fmt.Errorf("cid 2002 page %d: %w", i, err)
		}
	}

	return nil
}

// generateComicImages 从 MemoryStorage 读取漫画信息并生成 mock 图片
func generateComicImages(store *comicpkg.MemoryStorage, cid int, galleryRoot string, numPages int, seedFn func(i int) byte) error {
	for i := 1; i <= numPages; i++ {
		if err := generateSingleImage(galleryRoot, cid, i, seedFn(i)); err != nil {
			return fmt.Errorf("generate cid %d page %d: %w", cid, i, err)
		}
	}
	return nil
}

func generateSingleImage(galleryRoot string, cid, page int, seed byte) error {
	prefix := api.StoragePrefix(cid)
	saveDir := filepath.Join(galleryRoot, prefix, fmt.Sprintf("[%d] CID%d", cid, cid))
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return err
	}
	filename := filepath.Join(saveDir, fmt.Sprintf("%d.jpg", page))
	return generateMockImage(filename, seed)
}

// generateMockImage 生成指定种子颜色的 1x1 PNG 图片
func generateMockImage(filename string, seed byte) error {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: seed, G: seed, B: seed, A: 255})
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
