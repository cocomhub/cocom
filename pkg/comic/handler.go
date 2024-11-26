package comic

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/suixibing/cocom/pkg/clog"
)

// VerifyRequest 验证请求参数
type VerifyRequest struct {
	Pattern string `json:"pattern"`  // 匹配规则
	AutoFix bool   `json:"auto_fix"` // 是否自动修复
	Workers int    `json:"workers"`  // 并发数
}

// VerifyResponse 验证响应结果
type VerifyResponse struct {
	Progress *VerifyProgress `json:"progress"`
	Error    string          `json:"error,omitempty"`
}

// HandleVerify 处理验证请求
func HandleVerify(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 解析请求参数
		var req VerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 验证参数
		if req.Pattern == "" {
			http.Error(w, "pattern is required", http.StatusBadRequest)
			return
		}
		if _, err := regexp.Compile(req.Pattern); err != nil {
			http.Error(w, ErrComicPattern.SetIErr(err).Error(), http.StatusBadRequest)
			return
		}
		if req.Workers <= 0 {
			req.Workers = 4
		}

		// 启动验证任务
		if err := service.StartVerify(ctx, req.Pattern, req.AutoFix); err != nil {
			clog.Errorf(ctx, "启动验证任务失败: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 返回进度
		resp := VerifyResponse{
			Progress: service.GetVerifyProgress(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// HandleVerifyProgress 处理进度查询请求
func HandleVerifyProgress(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := VerifyResponse{
			Progress: service.GetVerifyProgress(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// HandleVerifyCancel 处理取消验证请求
func HandleVerifyCancel(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service.CancelVerify()
		w.WriteHeader(http.StatusOK)
	}
}

// HandleInvalidComics 处理获取无效漫画请求
func HandleInvalidComics(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		comics, err := service.GetInvalidComics(ctx)
		if err != nil {
			clog.Errorf(ctx, "获取无效漫画失败: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comics)
	}
}
