package comic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/imaging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestVerifyImage(t *testing.T) {
	// 准备测试目录
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "test.jpg")

	// 创建有效图片
	err := os.WriteFile(srcPath, []byte("test image data"), 0o644)
	assert.NoError(t, err)

	// 创建无效图片
	invalidPath := filepath.Join(tmpDir, "invalid.jpg")
	err = os.WriteFile(invalidPath, []byte("invalid data"), 0o644)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		wantErr  bool
		wantInfo bool
	}{
		{
			name:     "valid image",
			path:     srcPath,
			wantErr:  false,
			wantInfo: true,
		},
		{
			name:     "invalid image",
			path:     invalidPath,
			wantErr:  true,
			wantInfo: false,
		},
		{
			name:     "non-existent file",
			path:     "nonexistent.jpg",
			wantErr:  true,
			wantInfo: false,
		},
	}

	ctx := clog.NewTraceCtx("test")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := imaging.NewImageHandler(ctx, tt.path, "")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			info := handler.GetInfo()
			if tt.wantInfo {
				assert.NotNil(t, info)
				assert.False(t, info.Invalid)
			} else {
				assert.True(t, info.Invalid)
			}
		})
	}
}

func TestVerifyComic(t *testing.T) {
	// 准备测试环境
	db := setupTestDB(t)
	if db == nil {
		return
	}

	tmpDir := t.TempDir()
	comic := createTestComic(t, db, tmpDir)

	ctx := clog.NewTraceCtx("test")
	verifier := NewComicVerifier(ctx, db)

	// 测试验证功能
	t.Run("verify comic", func(t *testing.T) {
		result := verifier.verifyComic(comic, true)
		assert.NotNil(t, result)
		assert.Equal(t, comic.ID, result.ComicID)
		assert.Equal(t, comic.Title, result.Title)
		assert.Len(t, result.Images, len(comic.Images))
		assert.Equal(t, len(comic.Images), result.InvalidCount) // 因为是无效的测试图片数据
		assert.Equal(t, 0, result.FixedCount)                   // 因为 URL 是无效的
		assert.NotZero(t, result.Timestamp)
	})

	// 测试自动修复
	t.Run("auto fix", func(t *testing.T) {
		// 创建一个有效的测试服务器来模拟图片下载
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("valid image data"))
		}))
		defer ts.Close()

		// 更新图片 URL 为测试服务器地址
		for _, img := range comic.Images {
			img.URL = ts.URL + "/" + filepath.Base(img.Path)
		}

		result := verifier.verifyComic(comic, true)
		assert.NotNil(t, result)
		assert.Equal(t, comic.ID, result.ComicID)
		assert.Equal(t, comic.Title, result.Title)
		assert.Len(t, result.Images, len(comic.Images))
		assert.Equal(t, len(comic.Images), result.InvalidCount)
		assert.Equal(t, len(comic.Images), result.FixedCount)
	})
}

func TestVerifyProgress_WithMetrics(t *testing.T) {
	// 准备测试环境
	db := setupTestDB(t)
	if db == nil {
		return
	}

	tmpDir := t.TempDir()
	comic := createTestComic(t, db, tmpDir)
	ctx := clog.NewTraceCtx("test")
	verifier := NewComicVerifier(ctx, db)

	// 测试取消功能
	t.Run("cancel verify", func(t *testing.T) {
		go func() {
			time.Sleep(50 * time.Millisecond)
			verifier.Cancel()
		}()

		err := verifier.Start(VerifyOptions{
			Pattern:    ".*",
			AutoFix:    true,
			Concurrent: 1,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	// 测试进度和性能指标
	t.Run("progress and metrics", func(t *testing.T) {
		// 启动验证任务
		go func() {
			err := verifier.Start(VerifyOptions{
				Pattern:    ".*",
				AutoFix:    true,
				Concurrent: 2,
			})
			assert.NoError(t, err)
		}()

		// 等待任务开始执行
		time.Sleep(100 * time.Millisecond)

		// 检查进度
		progress := verifier.GetProgress()
		assert.NotNil(t, progress)
		assert.Equal(t, 3, progress.Total) // 每个漫画有 3 张图片
		assert.True(t, progress.Checked > 0)
		assert.True(t, progress.Invalid > 0)
		assert.True(t, progress.Progress > 0)
		assert.NotZero(t, progress.StartTime)

		// 检查性能指标
		metrics := verifier.GetMetrics()
		assert.NotNil(t, metrics)
		assert.True(t, metrics.TotalFiles > 0)
		assert.True(t, metrics.ProcessedMB > 0)
		assert.True(t, metrics.Duration > 0)
		assert.True(t, metrics.AverageSpeed > 0)

		// 验证数据库更新
		var updated Comic
		err := db.Collection("comics").FindOne(ctx, map[string]interface{}{
			"_id": comic.ID,
		}).Decode(&updated)
		assert.NoError(t, err)
		assert.False(t, updated.Valid)
		assert.Equal(t, 3, updated.InvalidCount)
		assert.NotZero(t, updated.LastVerify)
	})
}

func TestVerifyComic_Basic(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify comic basic", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)
		verifier := NewComicVerifier(clog.NewTraceCtx("test"), mt.DB)

		// 测试基本验证功能
		t.Run("basic verify", func(t *testing.T) {
			result := verifier.verifyComic(comic, false)
			assert.NotNil(t, result)
			assert.Equal(t, comic.ID, result.ComicID)
			assert.Equal(t, comic.Title, result.Title)
			assert.Len(t, result.Images, len(comic.Images))
			assert.Equal(t, len(comic.Images), result.InvalidCount) // 因为是测试数据，所有文件都是无效的
			assert.Equal(t, 0, result.FixedCount)                   // 没有启用自动修复
		})

		// 测试自动修复功能
		t.Run("auto fix", func(t *testing.T) {
			// 创建测试服务器模拟图片下载
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("test image data"))
			}))
			defer ts.Close()

			// 更新图片 URL
			for _, img := range comic.Images {
				img.URL = ts.URL + "/" + filepath.Base(img.Path)
			}

			result := verifier.verifyComic(comic, true)
			assert.NotNil(t, result)
			assert.Equal(t, comic.ID, result.ComicID)
			assert.Equal(t, len(comic.Images), result.InvalidCount)
			assert.Equal(t, len(comic.Images), result.FixedCount) // 所有文件都应该被修复
		})
	})
}

func TestVerifyComic_Progress(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify comic progress", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)

		// 添加 mock 响应
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
			}),
			mtest.CreateSuccessResponse(),
		)

		ctx := clog.NewTraceCtx("test")
		verifier := NewComicVerifier(ctx, mt.DB)

		// 启动验证任务
		err := verifier.Start(VerifyOptions{
			Pattern:    comic.Title,
			AutoFix:    true,
			Concurrent: 2,
		})
		assert.NoError(t, err)

		// 验证进度
		progress := verifier.GetProgress()
		assert.NotNil(t, progress)
		assert.Equal(t, len(comic.Images), progress.Total)
		assert.Equal(t, len(comic.Images), progress.Checked)
		assert.Equal(t, len(comic.Images), progress.Invalid)
		assert.Equal(t, float64(100), progress.Progress)
	})
}

func TestVerifyComic_Schedule(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify comic schedule", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)

		// 添加 mock 响应
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
			}),
			mtest.CreateSuccessResponse(),
		)

		ctx, cancel := context.WithTimeout(clog.NewTraceCtx("test"), 200*time.Millisecond)
		defer cancel()

		verifier := NewComicVerifier(ctx, mt.DB)

		// 启动定时检查
		err := verifier.StartSchedule(ScheduleConfig{
			Pattern:    comic.Title,
			Interval:   100 * time.Millisecond,
			AutoFix:    true,
			Concurrent: 2,
		})
		assert.Error(t, err) // 应该返回 context.DeadlineExceeded
	})
}

func TestVerifyComic_State(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify comic state", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)
		ctx := clog.NewTraceCtx("test")
		verifier := NewComicVerifier(ctx, mt.DB)

		// 保存状态
		verifier.lastChecked = comic.ID.Hex()
		verifier.progress = &VerifyProgress{
			Total:     len(comic.Images),
			Checked:   1,
			Invalid:   1,
			Progress:  33.33,
			StartTime: time.Now(),
		}

		err := verifier.SaveState()
		assert.NoError(t, err)
		assert.FileExists(t, "verify_state.json")

		// 加载状态
		verifier = NewComicVerifier(ctx, mt.DB)
		err = verifier.LoadState()
		assert.NoError(t, err)
		assert.Equal(t, comic.ID.Hex(), verifier.lastChecked)
		assert.Equal(t, len(comic.Images), verifier.progress.Total)
		assert.Equal(t, 1, verifier.progress.Checked)
		assert.Equal(t, 1, verifier.progress.Invalid)
		assert.InDelta(t, 33.33, verifier.progress.Progress, 0.01)

		// 清理状态文件
		os.Remove("verify_state.json")
	})
}

func TestVerifyComic_PriorityQueue(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify comic priority queue", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comics := []*Comic{
			createTestComic(t, mt.DB, filepath.Join(tmpDir, "comic1")),
			createTestComic(t, mt.DB, filepath.Join(tmpDir, "comic2")),
			createTestComic(t, mt.DB, filepath.Join(tmpDir, "comic3")),
		}

		// 创建优先级队列
		pq := &PriorityQueue{}

		// 添加优先级规则
		pq.AddRule(PriorityRule{
			Name:   "last_verify",
			Weight: 1,
			Evaluate: func(c *Comic) float64 {
				return float64(c.LastVerify.Unix())
			},
		})

		// 设置不同的最后验证时间
		for i, comic := range comics {
			comic.LastVerify = time.Now().Add(time.Duration(i) * time.Hour)
			pq.Push(comic)
		}

		// 验证优先级顺序
		var result []*Comic
		for i := 0; i < len(comics); i++ {
			comic := pq.Pop()
			assert.NotNil(t, comic)
			result = append(result, comic)
		}

		// 应该按最后验证时间从早到晚排序
		for i := 1; i < len(result); i++ {
			assert.True(t, result[i-1].LastVerify.Before(result[i].LastVerify))
		}
	})
}
