package comic

import (
	"context"
	"sync"
	"time"

	"github.com/suixibing/cocom/pkg/clog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Service 漫画服务
type Service struct {
	db       *mongo.Database
	verifier *ComicVerifier
	mu       sync.RWMutex
}

// NewService 创建漫画服务
func NewService(db *mongo.Database) *Service {
	return &Service{
		db: db,
	}
}

// StartVerify 启动验证任务
func (s *Service) StartVerify(ctx context.Context, pattern string, autoFix bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已有任务在运行，先取消
	if s.verifier != nil {
		s.verifier.Cancel()
	}

	// 创建新的验证器
	s.verifier = NewComicVerifier(ctx, s.db)

	// 启动验证任务
	return s.verifier.Start(VerifyOptions{
		Pattern:    pattern,
		AutoFix:    autoFix,
		Concurrent: 4,
	})
}

// GetVerifyProgress 获取验证进度
func (s *Service) GetVerifyProgress() *VerifyProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.verifier == nil {
		return &VerifyProgress{}
	}
	return s.verifier.GetProgress()
}

// CancelVerify 取消验证任务
func (s *Service) CancelVerify() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.verifier != nil {
		s.verifier.Cancel()
		s.verifier = nil
	}
}

// GetInvalidComics 获取所有无效漫画
func (s *Service) GetInvalidComics(ctx context.Context) ([]*Comic, error) {
	filter := bson.M{
		"valid": false,
	}
	cursor, err := s.db.Collection("comics").Find(ctx, filter)
	if err != nil {
		return nil, ErrComicDB.SetIErr(err)
	}
	defer cursor.Close(ctx)

	var comics []*Comic
	if err := cursor.All(ctx, &comics); err != nil {
		return nil, ErrComicDB.SetIErr(err)
	}
	return comics, nil
}

// ScheduleVerify 定时验证任务
func (s *Service) ScheduleVerify(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.StartVerify(ctx, ".*", true); err != nil {
				clog.Errorf(ctx, "定时验证任务失败: %v", err)
			}
		}
	}
}

// GetVerifier 获取验证器实例
func (s *Service) GetVerifier() *ComicVerifier {
	return s.verifier
}

// StartScheduleVerify 启动定时检查
func (s *Service) StartScheduleVerify(ctx context.Context, cfg ScheduleConfig) error {
	if s.verifier == nil {
		s.verifier = NewComicVerifier(ctx, s.db)
	}
	return s.verifier.StartSchedule(cfg)
}
