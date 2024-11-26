package comic

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/suixibing/cocom/pkg/clog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestHandleVerify_Success(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify success", func(mt *mtest.T) {
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
		)

		// 创建请求
		req := VerifyRequest{
			Pattern: comic.Title,
			AutoFix: true,
			Workers: 2,
		}
		data, err := json.Marshal(req)
		assert.NoError(t, err)

		// 发送请求
		r := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(data))
		r = r.WithContext(clog.NewTraceCtx("test"))
		w := httptest.NewRecorder()

		// 处理请求
		handler := HandleVerify(service)
		handler(w, r)

		// 验证响应
		assert.Equal(t, http.StatusOK, w.Code)

		var resp VerifyResponse
		err = json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.NotNil(t, resp.Progress)
		assert.Equal(t, len(comic.Images), resp.Progress.Total)
		assert.Empty(t, resp.Error)
	})
}

func TestHandleVerify_InvalidRequest(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify invalid request", func(mt *mtest.T) {
		service := NewService(mt.DB)

		tests := []struct {
			name    string
			request VerifyRequest
			want    int
		}{
			{
				name: "empty pattern",
				request: VerifyRequest{
					Pattern: "",
					AutoFix: true,
				},
				want: http.StatusBadRequest,
			},
			{
				name: "invalid pattern",
				request: VerifyRequest{
					Pattern: "[",
					AutoFix: true,
				},
				want: http.StatusBadRequest,
			},
			{
				name: "invalid workers",
				request: VerifyRequest{
					Pattern: ".*",
					Workers: -1,
				},
				want: http.StatusBadRequest,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := json.Marshal(tt.request)
				assert.NoError(t, err)

				r := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(data))
				r = r.WithContext(clog.NewTraceCtx("test"))
				w := httptest.NewRecorder()

				handler := HandleVerify(service)
				handler(w, r)

				assert.Equal(t, tt.want, w.Code)

				var resp VerifyResponse
				err = json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.Error)
			})
		}
	})
}

func TestHandleVerifyProgress(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify progress", func(mt *mtest.T) {
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

		// 启动验证任务
		ctx := clog.NewTraceCtx("test")
		err := service.StartVerify(ctx, comic.Title, true)
		assert.NoError(t, err)

		// 等待任务开始执行
		time.Sleep(100 * time.Millisecond)

		// 测试进度查询
		r := httptest.NewRequest(http.MethodGet, "/verify/progress", nil)
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()

		handler := HandleVerifyProgress(service)
		handler(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp VerifyResponse
		err = json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.NotNil(t, resp.Progress)
		assert.Equal(t, len(comic.Images), resp.Progress.Total)
		assert.Equal(t, float64(100), resp.Progress.Progress)
	})
}

func TestHandleVerifyCancel(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("verify cancel", func(mt *mtest.T) {
		service := NewService(mt.DB)

		// 测试取消验证
		r := httptest.NewRequest(http.MethodPost, "/verify/cancel", nil)
		r = r.WithContext(clog.NewTraceCtx("test"))
		w := httptest.NewRecorder()

		handler := HandleVerifyCancel(service)
		handler(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandleInvalidComics(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("invalid comics", func(mt *mtest.T) {
		// 准备测试环境
		tmpDir := t.TempDir()
		comic := createTestComic(t, mt.DB, tmpDir)

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

		service := NewService(mt.DB)

		// 测试获取无效漫画
		r := httptest.NewRequest(http.MethodGet, "/comics/invalid", nil)
		r = r.WithContext(clog.NewTraceCtx("test"))
		w := httptest.NewRecorder()

		handler := HandleInvalidComics(service)
		handler(w, r)

		assert.Equal(t, http.StatusOK, w.Code)

		var comics []*Comic
		err := json.NewDecoder(w.Body).Decode(&comics)
		assert.NoError(t, err)
		assert.Len(t, comics, 1)
		assert.Equal(t, comic.ID, comics[0].ID)
		assert.False(t, comics[0].Valid)
		assert.Equal(t, 3, comics[0].InvalidCount)
	})
}
