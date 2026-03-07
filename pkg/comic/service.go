// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package comic

import (
	"context"
	"fmt"

	"github.com/cocomhub/cocom/pkg/clog"
)

type Service interface {
	StartVerifyTask(ctx context.Context, opts *VerifyOptions) (string, error)
	GetVerifyTask(ctx context.Context, taskID string) (*VerifyTask, error)
	GetVerifyTasks(ctx context.Context) ([]*VerifyTask, error)
	GetVerifyProgress(ctx context.Context, taskID string) (*VerifyProgress, error)
	CancelVerifyTask(ctx context.Context, taskID string) error
	StartScheduleVerify(ctx context.Context, cfg *ScheduleConfig) error

	SearchComics(ctx context.Context, filter *ComicFilter) ([]Comic, error)
	GetInvalidComics(ctx context.Context, filter *ComicFilter) ([]Comic, error)
	GetComicInfo(ctx context.Context, id string) (Comic, error)
}

// ServiceImpl 漫画服务
type ServiceImpl struct {
	ctx      context.Context
	cancel   context.CancelFunc
	storage  Storage
	verifier *ComicVerifier
}

// NewService 创建漫画服务
func NewService(ctx context.Context, storage Storage) (Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	verifier, err := NewComicVerifier(ctx, storage)
	if err != nil {
		clog.Errorf(ctx, "Create comic verifier failed: %v", err)
		cancel()
		return nil, err
	}
	return &ServiceImpl{
		ctx:      ctx,
		cancel:   cancel,
		storage:  storage,
		verifier: verifier,
	}, nil
}

// StartVerifyTask 启动验证任务
func (s *ServiceImpl) StartVerifyTask(ctx context.Context, opts *VerifyOptions) (string, error) {
	clog.Debugf(ctx, "Starting verify task with options: %+v", opts)
	taskID, err := s.verifier.Start(ctx, opts)
	if err != nil {
		clog.Errorf(ctx, "Failed to start verify task: %v", err)
		return "", fmt.Errorf("failed to start verify task: %w", err)
	}
	clog.Infof(ctx, "Verify task started with ID: %s", taskID)
	return taskID, nil
}

// GetVerifyTask 获取验证任务
func (s *ServiceImpl) GetVerifyTask(ctx context.Context, taskID string) (*VerifyTask, error) {
	clog.Debugf(ctx, "Getting verify task with ID: %s", taskID)
	task, err := s.verifier.GetTask(ctx, taskID)
	if err != nil {
		clog.Errorf(ctx, "Failed to get verify task [%s]: %v", taskID, err)
		return nil, fmt.Errorf("failed to get verify task: %w", err)
	}
	return task, nil
}

// GetVerifyTasks 获取验证任务列表
func (s *ServiceImpl) GetVerifyTasks(ctx context.Context) ([]*VerifyTask, error) {
	return s.verifier.GetTasks(), nil
}

// GetVerifyProgress 获取验证进度
func (s *ServiceImpl) GetVerifyProgress(ctx context.Context, taskID string) (*VerifyProgress, error) {
	progress := s.verifier.GetTaskProgress(taskID)
	if progress == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return progress, nil
}

// CancelVerifyTask 取消验证任务
func (s *ServiceImpl) CancelVerifyTask(ctx context.Context, taskID string) error {
	clog.Debugf(ctx, "Canceling verify task with ID: %s", taskID)
	err := s.verifier.CancelTask(ctx, taskID)
	if err != nil {
		clog.Errorf(ctx, "Failed to cancel verify task [%s]: %v", taskID, err)
		return fmt.Errorf("failed to cancel verify task: %w", err)
	}
	clog.Infof(ctx, "Verify task [%s] canceled", taskID)
	return nil
}

// StartScheduleVerify 启动定时验证
func (s *ServiceImpl) StartScheduleVerify(ctx context.Context, cfg *ScheduleConfig) error {
	return s.verifier.StartSchedule(ctx, cfg)
}

// SearchComics 搜索漫画
func (s *ServiceImpl) SearchComics(ctx context.Context, filter *ComicFilter) ([]Comic, error) {
	clog.Debugf(ctx, "Searching comics with filter: %+v", filter)
	comics, err := s.storage.Find(ctx, filter)
	if err != nil {
		clog.Errorf(ctx, "Failed to search comics: %v", err)
		return nil, fmt.Errorf("failed to search comics: %w", err)
	}
	clog.Infof(ctx, "Found %d comics matching filter", len(comics))
	return comics, nil
}

// GetInvalidComics 获取所有无效漫画
func (s *ServiceImpl) GetInvalidComics(ctx context.Context, filter *ComicFilter) ([]Comic, error) {
	clog.Debugf(ctx, "Getting invalid comics with filter: %+v", filter)
	comics, err := s.storage.Find(ctx, NewInvalidComicFilter(func(f *ComicFilter) {
		*f = *filter
	}))
	if err != nil {
		clog.Errorf(ctx, "Failed to get invalid comics: %v", err)
		return nil, fmt.Errorf("failed to get invalid comics: %w", err)
	}
	clog.Infof(ctx, "Found %d invalid comics matching filter", len(comics))
	return comics, nil
}

// GetComicInfo 获取漫画信息
func (s *ServiceImpl) GetComicInfo(ctx context.Context, id string) (Comic, error) {
	clog.Debugf(ctx, "Getting comic info with ID: %s", id)
	comic, err := s.storage.Get(ctx, id)
	if err != nil {
		clog.Errorf(ctx, "Failed to get comic info [%s]: %v", id, err)
		return nil, fmt.Errorf("failed to get comic info: %w", err)
	}
	return comic, nil
}
