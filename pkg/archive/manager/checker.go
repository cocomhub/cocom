// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/util"
)

type CheckReport struct {
	ID        int
	Path      string
	Size      int64
	Algorithm string
	Expected  string
	Actual    string
	Healthy   bool
	CheckedAt time.Time
}

func (h *helper) CheckAndUpdate(ctx context.Context, id int) (*ArchiveMeta, error) {
	m := h.Manager()
	meta, err := m.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if meta.Path == "" {
		return meta, ErrInvalidArgument
	}

	healthy, err := checksumFromFile(meta.Path, meta.Checksum)
	if err != nil {
		return meta, err
	}
	meta.ReplicaHealth = storage.NewHealthy(healthy)

	for i, locator := range meta.Locators {
		if locator.Healthy {
			continue
		}

		s, ok := storage.Get(locator.Backend)
		if !ok {
			continue
		}

		key := strings.TrimPrefix(locator.Key, "/")
		key = filepath.ToSlash(key)
		healthy, err := checksumFromStorage(ctx, s, key, meta.Checksum)
		if err != nil {
			return meta, err
		}
		meta.Locators[i].ReplicaHealth = storage.NewHealthy(healthy)
	}
	if err := m.Put(ctx, meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func checksumFromStorage(ctx context.Context, s storage.Storage, key string, c storage.Checksum) (bool, error) {
	r, _, err := s.Get(ctx, key)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer r.Close()

	switch strings.ToLower(c.Algorithm) {
	case "sha256":
		actual, err := util.Sha256(r)
		if err != nil {
			return false, err
		}
		return actual == c.Value, nil
	default:
		actual, err := util.MD5(r)
		if err != nil {
			return false, err
		}
		return actual == c.Value, nil
	}
}

func checksumFromFile(path string, c storage.Checksum) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()
	switch strings.ToLower(c.Algorithm) {
	case "sha256":
		actual, err := util.Sha256(f)
		if err != nil {
			return false, err
		}
		return actual == c.Value, nil
	default:
		actual, err := util.MD5(f)
		if err != nil {
			return false, err
		}
		return actual == c.Value, nil
	}
}
