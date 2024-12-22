package comic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/errwrap"
)

// Comic 漫画接口
type Comic interface {
	// 基本信息
	GetID() string
	GetTitle() string
	GetImages() []Image
	Object() interface{}

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
	Valid        bool      `json:"valid" bson:"valid"`                         // 图片是否全部正常
	InvalidCount int32     `json:"invalidCount" bson:"invalidCount,omitempty"` // 损坏图片数量
	FixedCount   int32     `json:"fixedCount" bson:"fixedCount,omitempty"`     // 已修复图片数量
	LastVerify   time.Time `json:"lastVerify" bson:"lastVerify"`               // 最后验证时间
}

// ComicImpl Comic接口的默认实现
type ComicImpl struct {
	ID         string  `json:"id" bson:"_id"`
	Title      string  `json:"title" bson:"title"`
	Images     []Image `json:"images" bson:"images"`
	VerifyInfo `json:"verify" bson:"verify"`
}

// NewComic 创建新的漫画实例
func NewComic(id, title string, images []Image) Comic {
	return &ComicImpl{
		ID:     id,
		Title:  title,
		Images: images,
	}
}

func NewComicImplByObject(obj interface{}) (*ComicImpl, error) {
	switch v := obj.(type) {
	case ComicImpl:
		return &v, nil
	case *ComicImpl:
		return v, nil
	case map[string]interface{}:
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

// Object 实现Comic接口
func (c *ComicImpl) Object() interface{} {
	return c
}

// MarshalJSON 实现Comic接口
func (c *ComicImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(c)
}

// UnmarshalJSON 实现Comic接口
func (c *ComicImpl) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

// IsValid 实现Comic接口
func (info *VerifyInfo) IsValid() bool {
	return info.Valid
}

// GetInvalidCount 实现Comic接口
func (info *VerifyInfo) GetInvalidCount() int32 {
	return info.InvalidCount
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

	if result.InvalidCount == 0 {
		info.Valid = true
		info.InvalidCount = 0
		info.FixedCount = result.FixedCount
	} else {
		info.Valid = false
		info.InvalidCount = result.InvalidCount
		info.FixedCount = result.FixedCount
	}
	info.LastVerify = result.Timestamp
}

// Downloader 图片下载器
type Downloader struct {
	bufPool sync.Pool
	client  *http.Client
}

// NewDownloader 创建下载器
func NewDownloader() *Downloader {
	return &Downloader{
		bufPool: sync.Pool{
			New: func() any {
				return make([]byte, 64*1024)
			},
		},
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,              // 最大空闲连接数
				MaxIdleConnsPerHost: 10,               // 每个主机的最大空闲连接数
				IdleConnTimeout:     90 * time.Second, // 空闲连接的超时时间
			},
			Timeout: 300 * time.Second, // 设置请求超时
		},
	}
}

// Download 下载图片
func (d *Downloader) Download(ctx context.Context, url, path string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errwrap.ErrImageDir.SetIErr(err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errwrap.ErrImageOpen.SetIErr(err)
	}

	// 执行请求
	resp, err := d.client.Do(req)
	if err != nil {
		return errwrap.ErrImageOpen.SetIErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errwrap.ErrImageOpen.SetIErrF("下载失败: HTTP %d", resp.StatusCode)
	}

	// 创建目标文件
	f, err := os.Create(path)
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

	clog.Debugf(ctx, "下载图片: %s -> %s (%d bytes)", url, path, written)
	return nil
}
