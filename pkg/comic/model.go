package comic

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/errwrap"
	"github.com/suixibing/cocom/pkg/imaging"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Comic 漫画信息
type Comic struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	Title        string             `bson:"title" json:"title"`
	Images       []*Image           `bson:"images" json:"images"`
	Valid        bool               `bson:"valid" json:"valid"`                 // 图片是否全部正常
	InvalidCount int                `bson:"invalid_count" json:"invalid_count"` // 损坏图片数量
	FixedCount   int                `bson:"fixed_count" json:"fixed_count"`     // 已修复图片数量
	LastVerify   time.Time          `bson:"last_verify" json:"last_verify"`     // 最后验证时间
}

// Image 图片信息
type Image struct {
	URL  string `bson:"url" json:"url"`   // 图片 URL
	Path string `bson:"path" json:"path"` // 本地路径
}

// VerifyResult 验证结果
type VerifyResult struct {
	ComicID      primitive.ObjectID `json:"comic_id"`
	Title        string             `json:"title"`
	Images       []*ImageResult     `json:"images"`
	InvalidCount int                `json:"invalid_count"`
	FixedCount   int                `json:"fixed_count"`
	Timestamp    time.Time          `json:"timestamp"`
}

// ImageResult 图片验证结果
type ImageResult struct {
	Path    string             `json:"path"`
	URL     string             `json:"url"`
	Invalid bool               `json:"invalid"`
	Error   string             `json:"error,omitempty"`
	Info    *imaging.ImageInfo `json:"info,omitempty"`
}

// Downloader 图片下载器
type Downloader struct {
	ctx    context.Context
	client *http.Client
}

// NewDownloader 创建下载器
func NewDownloader(ctx context.Context) *Downloader {
	return &Downloader{
		ctx:    ctx,
		client: &http.Client{},
	}
}

// Download 下载图片
func (d *Downloader) Download(url, path string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errwrap.ErrImageDir.SetIErr(err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(d.ctx, http.MethodGet, url, nil)
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

	// 复制数据
	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return errwrap.ErrImageSave.SetIErr(err)
	}

	clog.Debugf(d.ctx, "下载图片: %s -> %s (%d bytes)", url, path, written)
	return nil
}
