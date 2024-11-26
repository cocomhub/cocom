package comic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/suixibing/cocom/pkg/clog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestVerifyReport(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify report", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)

		ctx := clog.NewTraceCtx("test")
		verifier := NewComicVerifier(ctx, mt.DB)

		// 模拟查询响应
		mt.AddMockResponses(
			// 查询响应
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
			}),
			// 更新响应
			mtest.CreateSuccessResponse(),
		)

		// 启动验证任务
		err := verifier.Start(VerifyOptions{
			Pattern:    comic.Title,
			AutoFix:    true,
			Concurrent: 2,
		})
		assert.NoError(t, err)

		// 生成报告
		report := verifier.GenerateReport()
		assert.NotNil(t, report)
		assert.Equal(t, comic.Title, report.Pattern)
		assert.Equal(t, len(comic.Images), report.TotalFiles)
		assert.Equal(t, len(comic.Images), report.InvalidFiles) // 因为是测试数据，所有文件都是无效的
		assert.Equal(t, 0, report.FixedFiles)                   // 因为是测试数据，修复会失败
		assert.True(t, report.Duration > 0)
		assert.True(t, report.ProcessedMB > 0)
		assert.True(t, report.AverageSpeed > 0)

		// 保存报告
		reportPath := filepath.Join(tmpDir, "report.json")
		err = verifier.SaveReport(reportPath)
		assert.NoError(t, err)
		assert.FileExists(t, reportPath)

		// 验证保存的报告
		data, err := os.ReadFile(reportPath)
		assert.NoError(t, err)

		var savedReport VerifyReport
		err = json.Unmarshal(data, &savedReport)
		assert.NoError(t, err)
		assert.Equal(t, report.TotalFiles, savedReport.TotalFiles)
		assert.Equal(t, report.InvalidFiles, savedReport.InvalidFiles)
		assert.Equal(t, report.FixedFiles, savedReport.FixedFiles)
	})
}

func TestVerifyReport_WithRetry(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify report with retry", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)

		ctx := clog.NewTraceCtx("test")
		verifier := NewComicVerifier(ctx, mt.DB)

		// 添加一些重试记录
		retryRecords := []RetryRecord{
			{
				Time:    time.Now(),
				File:    "1.jpg",
				Error:   "download failed",
				Attempt: 1,
				Success: false,
			},
			{
				Time:    time.Now(),
				File:    "1.jpg",
				Error:   "",
				Attempt: 2,
				Success: true,
			},
		}

		// 模拟查询响应
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
				{Key: "retry_history", Value: retryRecords},
			}),
			mtest.CreateSuccessResponse(),
		)

		// 启动验证任务
		err := verifier.Start(VerifyOptions{
			Pattern:    comic.Title,
			AutoFix:    true,
			Concurrent: 2,
		})
		assert.NoError(t, err)

		// 生成报告
		report := verifier.GenerateReport()
		assert.NotNil(t, report)
		assert.Len(t, report.RetryHistory, len(retryRecords))
		assert.Equal(t, retryRecords[0].File, report.RetryHistory[0].File)
		assert.Equal(t, retryRecords[0].Attempt, report.RetryHistory[0].Attempt)
		assert.Equal(t, retryRecords[1].Success, report.RetryHistory[1].Success)
	})
}

func TestVerifyReport_Performance(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify report performance", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)

		ctx := clog.NewTraceCtx("test")
		verifier := NewComicVerifier(ctx, mt.DB)

		// 模拟查询响应
		mt.AddMockResponses(
			mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: comic.ID},
				{Key: "title", Value: comic.Title},
				{Key: "images", Value: comic.Images},
			}),
			mtest.CreateSuccessResponse(),
		)

		// 启动验证任务
		err := verifier.Start(VerifyOptions{
			Pattern:    comic.Title,
			AutoFix:    true,
			Concurrent: 2,
		})
		assert.NoError(t, err)

		// 生成报告
		report := verifier.GenerateReport()
		assert.NotNil(t, report)

		// 验证性能指标
		assert.True(t, report.Performance.CPUUsage > 0)
		assert.True(t, report.Performance.MemoryUsage > 0)
		assert.True(t, report.Performance.ErrorCount > 0)
		assert.True(t, report.Performance.RetryCount >= 0)

		// 验证资源使用
		assert.True(t, report.ResourceUsage.CPUTime > 0)
		assert.True(t, report.ResourceUsage.MaxMemory > 0)
		assert.True(t, report.ResourceUsage.DiskRead >= 0)
		assert.True(t, report.ResourceUsage.DiskWrite >= 0)
	})
}
