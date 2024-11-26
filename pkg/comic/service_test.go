package comic

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/suixibing/cocom/pkg/clog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestService_StartVerify(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify service", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)
		service := NewService(mt.DB)

		// 添加 mock 响应
		mt.AddMockResponses(
			// 查询响应
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
			}),
			// 更新响应
			mtest.CreateSuccessResponse(),
			// 查询进度响应
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "total", Value: len(comic.Images)},
				{Key: "checked", Value: len(comic.Images)},
			}),
		)

		// 测试验证功能
		ctx := clog.NewTraceCtx("test")
		err := service.StartVerify(ctx, comic.Title, true)
		assert.NoError(t, err)

		// 验证进度
		progress := service.GetVerifyProgress()
		assert.Equal(t, len(comic.Images), progress.Total)
		assert.Equal(t, len(comic.Images), progress.Checked)

		// 验证报告
		verifier := service.GetVerifier()
		assert.NotNil(t, verifier)

		report := verifier.GenerateReport()
		assert.NotNil(t, report)
		assert.Equal(t, comic.Title, report.Pattern)
		assert.Equal(t, len(comic.Images), report.TotalFiles)
		assert.Equal(t, len(comic.Images), report.InvalidFiles)
		assert.Equal(t, 0, report.FixedFiles)

		// 保存报告
		reportPath := filepath.Join(tmpDir, "report.json")
		err = verifier.SaveReport(reportPath)
		assert.NoError(t, err)
		assert.FileExists(t, reportPath)

		// 清理报告文件
		os.Remove(reportPath)
	})
}

func TestService_ScheduleVerify(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("schedule verify", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)
		service := NewService(mt.DB)

		// 添加 mock 响应
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
			}),
			mtest.CreateSuccessResponse(),
		)

		// 测试定时验证
		ctx, cancel := context.WithTimeout(clog.NewTraceCtx("test"), 200*time.Millisecond)
		defer cancel()

		cfg := ScheduleConfig{
			Pattern:       comic.Title,
			Interval:      100 * time.Millisecond,
			AutoFix:       true,
			Concurrent:    2,
			RetryInterval: time.Second,
			MaxRetries:    3,
			TimeWindow: []TimeRange{
				{Start: "00:00", End: "06:00"},
			},
			Priority: 1,
			Timeout:  time.Minute,
		}

		err := service.StartScheduleVerify(ctx, cfg)
		assert.Error(t, err) // 应该返回 context.DeadlineExceeded
	})
}

func TestService_GetInvalidComics(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("get invalid comics", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)
		service := NewService(mt.DB)

		// 添加 mock 响应
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
				{Key: "valid", Value: false},
				{Key: "invalid_count", Value: 3},
			}),
		)

		// 测试获取无效漫画
		ctx := clog.NewTraceCtx("test")
		comics, err := service.GetInvalidComics(ctx)
		assert.NoError(t, err)
		assert.Len(t, comics, 1)
		assert.Equal(t, comic.ID, comics[0].ID)
		assert.False(t, comics[0].Valid)
		assert.Equal(t, 3, comics[0].InvalidCount)
	})
}

func TestService_CancelVerify(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("cancel verify", func(mt *mtest.T) {
		// 准备测试环境
		service := NewService(mt.DB)

		// 启动验证任务
		ctx := clog.NewTraceCtx("test")
		go func() {
			err := service.StartVerify(ctx, ".*", true)
			assert.Error(t, err) // 应该返回 context.Canceled
		}()

		// 等待任务启动
		time.Sleep(50 * time.Millisecond)

		// 取消任务
		service.CancelVerify()

		// 验证进度
		progress := service.GetVerifyProgress()
		assert.NotNil(t, progress)
		assert.Equal(t, 0, progress.Total)
		assert.Equal(t, 0, progress.Checked)
	})
}

func TestService_ErrorHandling(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("error handling", func(mt *mtest.T) {
		service := NewService(mt.DB)

		tests := []struct {
			name     string
			pattern  string
			autoFix  bool
			mockResp []bson.D
			wantErr  bool
		}{
			{
				name:    "empty pattern",
				pattern: "",
				autoFix: true,
				wantErr: true,
			},
			{
				name:    "invalid pattern",
				pattern: "[",
				autoFix: true,
				wantErr: true,
			},
			{
				name:    "database error",
				pattern: ".*",
				autoFix: true,
				mockResp: []bson.D{
					{{Key: "ok", Value: 0}, {Key: "errmsg", Value: "database error"}},
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if len(tt.mockResp) > 0 {
					mt.AddMockResponses(tt.mockResp...)
				}

				ctx := clog.NewTraceCtx("test")
				err := service.StartVerify(ctx, tt.pattern, tt.autoFix)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}
