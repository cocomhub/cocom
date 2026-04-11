// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"time"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
)

type ArchiveMeta struct {
	ID        int
	Name      string
	Path      string
	Size      int64
	FileCount int
	ModTime   time.Time
	Version   int `json:"version"`
	Type      archive.Type
	Checksum  storage.Checksum         `json:"checksum"`
	Locators  []storage.StorageLocator `json:"locators"`
	Health    storage.ReplicaHealth    `json:"health"`
}

type IndexFilter struct {
	ID     int
	Name   string
	Before time.Time
	After  time.Time
}
