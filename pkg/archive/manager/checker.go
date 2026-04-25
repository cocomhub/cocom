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

		savePath := ""
		if !meta.ReplicaHealth.Healthy {
			savePath = meta.Path
		}

		key := strings.TrimPrefix(locator.Key, "/")
		key = filepath.ToSlash(key)
		healthy, err := checksumFromStorage(ctx, s, key, meta.Checksum, savePath)
		if err != nil {
			return meta, fmt.Errorf("helper: check id=%d, err: %w", id, err)
		}
		if meta.ReplicaHealth.Healthy && !healthy {
			slog.WarnContext(ctx, "storage locator unhealthy, try replicate", "key", key, "backend", locator.Backend)
			if err := h.replicate(ctx, m, s, filepath.Dir(locator.Key), meta); err != nil {
				slog.ErrorContext(ctx, "replicate file", "key", key, "backend", locator.Backend, "err", err)
			}
			healthy, err = checksumFromStorage(ctx, s, key, meta.Checksum, savePath)
			if err != nil {
				return meta, fmt.Errorf("helper: check2 id=%d, err: %w", id, err)
			}
		}

		meta.Locators[i].ReplicaHealth = storage.NewHealthy(healthy)
	}
	if err := m.Put(ctx, meta); err != nil {
		return meta, fmt.Errorf("helper: check id=%d, err: %w", id, err)
	}
	return meta, nil
}

func checksumFromStorage(ctx context.Context, s storage.Storage, key string, c storage.Checksum, savePath string) (healthy bool, err error) {
	meta, err := s.Stat(ctx, key)
	if err != nil {
		if storage.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if meta.ETag == c.Value {
		if savePath == "" {
			slog.Info("ETag matches", "key", key, "c", c, "etag", meta.ETag)
			return true, nil
		}
	} else if meta.ETag != "" && c.Algorithm == "md5" {
		slog.Warn("ETag does not match, but algorithm is md5", "key", key, "checksum", c, "etag", meta.ETag)
	}

	removeSavePath := false
	if savePath == "" {
		tmpFile, err2 := os.CreateTemp("", filepath.Base(key)+"-*")
		if err2 == nil {
			tmpFile.Close()
			savePath = tmpFile.Name()
			removeSavePath = true
		}
	}

	r, _, err := s.Get(ctx, key, storage.WithTrySaveFilePath(savePath))
	if err != nil {
		return false, err
	}
	defer r.Close()
	defer func() {
		if healthy && s.CanRePut() {
			f, err2 := os.Open(savePath)
			if err2 != nil {
				slog.WarnContext(ctx, "checksum succ re put file open fail", "key", key, "savePath", savePath, "err", err)
				return
			}
			defer f.Close()
			slog.InfoContext(ctx, "checksum succ re put file", "key", key, "savePath", savePath, "etag", meta.ETag)
			if _, err2 := s.Put(ctx, key, f, storage.WithOverwrite(true), storage.WithExpectedETag(func() string {
				if c.Algorithm == "md5" {
					return c.Value
				}
				return meta.ETag
			}())); err2 != nil {
				slog.WarnContext(ctx, "checksum succ re put file fail", "key", key, "savePath", savePath, "err", err2)
				return
			}
		}
		if removeSavePath {
			os.Remove(savePath)
		}
	}()

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
