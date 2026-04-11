// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"sort"
	"sync"
)

type IndexStore interface {
	Create(ctx context.Context, meta ArchiveMeta) error
	Get(ctx context.Context, id int) (ArchiveMeta, error)
	Update(ctx context.Context, meta ArchiveMeta) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error)
}

type MemoryIndexStore struct {
	mu sync.RWMutex
	m  map[int]ArchiveMeta
}

func NewMemoryIndexStore() *MemoryIndexStore {
	return &MemoryIndexStore{m: make(map[int]ArchiveMeta)}
}

func (s *MemoryIndexStore) Create(ctx context.Context, meta ArchiveMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[meta.ID] = meta
	return nil
}

func (s *MemoryIndexStore) Get(ctx context.Context, id int) (ArchiveMeta, error) {
	s.mu.RLock()
	v, ok := s.m[id]
	s.mu.RUnlock()
	if !ok {
		return ArchiveMeta{}, ErrNotFound
	}
	return v, nil
}

func (s *MemoryIndexStore) Update(ctx context.Context, meta ArchiveMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[meta.ID]
	if !ok {
		return ErrNotFound
	}
	s.m[meta.ID] = meta
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
		res = append(res, v)
	}
	s.mu.RUnlock()
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res, nil
}
