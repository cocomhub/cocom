// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cocomhub/cocom/pkg/storage"
)

func (h *helper) ReplicateMore(ctx context.Context, dst storage.Storage, prefix string, f IndexFilter) ([]ArchiveMeta, error) {
	m := h.Manager()
	items, err := m.List(ctx, f)
	if err != nil {
		return nil, err
	}
	for i := range items {
		meta := &items[i]
		err := h.replicate(ctx, m, dst, prefix, meta)
		if err != nil {
			return items[:i], err
		}
	}
	return items, nil
}

func (h *helper) Replicate(ctx context.Context, dst storage.Storage, prefix string, meta *ArchiveMeta) error {
	m := h.Manager()
	return h.replicate(ctx, m, dst, prefix, meta)
}

func (h *helper) replicate(ctx context.Context, m Manager, dst storage.Storage, prefix string, meta *ArchiveMeta) error {
	if err := meta.Validate(); err != nil {
		return err
	}
	base := filepath.Base(meta.Path)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ErrInvalidArgument
	}

	backend := dst.Name()
	key := storage.MustPath(prefix, base)
	locIdx := -1
	for i := range meta.Locators {
		if meta.Locators[i].Backend == backend {
			meta.Locators[i] = storage.StorageLocator{Backend: backend, Key: key, ReplicaHealth: storage.NewHealthy(false)}
			locIdx = i
		}
	}
	if locIdx == -1 {
		meta.Locators = append(meta.Locators, storage.StorageLocator{Backend: backend, Key: key, ReplicaHealth: storage.NewHealthy(false)})
		locIdx = len(meta.Locators) - 1
	}
	if err := m.Put(ctx, meta); err != nil {
		return err
	}

	fd, err := os.Open(meta.Path)
	if err != nil {
		return err
	}
	defer fd.Close()
	var objMeta *storage.ObjectMeta
	for range 3 {
		if objMeta, err = dst.Put(ctx, key, fd, storage.WithOverwrite(true)); err != nil {
			slog.ErrorContext(ctx, "replicate put failed", slog.String("key", key), slog.String("err", err.Error()))
			continue
		}
		if objMeta.ETag != meta.Checksum.Value {
			if objMeta.ETag != "" && meta.Checksum.Algorithm == "md5" {
				slog.ErrorContext(ctx, "replicate put etag not match", slog.String("key", key), slog.String("err", "ETag mismatch"), slog.String("etag", objMeta.ETag), slog.String("expected", meta.Checksum.Value))
				continue
			}
			healthy, err := checksumFromStorage(ctx, dst, key, meta.Checksum)
			if err != nil {
				return fmt.Errorf("replicate checksumFromStorage: %w", err)
			}
			if !healthy {
				slog.ErrorContext(ctx, "replicate put checksum from storage unhealthy", slog.String("key", key), slog.String("err", "checksum not match"))
				continue
			}
		}
		slog.InfoContext(ctx, "replicate put success", slog.String("uri", storage.MustURI(dst, key)))
		break
	}
	if err != nil {
		return fmt.Errorf("replicate put failed: %w", err)
	}

	meta.Locators[locIdx] = storage.StorageLocator{Backend: backend, Key: key, ReplicaHealth: storage.NewHealthy(true)}
	return m.Put(ctx, meta)
}
