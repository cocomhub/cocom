// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cocomhub/cocom/pkg/errwrap"
)

// Comic 漫画接口
type Comic interface {
	// 基本信息
	GetID() string
	GetTitle() string
	GetImages() []Image
	GetTags() []Tag
	Object() any

	// 归档信息
	GetArchivePath() string

	// 状态相关
	IsValid() bool
	GetInvalidCount() int32
	GetFixedCount() int32
	GetLastVerify() time.Time
	SetVerifyResult(result *VerifyResult)

	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

// Image 图片信息
type Image struct {
	ID   string `json:"id" bson:"id"`
	Path string `json:"path" bson:"path"`
	URL  string `json:"url" bson:"url"`
}

type VerifyInfo struct {
	Valid                   bool      `json:"valid" bson:"valid"`                                               // 图片是否全部正常
	InvalidCount            int32     `json:"invalidCount" bson:"invalidCount,omitempty"`                       // 损坏图片数量
	InvalidSubsamplingCount int32     `json:"invalidSubsamplingCount" bson:"invalidSubsamplingCount,omitempty"` // 子采样图片数量
	FixedCount              int32     `json:"fixedCount" bson:"fixedCount,omitempty"`                           // 已修复图片数量
	LastVerify              time.Time `json:"lastVerify" bson:"lastVerify"`                                     // 最后验证时间
}

// Tag 标签信息
type Tag struct {
	Count int    `json:"count,omitempty" bson:"count"`
	ID    int    `json:"id,omitempty" bson:"id"`
	Name  string `json:"name,omitempty" bson:"name"`
	Type  string `json:"type,omitempty" bson:"type"`
	URL   string `json:"url,omitempty" bson:"url"`
}

// ComicImpl Comic接口的默认实现
type ComicImpl struct {
	ID         string  `json:"id" bson:"_id"`
	Title      string  `json:"title" bson:"title"`
	Images     []Image `json:"images" bson:"images"`
	Tags       []Tag   `json:"tags,omitempty" bson:"tags"`
	VerifyInfo `json:"verify" bson:"verify"`

	// archivePath 用于 MemoryStorage 追踪归档路径，不在 JSON 序列化中暴露
	archivePath string
}

// SetArchivePath 设置归档路径（仅供 MemoryStorage 内部使用）
func (c *ComicImpl) SetArchivePath(path string) {
	c.archivePath = path
}

// NewComic 创建新的漫画实例
func NewComic(id, title string, images []Image) Comic {
	return &ComicImpl{
		ID:     id,
		Title:  title,
		Images: images,
	}
}

func NewComicImplByObject(obj any) (*ComicImpl, error) {
	switch v := obj.(type) {
	case ComicImpl:
		return &v, nil
	case *ComicImpl:
		return v, nil
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var comic ComicImpl
		err = json.Unmarshal(data, &comic)
		if err != nil {
			return nil, err
		}
		return &comic, nil
	default:
		return nil, fmt.Errorf("invalid object type: %T", obj)
	}
}

// GetID 实现Comic接口
func (c *ComicImpl) GetID() string {
	return c.ID
}

// GetTitle 实现Comic接口
func (c *ComicImpl) GetTitle() string {
	return c.Title
}

// GetImages 实现Comic接口
func (c *ComicImpl) GetImages() []Image {
	return c.Images
}

// GetTags 实现Comic接口
func (c *ComicImpl) GetTags() []Tag {
	return c.Tags
}

// GetArchivePath 实现Comic接口
func (c *ComicImpl) GetArchivePath() string {
	return c.archivePath
}

// Object 实现Comic接口
func (c *ComicImpl) Object() any {
	return c
}

// MarshalJSON 实现Comic接口
func (c *ComicImpl) MarshalJSON() ([]byte, error) {
	// 使用 type alias 避免递归
	type comicAlias ComicImpl
	return json.Marshal((*comicAlias)(c))
}

// UnmarshalJSON 实现Comic接口
func (c *ComicImpl) UnmarshalJSON(data []byte) error {
	// 使用 type alias 避免递归
	type comicAlias ComicImpl
	alias := (*comicAlias)(c)
	return json.Unmarshal(data, alias)
}

// IsValid 实现Comic接口
func (info *VerifyInfo) IsValid() bool {
	return info.Valid
}

// GetInvalidCount 实现Comic接口
func (info *VerifyInfo) GetInvalidCount() int32 {
	return info.InvalidCount
}

// GetInvalidSubsamplingCount 实现Comic接口
func (info *VerifyInfo) GetInvalidSubsamplingCount() int32 {
	return info.InvalidSubsamplingCount
}

// GetFixedCount 实现Comic接口
func (info *VerifyInfo) GetFixedCount() int32 {
	return info.FixedCount
}

// GetLastVerify 实现Comic接口
func (info *VerifyInfo) GetLastVerify() time.Time {
	return info.LastVerify
}

// SetVerifyResult 实现Comic接口
func (info *VerifyInfo) SetVerifyResult(result *VerifyResult) {
	if result == nil {
		return
	}

	info.Valid = result.Valid
	info.InvalidCount = result.InvalidCount
	info.InvalidSubsamplingCount = result.InvalidSubsamplingCount
	info.FixedCount = result.FixedCount
	info.LastVerify = result.Timestamp
}

// downloader 图片下载器
type downloader struct {
	bufPool sync.Pool
	client  *http.Client
}

// NewDownloader 创建下载器
func NewDownloader() *downloader {
	return &downloader{
		bufPool: sync.Pool{
			New: func() any {
				return make([]byte, 64*1024)
			},
		},
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:          1000,             // 最大空闲连接数
				MaxIdleConnsPerHost:   100,              // 每个主机的最大空闲连接数
				IdleConnTimeout:       90 * time.Second, // 空闲连接的超时时间
				TLSHandshakeTimeout:   10 * time.Second, // TLS握手超时
				ResponseHeaderTimeout: 15 * time.Second, // 响应头超时
				ExpectContinueTimeout: 1 * time.Second,  // Expect Continue超时
			},
		},
	}
}

// Download 下载图片
func (d *downloader) Download(ctx context.Context, url, path string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errwrap.ErrImageDir.SetIErr(err)
	}

	err := d.doDownload(ctx, url, path)
	if err == nil {
		return nil
	}

	for proxy := range map[string]struct{}{
		"http://129.226.212.209:18081": {},
		"http://43.159.49.114:18081":   {},
	} {
		url2 := d.proxyURL(url, proxy)
		slog.InfoContext(ctx, "Using proxy", slog.String("url", url), slog.String("url2", url2))
		err := d.doDownload(ctx, url2, path)
		if err == nil {
			return nil
		}
	}
	return err
}

func (d *downloader) proxyURL(url, proxyURL string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = proxyURL + "/" + url
	return url
}

func (d *downloader) doDownload(ctx context.Context, url, path string) error {
	const maxRetries = 3 // 最大重试次数
	var retryErr error

	for retry := range maxRetries {
		// 每次重试创建新请求
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return errwrap.ErrImageOpen.SetIErr(err)
		}

		// 断点续传逻辑
		fileMode := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		var currentSize int64 = 0
		if stat, err := os.Stat(path); err == nil && stat.Size() > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", stat.Size()))
			fileMode = os.O_WRONLY | os.O_APPEND
			currentSize = stat.Size()
		}

		// 执行请求
		resp, err := d.client.Do(req)
		if err != nil {
			// 网络错误可重试
			if isNetErrorRetriable(err) && retry < maxRetries-1 {
				retryErr = err
				slog.WarnContext(ctx, "下载失败", slog.Int("retry", retry+1), slog.Int("maxRetries", maxRetries), slog.String("err", err.Error()))
				time.Sleep(time.Duration(retry+1) * time.Second) // 指数退避
				continue
			}
			return errwrap.ErrImageOpen.SetIErr(err)
		}

		// 处理非成功状态码
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			if isStatusCodeRetriable(resp.StatusCode) && retry < maxRetries-1 {
				slog.WarnContext(ctx, "服务器返回", slog.Int("statusCode", resp.StatusCode), slog.Int("retry", retry+1), slog.Int("maxRetries", maxRetries))
				time.Sleep(time.Duration(retry+1) * time.Second)
				continue
			}
			return errwrap.ErrImageOpen.SetIErrF("下载失败: HTTP %d", resp.StatusCode)
		}

		// 处理文件写入
		f, err := os.OpenFile(path, fileMode, 0o644)
		if err != nil {
			resp.Body.Close()
			return errwrap.ErrImageSave.SetIErr(err)
		}

		// 如果服务器不支持断点续传（返回200但要求续传）
		if currentSize > 0 && resp.StatusCode == http.StatusOK {
			if err := f.Truncate(0); err != nil {
				resp.Body.Close()
				f.Close()
				return errwrap.ErrImageSave.SetIErr(err)
			}
			if _, err := f.Seek(0, 0); err != nil {
				resp.Body.Close()
				f.Close()
				return errwrap.ErrImageSave.SetIErr(err)
			}
		}

		// 复制数据
		buf := d.bufPool.Get().([]byte)
		written, err := io.CopyBuffer(f, resp.Body, buf)
		d.bufPool.Put(buf)
		f.Close()
		resp.Body.Close()

		if err != nil {
			if isNetErrorRetriable(err) && retry < maxRetries-1 {
				retryErr = err
				slog.WarnContext(ctx, "写入失败", slog.Int("retry", retry+1), slog.Int("maxRetries", maxRetries), slog.String("err", err.Error()))
				time.Sleep(time.Duration(retry+1) * time.Second)
				continue
			}
			return errwrap.ErrImageSave.SetIErr(err)
		}

		// 设置文件时间
		if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
			if fileTime, err := http.ParseTime(lastModified); err == nil {
				os.Chtimes(path, fileTime, fileTime)
			}
		}

		slog.DebugContext(ctx, "下载完成", slog.String("url", url), slog.String("path", path), slog.Int64("written", written))
		return nil
	}

	return errwrap.ErrImageOpen.SetIErr(retryErr)
}

// 可重试的网络错误判断
func isNetErrorRetriable(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}

// 可重试的状态码判断
func isStatusCodeRetriable(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

func (d *downloader) DownloadV1(ctx context.Context, url, path string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errwrap.ErrImageDir.SetIErr(err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errwrap.ErrImageOpen.SetIErr(err)
	}

	// stat, err := os.Stat(path)
	// if err == nil {
	// 	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", stat.Size()))
	// 	slog.InfoContext(ctx, "下载图片(续传)", slog.String("url", url), slog.String("path", path), stat.Size())
	// }

	// 执行请求
	resp, err := d.client.Do(req)
	if err != nil {
		return errwrap.ErrImageOpen.SetIErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return errwrap.ErrImageOpen.SetIErrF("下载失败: HTTP %d", resp.StatusCode)
	}

	// 创建目标文件
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return errwrap.ErrImageSave.SetIErr(err)
	}
	defer f.Close()

	// 使用更大的缓冲区
	buf := d.bufPool.Get()
	defer d.bufPool.Put(buf)

	written, err := io.CopyBuffer(f, resp.Body, buf.([]byte))
	if err != nil {
		return errwrap.ErrImageSave.SetIErr(err)
	}

	// 从响应头中获取 Last-Modified 时间
	lastModified := resp.Header.Get("Last-Modified")
	if lastModified != "" {
		fileTime, err := http.ParseTime(lastModified)
		if err == nil {
			_ = os.Chtimes(path, fileTime, fileTime)
		}
	}

	slog.DebugContext(ctx, "下载成功", slog.String("url", url), slog.String("path", path), slog.Int64("written", written))
	return nil
}

// WgetDownloader 使用wget的下载器
type WgetDownloader struct {
	wgetPath   string // wget可执行文件路径
	timeout    time.Duration
	maxRetries int
	userAgent  string
}

// NewWgetDownloader 创建下载器
func NewWgetDownloader() *WgetDownloader {
	return &WgetDownloader{
		wgetPath:   findWgetPath(),    // 自动查找wget路径
		timeout:    300 * time.Second, // 总超时时间
		maxRetries: 5,                 // 最大重试次数
		userAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	}
}

// Download 使用wget下载文件
func (d *WgetDownloader) Download(ctx context.Context, url, path string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errwrap.ErrImageDir.SetIErr(err)
	}

	// 构造wget命令参数
	args := []string{
		"-nv", // 非冗余模式
		"-c",  // 断点续传
		"--tries", fmt.Sprintf("%d", d.maxRetries),
		"--timeout", fmt.Sprintf("%.0f", d.timeout.Seconds()),
		"--waitretry", "3", // 重试间隔
		"--user-agent", d.userAgent,
		"--header", "Accept-Encoding: gzip, deflate",
		"-O", path, // 输出路径
		url,
	}

	// 创建命令上下文
	cmdCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(cmdCtx, d.wgetPath, args...)

	// 捕获输出用于调试
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		// 解析wget错误代码
		exitCode := cmd.ProcessState.ExitCode()
		errMsg := strings.TrimSpace(stderr.String())

		// 分类处理常见错误
		switch {
		case exitCode == 3: // 文件I/O错误
			return errwrap.ErrImageSave.SetIErrF("wget I/O错误: %s", errMsg)
		case exitCode == 4: // 网络失败
			return errwrap.ErrImageOpen.SetIErrF("网络错误: %s", errMsg)
		case exitCode == 8: // 服务器错误
			return errwrap.ErrImageOpen.SetIErrF("服务器返回错误: %s", errMsg)
		default:
			return errwrap.ErrImageOpen.SetIErrF("wget失败(%d): %s", exitCode, errMsg)
		}
	}

	// 验证下载文件
	if stat, err := os.Stat(path); err != nil || stat.Size() == 0 {
		return errwrap.ErrImageSave.SetIErrF("下载文件验证失败")
	}

	slog.DebugContext(ctx, "下载成功", slog.String("url", url), slog.String("path", path))
	return nil
}

// 查找wget可执行文件路径
func findWgetPath() string {
	// 尝试常见路径
	paths := []string{
		"/usr/bin/wget",
		"/bin/wget",
		"/opt/homebrew/bin/wget",
		"C:\\cygwin\\bin\\wget.exe",
		"C:\\Program Files\\Git\\usr\\bin\\wget.exe",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// 最后尝试PATH环境变量
	if path, err := exec.LookPath("wget"); err == nil {
		return path
	}

	panic("wget未找到，请确保已安装wget")
}
