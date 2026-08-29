// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"os"

	"github.com/cocomhub/cocom/pkg/storage"
)

type Policy struct {
	Name string
}

func (h *helper) ApplyRetention(ctx context.Context, f IndexFilter) (int, error) {
	m := h.Manager()
	items, err := m.List(ctx, f)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range items {
		meta := &items[i]
		if meta.Path == "" {
			return n, ErrInvalidArgument
		}
		for _, loc := range meta.Locators {
			if !loc.Healthy {
				continue
			}

			if err := os.Remove(meta.Path); err != nil && !os.IsNotExist(err) {
				return n, err
			}
			meta.ReplicaHealth = storage.NewHealthy(false)
			if err := m.Put(ctx, meta); err != nil {
				return n, err
			}
			n++
			break
		}
	}
	return n, nil
}
