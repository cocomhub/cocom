// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"os"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
)

type Manager interface {
	Algorithm() archive.Type
	MetaRecordFileList() bool
	Replicates() []storage.Storage
	Register(ctx context.Context, meta *ArchiveMeta) error
	Put(ctx context.Context, meta *ArchiveMeta) error
	Get(ctx context.Context, id int) (*ArchiveMeta, error)
	Find(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error)
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error)
	Replicate(ctx context.Context, dest IndexStore, f IndexFilter) (int, error)
	Check(ctx context.Context, id int) error
}

type manager struct {
	cfg        Config
	algo       archive.Type
	index      IndexStore
	replicates []storage.Storage
}

func New(cfg ...Config) Manager {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	var replicates []storage.Storage
	for _, s := range c.Replicates {
		replicates = append(replicates, storage.MustGet(s))
	}

	var index IndexStore
	if f, ok := indexFactories[c.Index.Type]; ok {
		index = f(c.Index)
	} else {
		panic(fmt.Errorf("index store type %q not registered", c.Index.Type))
	}

	return &manager{
		cfg:        c,
		algo:       c.Algorithm,
		index:      index,
		replicates: replicates,
	}
}

func (m *manager) Algorithm() archive.Type {
	return m.algo
}

func (m *manager) MetaRecordFileList() bool {
	return m.cfg.MetaRecordFileList
}

func (m *manager) Replicates() []storage.Storage {
	return m.replicates
}

func (m *manager) Register(ctx context.Context, meta *ArchiveMeta) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	meta.Type = m.algo
	_, err := m.index.Get(ctx, meta.ID)
	if err == nil {
		return ErrAlreadyExists
	}
	if err != nil && !IsNotFound(err) {
		return err
	}
	return m.index.Create(ctx, meta)
}

func (m *manager) Put(ctx context.Context, meta *ArchiveMeta) error {
	if meta.ID == 0 || meta.Path == "" {
		return ErrInvalidArgument
	}
	meta.Type = m.algo
	_, err := m.index.Get(ctx, meta.ID)
	if err == nil {
		return m.index.Update(ctx, meta)
	}
	if err != nil && !IsNotFound(err) {
		return err
	}
	return m.index.Create(ctx, meta)
}

func (m *manager) Get(ctx context.Context, id int) (*ArchiveMeta, error) {
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
	for i := range items {
		meta := &items[i]
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
