// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/gin-gonic/gin"
)

// Handler 处理comic相关的HTTP请求
type Handler struct {
	ctx     context.Context
	service Service
}

// NewHandler 创建处理器实例
func NewHandler(ctx context.Context, service Service) *Handler {
	return &Handler{
		ctx:     ctx,
		service: service,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	r.POST("/verify", h.StartVerifyTask)
	r.GET("/verify/:taskID", h.GetVerifyTask)
	r.GET("/verify/:taskID/progress", h.GetVerifyProgress)
	r.DELETE("/verify/:taskID", h.CancelVerifyTask)
	r.GET("/verify", h.GetVerifyTasks)
	r.POST("/verify/schedule", h.StartScheduleVerify)

	r.GET("/search", h.SearchComics)
	r.GET("/search/invalid", h.GetInvalidComics)
	r.GET("/:cid", h.GetComicInfo)
	r.GET("/:cid/cover", h.GetComicCoverPath)
	r.POST("/:cid/archive", h.ArchiveComic)
	r.POST("/:cid/restore", h.RestoreComic)
}

// StartVerifyTask 启动验证任务
func (h *Handler) StartVerifyTask(c *gin.Context) {
	ctx := c.Request.Context()

	var opts VerifyOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		httpwrap.GinRespondError(c, http.StatusBadRequest, -1, err.Error())
		return
	}

	taskID, err := h.service.StartVerifyTask(ctx, &opts)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	httpwrap.GinRespondOK(c, gin.H{
		"task_id": taskID,
		"message": "验证任务已启动",
	})
}

// GetVerifyTask 获取验证任务
func (h *Handler) GetVerifyTask(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("taskID")

	task, err := h.service.GetVerifyTask(ctx, taskID)
	if errors.Is(err, ErrTaskNotFound) {
		httpwrap.GinRespondError(c, http.StatusNotFound, -1, err.Error())
		return
	}
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	httpwrap.GinRespondOK(c, task)
}

// GetVerifyProgress 获取验证进度
func (h *Handler) GetVerifyProgress(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("taskID")

	progress, err := h.service.GetVerifyProgress(ctx, taskID)
	if errors.Is(err, ErrTaskNotFound) {
		httpwrap.GinRespondError(c, http.StatusNotFound, -1, err.Error())
		return
	}
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	httpwrap.GinRespondOK(c, progress)
}

// CancelVerifyTask 取消验证任务
func (h *Handler) CancelVerifyTask(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := c.Param("taskID")

	err := h.service.CancelVerifyTask(ctx, taskID)
	if errors.Is(err, ErrTaskNotFound) {
		httpwrap.GinRespondError(c, http.StatusNotFound, -1, err.Error())
		return
	}
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	httpwrap.GinRespondOK(c, gin.H{
		"message": "任务已取消",
	})
}

// GetVerifyTasks 列出所有验证任务
func (h *Handler) GetVerifyTasks(c *gin.Context) {
	ctx := c.Request.Context()

	tasks, err := h.service.GetVerifyTasks(ctx)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	result := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, map[string]any{
			"task_id":  task.GetProgress().TaskID,
			"progress": task.GetProgress(),
		})
	}

	httpwrap.GinRespondOK(c, gin.H{
		"tasks": result,
	})
}

// StartScheduleVerify 启动定时任务
func (h *Handler) StartScheduleVerify(c *gin.Context) {
	ctx := c.Request.Context()

	var cfg ScheduleConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		httpwrap.GinRespondError(c, http.StatusBadRequest, -1, err.Error())
		return
	}

	err := h.service.StartScheduleVerify(ctx, &cfg)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	httpwrap.GinRespondOK(c, gin.H{
		"message": "定时任务已启动",
	})
}

// SearchComics 搜索漫画
func (h *Handler) SearchComics(c *gin.Context) {
	ctx := c.Request.Context()

	filter, err := h.getComicFilter(c)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusBadRequest, -1, err.Error())
		return
	}

	comics, err := h.service.SearchComics(ctx, filter)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	httpwrap.GinRespondOK(c, gin.H{
		"comics": comics,
	})
}

// GetInvalidComics 获取无效漫画
func (h *Handler) GetInvalidComics(c *gin.Context) {
	ctx := c.Request.Context()

	filter, err := h.getComicFilter(c)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusBadRequest, -1, err.Error())
		return
	}

	comics, err := h.service.GetInvalidComics(ctx, filter)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	httpwrap.GinRespondOK(c, gin.H{
		"comics": comics,
	})
}

// GetComicInfo 获取漫画信息
func (h *Handler) GetComicInfo(c *gin.Context) {
	id := c.Param("cid")
	info, err := h.service.GetComicInfo(c.Request.Context(), id)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusNotFound, -1, err.Error())
		return
	}

	httpwrap.GinRespondOK(c, info)
}

// GetComicCoverPath 获取漫画封面路径
func (h *Handler) GetComicCoverPath(c *gin.Context) {
	id := c.Param("cid")
	info, err := h.service.GetComicInfo(c.Request.Context(), id)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusNotFound, -1, err.Error())
		return
	}

	images := info.GetImages()
	if len(images) == 0 {
		c.String(http.StatusForbidden, "")
		return
	}

	c.String(http.StatusOK, images[0].Path)
}

func (h *Handler) getComicFilter(c *gin.Context) (*ComicFilter, error) {
	filter := &ComicFilter{}

	if c.Query("cid") != "" {
		filter.SetID(c.Query("cid"))
	} else if c.Query("idRangeLeft") != "" || c.Query("idRangeRight") != "" {
		if c.Query("idRangeLeft") != "" {
			idRangeLeft, err := strconv.ParseInt(c.Query("idRangeLeft"), 10, 64)
			if err != nil {
				return nil, err
			}
			filter.SetIDRangeLeft(idRangeLeft)
		}
		if c.Query("idRangeRight") != "" {
			idRangeRight, err := strconv.ParseInt(c.Query("idRangeRight"), 10, 64)
			if err != nil {
				return nil, err
			}
			filter.SetIDRangeRight(idRangeRight)
		}
	}

	if c.Query("title") != "" {
		filter.SetTitlePattern(c.Query("title"))
	}

	if c.Query("pageMin") != "" {
		pageMin, err := strconv.ParseInt(c.Query("pageMin"), 10, 64)
		if err != nil {
			return nil, err
		}
		filter.SetPageMin(pageMin)
	}

	if c.Query("pageMax") != "" {
		pageMax, err := strconv.ParseInt(c.Query("pageMax"), 10, 64)
		if err != nil {
			return nil, err
		}
		filter.SetPageMax(pageMax)
	}

	if c.Query("valid") != "" {
		filter.SetValid(c.Query("valid") == "true")
	}

	if c.Query("hasValid") != "" {
		filter.SetHasValid(c.Query("hasValid") == "true")
	}

	if c.Query("limit") != "" {
		limit, err := strconv.ParseInt(c.Query("limit"), 10, 64)
		if err != nil {
			return nil, err
		}
		filter.SetLimit(limit)
	}

	if c.Query("skip") != "" {
		skip, err := strconv.ParseInt(c.Query("skip"), 10, 64)
		if err != nil {
			return nil, err
		}
		filter.SetSkip(skip)
	}
	return filter, nil
}

// // GetTaskInfo 获取任务信息
// func (h *Handler) GetTaskInfo(taskID string) (*TaskInfo, error) {
// 	task := h.service.GetTask(taskID)
// 	if task == nil {
// 		return nil, fmt.Errorf("任务不存在: %s", taskID)
// 	}

// 	return &TaskInfo{
// 		ID:       task.ID,
// 		Progress: task.Progress,
// 	}, nil
// }

// TaskInfo 任务信息
type TaskInfo struct {
	ID       string          `json:"id"`       // 任务ID
	Progress *VerifyProgress `json:"progress"` // 进度信息
}

// GetTask 获取任务
func (s *ServiceImpl) GetTask(taskID string) *VerifyTask {
	if value, ok := s.verifier.tasks.Load(taskID); ok {
		if task, ok := value.(*VerifyTask); ok {
			return task
		}
	}
	return nil
}

func (h *Handler) ArchiveComic(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("cid")
	force := strings.ToLower(strings.TrimSpace(c.Query("force"))) == "true"
	if !force {
		comic, gErr := h.service.GetComicInfo(ctx, id)
		if gErr != nil {
			httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, gErr.Error())
			return
		}
		invalids := make([]map[string]int, 0)
		for i, img := range comic.GetImages() {
			if info, err := imaging.VerifyImage(ctx, img.Path); err != nil || info == nil {
				invalids = append(invalids, map[string]int{"index": i + 1})
			}
		}
		if len(invalids) > 0 {
			httpwrap.GinRespond(c, http.StatusOK, -1001, "验证失败", gin.H{
				"invalid_images": invalids,
			})
			return
		}
	}
	if force {
		ctx = context.WithValue(ctx, "archive.force", true)
	}
	err := h.service.ArchiveComic(ctx, id)
	if err != nil {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	archived := h.tryGetArchived(ctx, id)
	if !archived {
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -2, "漫画未归档")
		return
	}
	httpwrap.GinRespondOK(c, "")
}

func (h *Handler) RestoreComic(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("cid")
	err := h.service.RestoreComic(ctx, id)
	if err != nil {
		var md5Err *ArchiveMD5MismatchError
		if errors.As(err, &md5Err) {
			httpwrap.GinRespond(c, http.StatusOK, -2001, "存档文件校验失败", gin.H{
				"expected_md5": md5Err.Expected,
				"actual_md5":   md5Err.Actual,
			})
			return
		}
		httpwrap.GinRespondError(c, http.StatusInternalServerError, -1, err.Error())
		return
	}

	archived := h.tryGetArchived(ctx, id)
	if !archived {
		// ignore
	}
	httpwrap.GinRespondOK(c, "")
}

func (h *Handler) tryGetArchived(ctx context.Context, id string) bool {
	info, err := h.service.GetComicInfo(ctx, id)
	if err != nil || info == nil {
		return false
	}
	data, err := info.MarshalJSON()
	if err != nil {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	archive, ok := m["archive"]
	if !ok || archive == nil {
		return false
	}
	if am, ok := archive.(map[string]any); ok {
		if path, ok := am["path"].(string); ok && path != "" {
			return true
		}
	}
	return false
}
