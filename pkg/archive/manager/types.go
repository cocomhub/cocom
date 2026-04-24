// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"
	"time"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
)

type ArchiveMeta struct {
	ID        int                      `json:"id" bson:"id"`
	Name      string                   `json:"name" bson:"name"`
	Path      string                   `json:"path" bson:"path"`
	Size      int64                    `json:"size" bson:"size"`
	FileCount int                      `json:"file_count" bson:"file_count"`
	ModTime   time.Time                `json:"mod_time" bson:"mod_time"`
	Version   int                      `json:"version" bson:"version"`
	Type      archive.Type             `json:"type" bson:"type"`
	Checksum  storage.Checksum         `json:"checksum" bson:"checksum"`
	Locators  []storage.StorageLocator `json:"locators" bson:"locators"`
	storage.ReplicaHealth
}

func (m *ArchiveMeta) Validate() error {
	if m == nil || m.ID == 0 || m.Path == "" {
		return fmt.Errorf("meta 无效: %+v", m)
	}
	return nil
}

type IndexFilter struct {
	ID     int
	Name   string
	Before time.Time
	After  time.Time
}
