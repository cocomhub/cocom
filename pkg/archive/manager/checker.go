// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"log/slog"
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

func (h *helper) Check(ctx context.Context, id int, force bool) (*ArchiveMeta, error) {
	m := h.Manager()
	meta, err := m.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("helper: check id=%d, err: %w", id, err)
	}
	if err = meta.Validate(); err != nil {
		return meta, fmt.Errorf("helper: check id=%d, err: %w", id, err)
	}

	healthy, err := checksumFromFile(meta.Path, meta.Checksum)
	if err != nil {
		return meta, fmt.Errorf("helper: check id=%d, err: %w", id, err)
	}
	meta.ReplicaHealth = storage.NewHealthy(healthy)

	for i, locator := range meta.Locators {
		if locator.Healthy && !force {
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
			return meta, fmt.Errorf("helper: check id=%d, err: %w", id, err)
		}
		meta.Locators[i].ReplicaHealth = storage.NewHealthy(healthy)
	}
	if err := m.Put(ctx, meta); err != nil {
		return meta, fmt.Errorf("helper: check id=%d, err: %w", id, err)
	}
	return meta, nil
}

func checksumFromStorage(ctx context.Context, s storage.Storage, key string, c storage.Checksum) (bool, error) {
	meta, err := s.Stat(ctx, key)
	if err != nil {
		if storage.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if meta.ETag == c.Value {
		slog.Info("ETag matches", "key", key, "c", c, "etag", meta.ETag)
		return true, nil
	} else if meta.ETag != "" && c.Algorithm == "md5" {
		slog.Warn("ETag does not match, but algorithm is md5", "key", key, "checksum", c, "etag", meta.ETag)
	}

	r, _, err := s.Get(ctx, key)
	if err != nil {
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
		if storage.IsNotFound(err) {
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
