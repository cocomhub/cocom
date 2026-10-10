// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/cocomhub/cocom/pkg/storage"
)

type IndexStore interface {
	Create(ctx context.Context, meta *ArchiveMeta) error
	Get(ctx context.Context, id int) (*ArchiveMeta, error)
	Update(ctx context.Context, meta *ArchiveMeta) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error)
}

type MemoryIndexStore struct {
	mu sync.RWMutex
	m  map[int]*ArchiveMeta
}

func NewMemoryIndexStore() *MemoryIndexStore {
	return &MemoryIndexStore{m: make(map[int]*ArchiveMeta)}
}

// cloneMeta 深拷贝 ArchiveMeta（含切片字段），避免把内部指针泄漏给调用方：
// 调用方对返回值的改写会绕过本 store 的锁，造成 data race 与历史丢失。
func cloneMeta(m *ArchiveMeta) *ArchiveMeta {
	if m == nil {
		return nil
	}
	c := *m
	if m.FileList != nil {
		c.FileList = append([]string(nil), m.FileList...)
	}
	if m.History != nil {
		c.History = append([]ArchiveMeta(nil), m.History...)
	}
	if m.Locators != nil {
		c.Locators = append([]storage.StorageLocator(nil), m.Locators...)
	}
	return &c
}

func (s *MemoryIndexStore) Create(ctx context.Context, meta *ArchiveMeta) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[meta.ID] = cloneMeta(meta)
	return nil
}

func (s *MemoryIndexStore) Get(ctx context.Context, id int) (*ArchiveMeta, error) {
	s.mu.RLock()
	v, ok := s.m[id]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("memory: get err %w: id=%d", ErrNotFound, id)
	}
	return cloneMeta(v), nil
}

func (s *MemoryIndexStore) Update(ctx context.Context, meta *ArchiveMeta) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[meta.ID]
	if !ok {
		return ErrNotFound
	}
	s.m[meta.ID] = cloneMeta(meta)
	return nil
}

func (s *MemoryIndexStore) Delete(ctx context.Context, id int) error {
	s.mu.Lock()
	delete(s.m, id)
	s.mu.Unlock()
	return nil
}

func (s *MemoryIndexStore) List(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error) {
	var res []ArchiveMeta
	s.mu.RLock()
	for _, v := range s.m {
		// copy value to avoid data race if caller modifies
		if f.ID != 0 && v.ID != f.ID {
			continue
		}
		if f.Name != "" && v.Name != f.Name {
			continue
		}
		if !f.Before.IsZero() && !v.ModTime.Before(f.Before) {
			continue
		}
		if !f.After.IsZero() && !v.ModTime.After(f.After) {
			continue
		}
		res = append(res, *cloneMeta(v))
	}
	s.mu.RUnlock()
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res, nil
}
