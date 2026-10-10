// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/pkg/util"
)

// 页面文件与元数据的对应关系（本包 doSavePages 的核心不变量）：
//
//	第 i 页（1-based）的磁盘文件 = <SaveDir>/<i>.<ext>，ext 由 PicInfo.T 决定。
//
// 因此任何「增删改序」都必须同时改动**元数据数组**与**磁盘文件名**，否则页码与文件
// 错位：画廊按 PageName(no) 取盘（cmd/server/view/picture.go），错位即显示错误图片。
// 本文件提供按这个不变量落盘的实现。

// pageEntry 描述一个目标页：它的元信息 + 磁盘来源文件。
type pageEntry struct {
	// id 是加载时的原始页号（1-based）；0 表示本次新插入、没有原始页号的页。
	// 所有请求里的位置字段（page/to_page/after_page/source_page）都按这个原始页号解释。
	id int
	// pic 保留完整的页元信息（T/Status/W/H 等），重建数组时原样带回，避免丢字段。
	pic api.PicInfo
	// srcPath 是来源文件绝对路径；空串表示磁盘上没有对应文件（历史数据缺文件），
	// 该页会被保留在元数据里但不会落盘。
	srcPath string
	// copyOnly 为 true 时落盘只**复制**来源文件、保留原文件（跨漫画插入时必须如此，
	// 否则会把源漫画的页文件搬走）；为 false 时按移动处理（本漫画目录内的调整）。
	copyOnly bool
}

// pageUploadPrefix 是上传端点生成的暂存文件名前缀。
// doSavePages 只接受带该前缀的 file 字段，避免客户端引用漫画目录里的任意文件。
const pageUploadPrefix = ".page-upload-"

// pageImageExts 是被认可的页面图片后缀。
var pageImageExts = []string{"jpg", "jpeg", "png", "gif", "webp"}

func isPageImageExt(ext string) bool {
	e := strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, want := range pageImageExts {
		if e == want {
			return true
		}
	}
	return false
}

// pageExtForType 返回 PicInfo.T 对应的文件后缀。
func pageExtForType(t string) string {
	return api.PicInfo{T: t}.PicType()
}

// buildPageEntries 把当前元数据解析成带磁盘来源的条目列表。
//
// 元数据里的后缀可能与磁盘不一致（历史数据、手工改动），因此先按元数据推断文件名，
// 不存在时再枚举同序号 + 其它合法后缀。两者都不存在时 srcPath 置空（不报错：
// 保留该页元数据，与修复前的行为一致，不会让整个保存失败）。
func buildPageEntries(saveDir string, pages []api.PicInfo) []pageEntry {
	entries := make([]pageEntry, 0, len(pages))
	for i, p := range pages {
		no := i + 1
		entries = append(entries, pageEntry{
			id:      no,
			pic:     p,
			srcPath: resolvePageSource(saveDir, no, p.PicType()),
		})
	}
	return entries
}

// resolvePageSource 返回第 no 页在磁盘上的实际文件路径；找不到时返回空串。
func resolvePageSource(saveDir string, no int, wantExt string) string {
	primary := filepath.Join(saveDir, fmt.Sprintf("%d.%s", no, wantExt))
	if fileExists(primary) {
		return primary
	}
	for _, ext := range pageImageExts {
		if ext == wantExt {
			continue
		}
		candidate := filepath.Join(saveDir, fmt.Sprintf("%d.%s", no, ext))
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// stagedUploadPath 校验并返回一个上传暂存文件的绝对路径。
// 只接受「本漫画目录下、且带 pageUploadPrefix 前缀、后缀合法、确实存在」的文件名。
func stagedUploadPath(saveDir, name string) (string, error) {
	base := filepath.Base(name)
	if base != name || base == "." || base == ".." {
		return "", fmt.Errorf("非法的上传文件名 %q", name)
	}
	if !strings.HasPrefix(base, pageUploadPrefix) {
		return "", fmt.Errorf("文件 %q 不是通过上传接口创建的暂存文件", name)
	}
	if !isPageImageExt(filepath.Ext(base)) {
		return "", fmt.Errorf("文件 %q 的后缀不是受支持的图片格式", name)
	}
	full := filepath.Join(saveDir, base)
	if !fileExists(full) {
		return "", fmt.Errorf("暂存文件 %q 不存在", name)
	}
	return full, nil
}

// applyDeletes 删除条目列表中 id 属于 deleted 的项（deleted 为去重后的原始页号集合）。
func applyDeletes(entries []pageEntry, deleted map[int]struct{}) []pageEntry {
	if len(deleted) == 0 {
		return entries
	}
	kept := make([]pageEntry, 0, len(entries))
	for _, e := range entries {
		if _, drop := deleted[e.id]; drop {
			continue
		}
		kept = append(kept, e)
	}
	return kept
}

// applySwap 交换两个原始页号所在的位置（前端 reorder 是「交换」语义）。
// 任一 id 不存在时返回错误，避免静默无效。
func applySwap(entries []pageEntry, from, to int) error {
	i, j := -1, -1
	for idx := range entries {
		switch entries[idx].id {
		case from:
			i = idx
		case to:
			j = idx
		}
	}
	if i < 0 || j < 0 {
		return fmt.Errorf("交换失败：第 %d 页或第 %d 页不存在", from, to)
	}
	entries[i], entries[j] = entries[j], entries[i]
	return nil
}

// applyReplace 用暂存上传文件替换原始页号为 page 的那一页。
func applyReplace(saveDir string, entries []pageEntry, page int, uploadName string) error {
	if uploadName == "" {
		return fmt.Errorf("替换第 %d 页缺少已上传的文件", page)
	}
	full, err := stagedUploadPath(saveDir, uploadName)
	if err != nil {
		return err
	}
	for idx := range entries {
		if entries[idx].id != page {
			continue
		}
		entries[idx].srcPath = full
		entries[idx].pic.T = picTypeFromFileName(uploadName)
		entries[idx].pic.Status = true
		return nil
	}
	return fmt.Errorf("替换失败：第 %d 页不存在", page)
}

// insertAt 在 anchors 中按原始页号定位插入点，返回新列表。
// after 为 0 或负数 → 插到最前；after 找不到对应页（例如同批被删）→ 追加到末尾。
func insertAt(entries []pageEntry, after int, news []pageEntry) []pageEntry {
	if len(news) == 0 {
		return entries
	}
	pos := len(entries)
	if after <= 0 {
		pos = 0
	} else {
		for idx := range entries {
			if entries[idx].id == after {
				pos = idx + 1
				break
			}
		}
	}
	out := make([]pageEntry, 0, len(entries)+len(news))
	out = append(out, entries[:pos]...)
	out = append(out, news...)
	out = append(out, entries[pos:]...)
	return out
}

// materializePages 按条目顺序把来源文件落成 <saveDir>/<i>.<ext>，并清理不再被引用的
// 页面图片文件。所有搬运先进入同卷暂存目录，避免改名过程中的相互覆盖
// （例如删第 2 页后 3.jpg 要变成 2.jpg，而 2.jpg 此刻仍存在）。
//
// 失败时尽力把暂存文件搬回原位，并保留暂存目录（日志给出路径）以免静默丢文件。
func materializePages(saveDir string, entries []pageEntry) error {
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return fmt.Errorf("创建保存目录失败: %w", err)
	}
	stage, err := os.MkdirTemp(saveDir, ".pages-stage")
	if err != nil {
		return fmt.Errorf("创建暂存目录失败: %w", err)
	}

	// 目标文件名集合（用于清理孤儿）；srcPath 为空的页不产生文件。
	targets := make(map[string]struct{}, len(entries))
	type stagedFile struct {
		name     string
		origPath string
		copied   bool
	}
	moved := make([]stagedFile, 0, len(entries))

	rollback := func() {
		for _, m := range moved {
			if m.copied {
				// 来源仍在原位，只需丢弃暂存副本。
				_ = os.Remove(filepath.Join(stage, m.name))
				continue
			}
			if m.origPath == "" {
				continue
			}
			if rErr := util.Move(filepath.Join(stage, m.name), m.origPath); rErr != nil {
				// 搬不回去也不能再动，留在暂存目录里由人处理。
				continue
			}
		}
	}

	for i, e := range entries {
		name := fmt.Sprintf("%d.%s", i+1, pageExtForType(e.pic.T))
		targets[name] = struct{}{}
		if e.srcPath == "" {
			continue
		}
		stagedPath := filepath.Join(stage, name)
		if e.copyOnly {
			if err := util.CopyFile(e.srcPath, stagedPath); err != nil {
				rollback()
				return fmt.Errorf("暂存第 %d 页文件（复制）失败: %w", i+1, err)
			}
			moved = append(moved, stagedFile{name: name, origPath: e.srcPath, copied: true})
			continue
		}
		if err := util.Move(e.srcPath, stagedPath); err != nil {
			rollback()
			return fmt.Errorf("暂存第 %d 页文件失败: %w", i+1, err)
		}
		moved = append(moved, stagedFile{name: name, origPath: e.srcPath})
	}

	if err := removeOrphanPageFiles(saveDir, targets); err != nil {
		rollback()
		return err
	}

	for _, m := range moved {
		if err := util.Move(filepath.Join(stage, m.name), filepath.Join(saveDir, m.name)); err != nil {
			rollback()
			return fmt.Errorf("落盘 %s 失败: %w", m.name, err)
		}
	}

	if err := os.RemoveAll(stage); err != nil {
		return fmt.Errorf("清理暂存目录失败: %w", err)
	}
	return nil
}

// removeOrphanPageFiles 删除 saveDir 下所有不在 targets 中的页面图片文件。
// 只处理目录顶层的普通文件，其它类型（子目录、非图片）不动。
func removeOrphanPageFiles(saveDir string, targets map[string]struct{}) error {
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取保存目录失败: %w", err)
	}
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !isPageImageExt(filepath.Ext(name)) {
			continue
		}
		if _, keep := targets[name]; keep {
			continue
		}
		if err := os.Remove(filepath.Join(saveDir, name)); err != nil {
			return fmt.Errorf("删除不再引用的页面文件 %s 失败: %w", name, err)
		}
	}
	return nil
}

// buildInsertEntries 构造一个 insert 变更要插入的页面条目。
//
// 两种来源：
//   - 已上传的单页（file 字段，必须是上传端点产生的暂存文件）；
//   - 从 source_cid 的 source_pages 复制（源文件保留，故 copyOnly=true）。
func buildInsertEntries(ctx context.Context, saveDir string, p savePagesItem) ([]pageEntry, error) {
	if name := p.fileName(); name != "" {
		full, err := stagedUploadPath(saveDir, name)
		if err != nil {
			return nil, err
		}
		return []pageEntry{{
			pic:     api.PicInfo{T: picTypeFromFileName(name), Status: true},
			srcPath: full,
		}}, nil
	}

	if p.SourceCID <= 0 || len(p.SourcePages) == 0 {
		return nil, fmt.Errorf("插入需要 source_cid + source_pages，或先通过上传接口得到 file")
	}

	var src api.ComicInfo
	if err := comic.GetComicInfo(ctx, p.SourceCID, &src); err != nil {
		return nil, fmt.Errorf("读取源漫画 %d 失败: %w", p.SourceCID, err)
	}
	if src.CID != 0 && src.CID != p.SourceCID {
		return nil, fmt.Errorf("源漫画 %d 返回了不匹配的记录（cid=%d）", p.SourceCID, src.CID)
	}
	srcDir := src.SaveDir()

	out := make([]pageEntry, 0, len(p.SourcePages))
	for _, sp := range p.SourcePages {
		if sp < 1 || sp > len(src.Images.Pages) {
			return nil, fmt.Errorf("源漫画第 %d 页不存在（共 %d 页）", sp, len(src.Images.Pages))
		}
		srcPic := src.Images.Pages[sp-1]
		srcPath := resolvePageSource(srcDir, sp, srcPic.PicType())
		if srcPath == "" {
			return nil, fmt.Errorf("源漫画第 %d 页的文件缺失（目录 %s）", sp, srcDir)
		}
		out = append(out, pageEntry{
			pic:      api.PicInfo{T: srcPic.T, Status: true},
			srcPath:  srcPath,
			copyOnly: true,
		})
	}
	return out, nil
}
