// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"time"

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
	tmp := newTempFileKey(key)
	if _, putErr := s.st.Put(ctx, tmp, bytes.NewReader(data), storage.WithOverwrite(true)); putErr != nil {
		return putErr
	}
	if _, mErr := s.st.Move(ctx, tmp, key); mErr != nil {
		_ = s.st.Delete(ctx, tmp)
		return mErr
	}
	return nil
}
func (s *IndexStoreFS) Get(ctx context.Context, id int) (*ArchiveMeta, error) {
	key := s.key(id)
	rc, _, err := s.st.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("fs: get err %w: id=%d, %s", ErrNotFound, id, err.Error())
	}
	defer rc.Close()
	var meta ArchiveMeta
	if err := json.NewDecoder(rc).Decode(&meta); err != nil {
		return nil, fmt.Errorf("fs: decode err %w: id=%d", err, id)
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
	tmp := newTempFileKey(key)
	if _, putErr := s.st.Put(ctx, tmp, bytes.NewReader(data), storage.WithOverwrite(true)); putErr != nil {
		return putErr
	}
	if _, mErr := s.st.Move(ctx, tmp, key); mErr != nil {
		_ = s.st.Delete(ctx, tmp)
		return mErr
	}
	return nil
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
			if IsNotFound(err) {
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

// newTempFileKey 生成带随机后缀的临时 key，消除跨调用复用固定 tmp 名的碰撞：
// - 并发 Create 同 ID：各自独立临时名，Put+Move 完整序线互不干扰；
// - 残留 tmp 不会因复用固定名被后续调用覆盖（覆盖本身无害，但幂等测试依赖独立键）。
// 采用纳秒时间 + 8 位安全随机，与 localfs 内部 tmp 命名错开；随机仅用于避免确定性攻击，
// 不强求 crypto 强度（无需安全随机源）。
func newTempFileKey(key string) string {
	return fmt.Sprintf("%s.tmp-%d-%08x", key, time.Now().UnixNano(), int(fastRand()))
}

func fastRand() uint64 {
	x := uint64(time.Now().UnixNano())
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	return x * 2685821657736338717
}
