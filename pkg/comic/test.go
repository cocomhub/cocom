package comic

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/suixibing/cocom/pkg/clog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *mongo.Database {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	var db *mongo.Database
	mt.Run("setup", func(mt *mtest.T) {
		db = mt.DB
		// 模拟数据库操作成功响应
		mt.AddMockResponses(
			mtest.CreateSuccessResponse(),                              // 用于数据库初始化
			mtest.CreateCursorResponse(0, "foo.bar", mtest.FirstBatch), // 用于空查询
		)
	})
	return db
}

// createTestComic 创建测试漫画
func createTestComic(t *testing.T, db *mongo.Database, tmpDir string) *Comic {
	// 创建测试图片
	imgPaths := []string{
		filepath.Join(tmpDir, "1.jpg"),
		filepath.Join(tmpDir, "2.jpg"),
		filepath.Join(tmpDir, "3.jpg"),
	}

	for _, path := range imgPaths {
		err := os.WriteFile(path, []byte("test image data"), 0o644)
		assert.NoError(t, err)
	}

	// 创建测试漫画
	comic := &Comic{
		ID:    primitive.NewObjectID(),
		Title: "Test Comic",
		Images: []*Image{
			{URL: "http://example.com/1.jpg", Path: imgPaths[0]},
			{URL: "http://example.com/2.jpg", Path: imgPaths[1]},
			{URL: "http://example.com/3.jpg", Path: imgPaths[2]},
		},
		Valid:        false,
		InvalidCount: 3,
		LastVerify:   time.Now(),
	}

	// 模拟插入操作
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("insert comic", func(mt *mtest.T) {
		// 模拟插入响应
		mt.AddMockResponses(
			// 插入成功响应
			mtest.CreateSuccessResponse(),
			// 查询响应
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
				{Key: "valid", Value: comic.Valid},
				{Key: "invalid_count", Value: comic.InvalidCount},
				{Key: "last_verify", Value: comic.LastVerify},
			}),
			// 更新响应
			mtest.CreateSuccessResponse(),
		)

		_, err := db.Collection("comics").InsertOne(context.Background(), comic)
		assert.NoError(t, err)
	})

	return comic
}

// createTestVerifier 创建测试验证器
func createTestVerifier(t *testing.T, db *mongo.Database) *ComicVerifier {
	ctx := clog.NewTraceCtx("test")
	verifier := NewComicVerifier(ctx, db)

	// 添加测试配置
	verifier.progress = &VerifyProgress{
		Total:     3,
		Checked:   1,
		Invalid:   1,
		Progress:  33.33,
		StartTime: time.Now(),
	}

	// 添加测试状态
	verifier.lastChecked = primitive.NewObjectID().Hex()
	verifier.lastPattern = ".*"

	// 添加测试结果
	verifier.results = []*VerifyResult{
		{
			ComicID:      primitive.NewObjectID(),
			Title:        "Test Comic 1",
			InvalidCount: 1,
			FixedCount:   0,
			Timestamp:    time.Now(),
		},
	}

	return verifier
}

// createTestService 创建测试服务
func createTestService(t *testing.T, db *mongo.Database) *Service {
	service := NewService(db)
	service.verifier = createTestVerifier(t, db)
	return service
}

// createTestMonitor 创建测试监控器
func createTestMonitor(t *testing.T) *Monitor {
	ctx := clog.NewTraceCtx("test")
	metrics := NewMetricsCollector()
	monitor := NewMonitor(ctx, metrics, time.Second)

	// 添加测试数据
	monitor.checkpoints = []string{"point1", "point2"}
	monitor.errorFiles = []string{"error1.jpg", "error2.jpg"}
	monitor.retryQueue = []string{"retry1.jpg", "retry2.jpg"}
	monitor.skippedFiles = []string{"skip1.jpg", "skip2.jpg"}

	// 添加测试统计
	monitor.stats = &MonitorStats{
		StartTime:      time.Now(),
		Duration:       time.Second,
		NumGoroutine:   5,
		NumCPU:         runtime.NumCPU(),
		ProcessedMB:    100,
		AverageSpeed:   50,
		CurrentSpeed:   45,
		TotalFiles:     10,
		ProcessedFiles: 8,
		FailedFiles:    2,
		CPUUsage:       50,
		MemoryUsage:    30,
		DiskIO:         1024,
		NetworkIO:      2048,
		ErrorCount:     2,
		RetryCount:     2,
		QueueLength:    2,
	}

	return monitor
}

// cleanupTestFiles 清理测试文件
func cleanupTestFiles(t *testing.T, paths ...string) {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			t.Logf("清理文件失败: %v", err)
		}
	}
}
