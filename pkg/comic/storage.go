package comic

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"sync"
	"time"
)

// Storage 定义了漫画存储的接口
type Storage interface {
	// 基本操作
	// Save(ctx context.Context, comic Comic) error
	Update(ctx context.Context, obj interface{}) error
	Get(ctx context.Context, id string) (Comic, error)
	// Delete(ctx context.Context, id string) error
	Find(ctx context.Context, filter *ComicFilter) ([]Comic, error)
	FindTotal(ctx context.Context, filter *ComicFilter) (int64, error)
	FindChannel(ctx context.Context, filter *ComicFilter) (chan Comic, error)

	// 验证相关
	// SaveVerifyResult(ctx context.Context, result *VerifyResult) error
	// GetVerifyResults(ctx context.Context, comicID string) (*VerifyResult, error)
}

const (
	DefaultOptionLimit = 10
	DefaultOptionSkip  = 0
)

// ComicFilter 漫画过滤器
type ComicFilter struct {
	ID           *string `json:"id,omitempty"`
	IDRangeLeft  *int64  `json:"idRangeLeft,omitempty"`
	IDRangeRight *int64  `json:"idRangeRight,omitempty"`
	TitlePattern *string `json:"titlePattern,omitempty"`
	PageMin      *int64  `json:"pageMin,omitempty"`
	PageMax      *int64  `json:"pageMax,omitempty"`
	Valid        *bool   `json:"valid,omitempty"`
	HasValid     *bool   `json:"hasValid,omitempty"`
	NotArchived  *bool   `json:"notArchived,omitempty"`
	Limit        int64   `json:"limit,omitempty"`
	Skip         int64   `json:"skip,omitempty"`
}

func NewComicFilter(opts ...func(*ComicFilter)) *ComicFilter {
	filter := &ComicFilter{
		Limit: DefaultOptionLimit,
		Skip:  DefaultOptionSkip,
	}
	for _, opt := range opts {
		opt(filter)
	}
	return filter.SetLimit(filter.Limit).SetSkip(filter.Skip)
}

func NewInvalidComicFilter(opts ...func(*ComicFilter)) *ComicFilter {
	return NewComicFilter(opts...).SetValid(false)
}

func (filter *ComicFilter) SetID(id string) *ComicFilter {
	if id == "" {
		filter.ID = nil
		return filter
	}
	filter.ID = &id
	return filter
}

func (filter *ComicFilter) SetIDRangeLeft(idRangeLeft int64) *ComicFilter {
	filter.IDRangeLeft = &idRangeLeft
	return filter
}

func (filter *ComicFilter) SetIDRangeRight(idRangeRight int64) *ComicFilter {
	filter.IDRangeRight = &idRangeRight
	return filter
}

func (filter *ComicFilter) SetTitlePattern(pattern string) *ComicFilter {
	if pattern == "" || pattern == "*" {
		filter.TitlePattern = nil
		return filter
	}
	filter.TitlePattern = &pattern
	return filter
}

func (filter *ComicFilter) SetPageMin(pageMin int64) *ComicFilter {
	filter.PageMin = &pageMin
	return filter
}

func (filter *ComicFilter) SetPageMax(pageMax int64) *ComicFilter {
	filter.PageMax = &pageMax
	return filter
}

func (filter *ComicFilter) SetValid(valid bool) *ComicFilter {
	filter.Valid = &valid
	return filter
}

func (filter *ComicFilter) SetHasValid(hasValid bool) *ComicFilter {
	filter.HasValid = &hasValid
	return filter
}

func (filter *ComicFilter) SetNotArchived(notArchived bool) *ComicFilter {
	filter.NotArchived = &notArchived
	return filter
}

func (filter *ComicFilter) SetLimit(limit int64) *ComicFilter {
	if limit <= 0 {
		limit = DefaultOptionLimit
	}
	filter.Limit = limit
	return filter
}

func (filter *ComicFilter) SetSkip(skip int64) *ComicFilter {
	if skip <= 0 {
		skip = DefaultOptionSkip
	}
	filter.Skip = skip
	return filter
}

// VerifyResult 验证结果
type VerifyResult struct {
	ID                      string    `json:"id"`                      // 结果ID
	ComicID                 string    `json:"comicId"`                 // 漫画ID
	Valid                   bool      `json:"valid"`                   // 是否有效
	InvalidCount            int32     `json:"invalidCount"`            // 无效数量
	InvalidSubsamplingCount int32     `json:"invalidSubsamplingCount"` // 无效子采样数量
	FixedCount              int32     `json:"fixedCount"`              // 修复数量
	Error                   error     `json:"error"`                   // 错误信息
	Timestamp               time.Time `json:"timestamp"`               // 时间戳

	fixImages []Image // 异常图片
}

// MemoryStorage 内存存储实现
type MemoryStorage struct {
	comics map[string]Comic
	mu     sync.RWMutex
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		comics: make(map[string]Comic),
	}
}

// Get 实现Storage接口
func (m *MemoryStorage) Get(ctx context.Context, id string) (Comic, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if comic, ok := m.comics[id]; ok {
		return comic, nil
	}
	return nil, ErrComicNotFound
}

// AddComic 添加漫画数据
func (m *MemoryStorage) Save(ctx context.Context, comic Comic) error {
	if comic == nil || comic.GetID() == "" {
		return fmt.Errorf("invalid comic id")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.comics[comic.GetID()] = comic
	return nil
}

// Find 实现Storage接口
func (m *MemoryStorage) Find(ctx context.Context, filter *ComicFilter) ([]Comic, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Comic
	for _, comic := range m.comics {
		if filter == nil {
			result = append(result, comic)
			continue
		}

		// 使用正则表达式匹配标题
		if filter.TitlePattern != nil {
			re, err := regexp.Compile(*filter.TitlePattern)
			if err != nil {
				return nil, fmt.Errorf("无效的匹配模式: %w", err)
			}
			if re.MatchString(comic.GetTitle()) {
				result = append(result, comic)
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].GetID() < result[j].GetID()
	})
	return result, nil
}

// SaveVerifyResult 实现Storage接口
func (m *MemoryStorage) SaveVerifyResult(ctx context.Context, result *VerifyResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if comic, ok := m.comics[result.ComicID]; ok {
		comic.SetVerifyResult(result)
		return nil
	}
	return ErrComicNotFound
}
