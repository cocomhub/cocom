// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"os"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/localfs"
)

type Manager interface {
	PrimaryStorage() storage.Storage
	Algorithm() archive.Type
	Register(ctx context.Context, meta ArchiveMeta) error
	Put(ctx context.Context, meta ArchiveMeta) error
	Get(ctx context.Context, id int) (ArchiveMeta, error)
	Find(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error)
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error)
	Replicate(ctx context.Context, dest IndexStore, f IndexFilter) (int, error)
	Check(ctx context.Context, id int) error
}

type manager struct {
	cfg     Config
	rootDir string
	algo    archive.Type
	index   IndexStore
}

func New(cfg ...Config) Manager {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	var index IndexStore
	if c.Index.Type == "file" {
		fs, ok := storage.Get(c.Index.FileStoreName)
		if !ok {
			panic(fmt.Errorf("index file store %q not found", c.Index.FileStoreName))
		}
		index = NewIndexStoreFS(fs, c.Index.FileStorePrefix)
	} else {
		index = NewMemoryIndexStore()
	}

	return &manager{
		cfg:     c,
		rootDir: c.RootDir,
		algo:    c.Algorithm,
		index:   index,
	}
}

func (m *manager) PrimaryStorage() storage.Storage {
	return localfs.New("archive-primary", m.rootDir)
}

func (m *manager) Algorithm() archive.Type {
	return m.algo
}

func (m *manager) Register(ctx context.Context, meta ArchiveMeta) error {
	if meta.ID == 0 || meta.Path == "" {
		return ErrInvalidArgument
	}
	meta.Type = m.algo
	_, err := m.index.Get(ctx, meta.ID)
	if err == nil {
		return ErrAlreadyExists
	}
	if err != nil && err != ErrNotFound {
		return err
	}
	return m.index.Create(ctx, meta)
}

func (m *manager) Put(ctx context.Context, meta ArchiveMeta) error {
	if meta.ID == 0 || meta.Path == "" {
		return ErrInvalidArgument
	}
	meta.Type = m.algo
	_, err := m.index.Get(ctx, meta.ID)
	if err == nil {
		return m.index.Update(ctx, meta)
	}
	if err != nil && err != ErrNotFound {
		return err
	}
	return m.index.Create(ctx, meta)
}

func (m *manager) Get(ctx context.Context, id int) (ArchiveMeta, error) {
	return m.index.Get(ctx, id)
}

func (m *manager) Find(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error) {
	return m.index.List(ctx, f)
}

func (m *manager) Delete(ctx context.Context, id int) error {
	return m.index.Delete(ctx, id)
}

func (m *manager) List(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error) {
	return m.index.List(ctx, f)
}

func (m *manager) Replicate(ctx context.Context, dest IndexStore, f IndexFilter) (int, error) {
	items, err := m.index.List(ctx, f)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, meta := range items {
		if _, e := dest.Get(ctx, meta.ID); e == nil {
			if err := dest.Update(ctx, meta); err != nil {
				return n, err
			}
			n++
			continue
		}
		if err := dest.Create(ctx, meta); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (m *manager) Check(ctx context.Context, id int) error {
	meta, err := m.index.Get(ctx, id)
	if err != nil {
		return err
	}
	if meta.Path == "" {
		return ErrInvalidArgument
	}
	st, e := os.Stat(meta.Path)
	if e != nil {
		return ErrNotFound
	}
	if st.IsDir() {
		return ErrInvalidArgument
	}
	return nil
}
