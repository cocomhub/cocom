// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/cocomhub/cocom/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMongoDefaultEncodeDecode(t *testing.T) {
	m := &mongoIndexStore{idField: "id", nameField: "name", modTimeField: "modTime"}
	m.filterBuilder = m.defaultFilter
	m.encode = m.defaultEncode
	m.decode = m.defaultDecode
	now := time.Now().UTC().Round(time.Second)
	meta := ArchiveMeta{
		ID:        101,
		Name:      "foo",
		Path:      "/p/a",
		Size:      123,
		FileCount: 3,
		ModTime:   now,
		Version:   1,
		Type:      archive.TypeSingle,
		Checksum: storage.Checksum{
			Algorithm: "md5",
			Value:     "abc123",
		},
		Locators: []storage.StorageLocator{
			{
				Backend:       "dstfs",
				Key:           "archive/foo.7z",
				ReplicaHealth: storage.ReplicaHealth{Healthy: true, CheckedAt: now},
			},
		},
		Health: storage.ReplicaHealth{Healthy: true, CheckedAt: now},
	}
	doc, err := m.encode(meta)
	if err != nil {
		t.Fatalf("encode err: %v", err)
	}
	got, err := m.decode(doc)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if got.ID != meta.ID || got.Name != meta.Name || !got.ModTime.Equal(meta.ModTime) {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", got, meta)
	}
	if got.Checksum != meta.Checksum {
		t.Fatalf("checksum mismatch: %+v vs %+v", got.Checksum, meta.Checksum)
	}
	if len(got.Locators) != 1 || got.Locators[0].Backend != meta.Locators[0].Backend || got.Locators[0].Key != meta.Locators[0].Key {
		t.Fatalf("locator mismatch: %+v vs %+v", got.Locators, meta.Locators)
	}
	if got.Health.Healthy != meta.Health.Healthy || !got.Health.CheckedAt.Equal(meta.Health.CheckedAt) {
		t.Fatalf("health mismatch: %+v vs %+v", got.Health, meta.Health)
	}
}

func TestMongoDefaultFilter(t *testing.T) {
	m := &mongoIndexStore{idField: "id", nameField: "name", modTimeField: "modTime"}
	m.filterBuilder = m.defaultFilter
	now := time.Now()
	f := IndexFilter{Name: "x", After: now.Add(-time.Hour), Before: now.Add(time.Hour)}
	q := m.filterBuilder(f)
	if q["id"] != nil {
		t.Fatalf("unexpected id in filter")
	}
	if q["name"] != "x" {
		t.Fatalf("name mismatch in filter")
	}
	if _, ok := q["modTime"]; !ok {
		t.Fatalf("missing modTime range in filter")
	}
}

func TestComicInfoFilter(t *testing.T) {
	m := NewComicInfoArchiveIndexStore(nil).(*mongoIndexStore)
	now := time.Now()
	q := m.filterBuilder(IndexFilter{Name: "n", After: now})
	if q["cid"] != nil {
		t.Fatalf("cid should be nil when ID not set")
	}
	if q["archive.name"] != "n" {
		t.Fatalf("archive.name mismatch")
	}
	if _, ok := q["archive.modTime"]; !ok {
		t.Fatalf("missing archive.modTime range")
	}
}

func TestMongoDefaultDecodeMapValues(t *testing.T) {
	m := &mongoIndexStore{idField: "id", nameField: "name", modTimeField: "modTime"}
	m.decode = m.defaultDecode
	now := time.Now().UTC().Round(time.Second)

	got, err := m.decode(bson.M{
		"id":      int32(7),
		"name":    "mapped",
		"modTime": now,
		"checksum": bson.M{
			"algorithm": "sha256",
			"value":     "deadbeef",
		},
		"locators": []any{
			bson.M{
				"backend":   "dstfs",
				"key":       "rep/mapped.7z",
				"healthy":   true,
				"checkedAt": now,
			},
		},
		"health": bson.M{
			"healthy":   true,
			"checkedAt": now,
		},
	})
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if got.Checksum.Algorithm != "sha256" || got.Checksum.Value != "deadbeef" {
		t.Fatalf("checksum decode mismatch: %+v", got.Checksum)
	}
	if len(got.Locators) != 1 || got.Locators[0].Backend != "dstfs" || got.Locators[0].Key != "rep/mapped.7z" {
		t.Fatalf("locator decode mismatch: %+v", got.Locators)
	}
	if !got.Health.Healthy || !got.Health.CheckedAt.Equal(now) {
		t.Fatalf("health decode mismatch: %+v", got.Health)
	}
}

func TestComicInfoDecodeEmbeddedMapValues(t *testing.T) {
	m := NewComicInfoArchiveIndexStore(nil).(*mongoIndexStore)
	now := time.Now().UTC().Round(time.Second)

	got, err := m.decode(bson.M{
		"cid": int32(11),
		"archive": bson.M{
			"id":      int32(11),
			"name":    "embedded",
			"modTime": now,
			"checksum": bson.M{
				"algorithm": "md5",
				"value":     "cafebabe",
			},
			"locators": []any{
				bson.M{
					"store":     "legacy-store",
					"key":       "archive/embedded.7z",
					"healthy":   false,
					"checkedAt": now,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if got.ID != 11 || got.Name != "embedded" {
		t.Fatalf("embedded decode mismatch: %+v", got)
	}
	if got.Checksum.Algorithm != "md5" || got.Checksum.Value != "cafebabe" {
		t.Fatalf("embedded checksum mismatch: %+v", got.Checksum)
	}
	if len(got.Locators) != 1 || got.Locators[0].Backend != "legacy-store" || got.Locators[0].Key != "archive/embedded.7z" {
		t.Fatalf("embedded locator mismatch: %+v", got.Locators)
	}
}

func TestSkipIntegrationWhenNoEnv(t *testing.T) {
	if os.Getenv("MONGO_TEST") == "" {
		t.Skip("MONGO_TEST not set")
	}
}

func TestMongoIndexStoreIntegrationCRUDAndList(t *testing.T) {
	if os.Getenv("MONGO_TEST") == "" {
		t.Skip("MONGO_TEST not set")
	}

	ctx := context.Background()
	coll := mongowrap.DB("cocom").Collection(fmt.Sprintf("archive_index_test_%d", time.Now().UnixNano()))
	defer coll.Drop(ctx)

	store := NewMongoIndexStore(coll)
	now := time.Now().UTC().Round(time.Second)
	meta := ArchiveMeta{
		ID:      501,
		Name:    "mongo-generic",
		Path:    "/tmp/mongo-generic.7z",
		ModTime: now,
		Version: 1,
		Type:    archive.TypeSingle,
		Checksum: storage.Checksum{
			Algorithm: "md5",
			Value:     "001122",
		},
		Locators: []storage.StorageLocator{
			{
				Backend:       "backup",
				Key:           "archive/mongo-generic.7z",
				ReplicaHealth: storage.ReplicaHealth{Healthy: true, CheckedAt: now},
			},
		},
		Health: storage.ReplicaHealth{Healthy: true, CheckedAt: now},
	}
	if err := store.Create(ctx, meta); err != nil {
		t.Fatalf("create err: %v", err)
	}

	got, err := store.Get(ctx, meta.ID)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if got.Checksum != meta.Checksum || len(got.Locators) != 1 || got.Locators[0].Backend != "backup" {
		t.Fatalf("generic get mismatch: %+v", got)
	}

	meta.Name = "mongo-generic-updated"
	if err := store.Update(ctx, meta); err != nil {
		t.Fatalf("update err: %v", err)
	}

	items, err := store.List(ctx, IndexFilter{Name: meta.Name, After: now.Add(-time.Minute), Before: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("list err: %v", err)
	}
	if len(items) != 1 || items[0].ID != meta.ID {
		t.Fatalf("list mismatch: %+v", items)
	}

	if err := store.Delete(ctx, meta.ID); err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if _, err := store.Get(ctx, meta.ID); err != ErrNotFound {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestComicInfoArchiveIndexStoreIntegrationCRUDAndList(t *testing.T) {
	if os.Getenv("MONGO_TEST") == "" {
		t.Skip("MONGO_TEST not set")
	}

	ctx := context.Background()
	coll := mongowrap.DB("cocom").Collection(fmt.Sprintf("comic_info_archive_test_%d", time.Now().UnixNano()))
	defer coll.Drop(ctx)

	store := NewComicInfoArchiveIndexStore(coll)
	now := time.Now().UTC().Round(time.Second)
	meta := ArchiveMeta{
		ID:      601,
		Name:    "embedded-generic",
		Path:    "/tmp/embedded.7z",
		ModTime: now,
		Version: 1,
		Type:    archive.TypeDouble,
		Checksum: storage.Checksum{
			Algorithm: "sha256",
			Value:     "998877",
		},
		Locators: []storage.StorageLocator{
			{
				Backend:       "backup2",
				Key:           "archive/embedded.7z",
				ReplicaHealth: storage.ReplicaHealth{Healthy: false, CheckedAt: now},
			},
		},
	}
	if err := store.Create(ctx, meta); err != nil {
		t.Fatalf("create err: %v", err)
	}

	got, err := store.Get(ctx, meta.ID)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if got.Checksum != meta.Checksum || len(got.Locators) != 1 || got.Locators[0].Backend != "backup2" {
		t.Fatalf("embedded get mismatch: %+v", got)
	}

	meta.Name = "embedded-updated"
	if err := store.Update(ctx, meta); err != nil {
		t.Fatalf("update err: %v", err)
	}

	items, err := store.List(ctx, IndexFilter{Name: meta.Name, After: now.Add(-time.Minute)})
	if err != nil {
		t.Fatalf("list err: %v", err)
	}
	if len(items) != 1 || items[0].ID != meta.ID {
		t.Fatalf("list mismatch: %+v", items)
	}

	if err := store.Delete(ctx, meta.ID); err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if _, err := store.Get(ctx, meta.ID); err != ErrNotFound {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}
