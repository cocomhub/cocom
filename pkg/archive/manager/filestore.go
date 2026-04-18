// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"

	"github.com/cocomhub/cocom/pkg/storage"
)

type IndexStoreFS struct {
	st     storage.Storage
	prefix string
}

func NewIndexStoreFS(st storage.Storage, prefix string) *IndexStoreFS {
	return &IndexStoreFS{st: st, prefix: prefix}
}

func (s *IndexStoreFS) key(id int) string {
	return path.Join(s.prefix, fmt.Sprintf("%d.json", id))
}

func (s *IndexStoreFS) Create(ctx context.Context, meta *ArchiveMeta) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	key := s.key(meta.ID)
	exists, err := s.st.Exists(ctx, key)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyExists
	}
	data, err := json.MarshalIndent(meta, "", " ")
	if err != nil {
		return err
	}
	tmp := key + ".tmp"
	if _, err := s.st.Put(ctx, tmp, bytes.NewReader(data), storage.WithOverwrite(true)); err != nil {
		return err
	}
	_, err = s.st.Move(ctx, tmp, key)
	return err
}

func (s *IndexStoreFS) Get(ctx context.Context, id int) (*ArchiveMeta, error) {
	key := s.key(id)
	rc, _, err := s.st.Get(ctx, key)
	if err != nil {
		return nil, ErrNotFound
	}
	defer rc.Close()
	var meta ArchiveMeta
	if err := json.NewDecoder(rc).Decode(&meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *IndexStoreFS) Update(ctx context.Context, meta *ArchiveMeta) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	key := s.key(meta.ID)
	exists, err := s.st.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	data, err := json.MarshalIndent(meta, "", " ")
	if err != nil {
		return err
	}
	tmp := key + ".tmp"
	if _, err := s.st.Put(ctx, tmp, bytes.NewReader(data), storage.WithOverwrite(true)); err != nil {
		return err
	}
	_ = s.st.Delete(ctx, key)
	_, err = s.st.Move(ctx, tmp, key)
	return err
}

func (s *IndexStoreFS) Delete(ctx context.Context, id int) error {
	key := s.key(id)
	exists, err := s.st.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return s.st.Delete(ctx, key)
}

func (s *IndexStoreFS) List(ctx context.Context, f IndexFilter) ([]ArchiveMeta, error) {
	if f.ID != 0 {
		m, err := s.Get(ctx, f.ID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, nil
			}
			return nil, err
		}
		if f.Name != "" && m.Name != f.Name {
			return nil, nil
		}
		if !f.Before.IsZero() && !m.ModTime.Before(f.Before) {
			return nil, nil
		}
		if !f.After.IsZero() && !m.ModTime.After(f.After) {
			return nil, nil
		}
		return []ArchiveMeta{*m}, nil
	}
	entries, err := s.st.List(ctx, s.prefix)
	if err != nil {
		return nil, err
	}
	res := make([]ArchiveMeta, 0, len(entries))
	for _, e := range entries {
		if !hasJSONSuffix(e.Key) {
			continue
		}
		rc, _, err := s.st.Get(ctx, e.Key)
		if err != nil {
			continue
		}
		var m ArchiveMeta
		decodeErr := json.NewDecoder(rc).Decode(&m)
		rc.Close()
		if decodeErr != nil {
			continue
		}
		if f.Name != "" && m.Name != f.Name {
			continue
		}
		if !f.Before.IsZero() && !m.ModTime.Before(f.Before) {
			continue
		}
		if !f.After.IsZero() && !m.ModTime.After(f.After) {
			continue
		}
		res = append(res, m)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res, nil
}

func hasJSONSuffix(key string) bool {
	n := len(key)
	return n >= 5 && key[n-5:] == ".json"
}
