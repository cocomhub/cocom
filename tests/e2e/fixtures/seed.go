// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/cocomhub/cocom/cmd/server/api"
	comicpkg "github.com/cocomhub/cocom/pkg/comic"
)

// seedSpec 描述一部需要生成 mock 图片的漫画
// 2001/2002 前 2 页相同（MD5 匹配），其余页不同以保留对比语义。
type seedSpec struct {
	info   *api.ComicInfo
	pages  int
	seedFn func(i int) byte
}

// SeedE2EData 填充 E2E 测试需要的 mock 图片文件
// 图片目录名与生产写入路径一致（SaveDirName()），扩展名 jpg（模板/保存逻辑为 .jpg）。
func SeedE2EData(ctx context.Context, store *comicpkg.MemoryStorage, galleryRoot string) error {
	specs, err := collectSeedSpecs(ctx, store)
	if err != nil {
		return err
	}
	for _, sp := range specs {
		if err := generateSingleImageByInfo(sp.info, galleryRoot, sp.pages, sp.seedFn); err != nil {
			return fmt.Errorf("cid %d: %w", sp.info.CID, err)
		}
	}
	return nil
}

// collectSeedSpecs 从 store 读取 E2E 相关漫画（2001/2002 比对与 3001/3002/3003 侧边栏），
// 页数显式声明：比对漫画各 5 页（前 2 页相同后 3 页不同），侧边栏漫画按场景 2/2/3 页。
func collectSeedSpecs(ctx context.Context, store *comicpkg.MemoryStorage) ([]seedSpec, error) {
	type comicSpec struct {
		cid   int
		pages int
	}
	specsDef := []comicSpec{
		{cid: 2001, pages: 5},
		{cid: 2002, pages: 5},
		{cid: 3001, pages: 2},
		{cid: 3002, pages: 2},
		{cid: 3003, pages: 3},
	}
	var specs []seedSpec
	for _, def := range specsDef {
		cid := def.cid
		c, err := store.Get(ctx, fmt.Sprintf("%d", cid))
		if err != nil {
			return nil, fmt.Errorf("get cid %d from store: %w", cid, err)
		}
		data, err := json.Marshal(c)
		if err != nil {
			return nil, fmt.Errorf("marshal cid %d: %w", cid, err)
		}
		var info api.ComicInfo
		if err := json.Unmarshal(data, &info); err != nil {
			return nil, fmt.Errorf("unmarshal cid %d: %w", cid, err)
		}

		var seedFn func(int) byte
		switch cid {
		case 2001:
			seedFn = func(i int) byte { return byte(i * 50) }
		case 2002:
			seedFn = func(i int) byte {
				if i <= 2 {
					return byte(i * 50) // 前 2 页与 2001 相同，保留 MD5 匹配语义
				}
				return byte(i*50 + 2002&0xff)
			}
		case 3001, 3002, 3003:
			seedFn = func(i int) byte { return byte(i * 50) }
		}
		specs = append(specs, seedSpec{info: &info, pages: def.pages, seedFn: seedFn})
	}
	return specs, nil
}

// generateSingleImageByInfo 按 ComicInfo 生成该漫画全部页面的 mock jpg 图片
func generateSingleImageByInfo(info *api.ComicInfo, galleryRoot string, numPages int, seedFn func(i int) byte) error {
	saveDir := filepath.Join(galleryRoot, info.StoragePrefix(), info.SaveDirName())
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return err
	}

	// 按页面列表（PicType）生成对应扩展名的图。种子数据页面均 T:"j" → jpg。
	numFiles := numPages
	if n := len(info.Images.Pages); n > 0 {
		numFiles = n
	}
	for i := 0; i < numFiles; i++ {
		var pageName string
		if i < len(info.Images.Pages) {
			pageName = info.Images.PageName(i + 1)
		} else {
			// 页面列表缺失时兜底为 1.jpg ... N.jpg（与生产 PageName 规则一致）
			pageName = fmt.Sprintf("%d.jpg", i+1)
		}
		filename := filepath.Join(saveDir, pageName)
		if err := generateMockImage(filename, seedFn(i+1)); err != nil {
			return err
		}
	}

	// 封面/缩略图文件：模板以 /galleries/{cid}/cover.jpg 或 thumb.jpg 引用，必需在磁盘上存在，
	// 否则详情页/首页的图片请求 404。若 Cover/Thumbnail 状态为真则生成同种子色图。
	if info.Images.Cover.Status {
		if err := generateMockImage(filepath.Join(saveDir, "cover."+info.Images.Cover.PicType()), seedFn(1)); err != nil {
			return err
		}
	}
	if info.Images.Thumbnail.Status {
		if err := generateMockImage(filepath.Join(saveDir, "thumb."+info.Images.Thumbnail.PicType()), seedFn(1)); err != nil {
			return err
		}
	}
	return nil
}

// generateMockImage 生成指定种子颜色的 1x1 JPG 图片（协议无关，供 .jpg 保存路径使用）
func generateMockImage(filename string, seed byte) error {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: seed, G: seed, B: seed, A: 255})
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
}
