// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cocomhub/cocom/pkg/storage"
)

func TestStorageObjectLifecycle(t *testing.T) {
	st, logPath := newFakeStorage(t, "baidupcs-test")
	ctx := context.Background()

	meta, err := st.Put(ctx, "./folder/../folder//a.txt", strings.NewReader("hello world"), storage.WithMD5())
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if meta.Key != "/folder/a.txt" {
		t.Fatalf("unexpected key: %s", meta.Key)
	}
	if meta.Size != 11 {
		t.Fatalf("unexpected size: %d", meta.Size)
	}
	if meta.ETag == "" {
		t.Fatalf("etag should not be empty")
	}

	logs := readFakeLog(t, logPath)
	if !strings.Contains(logs, "--profile=test upload") {
		t.Fatalf("unexpected command log: %s", logs)
	}
	if !strings.Contains(logs, "/apps/cocom/folder/a.txt") {
		t.Fatalf("remote root not applied: %s", logs)
	}

	got, err := st.Stat(ctx, "folder/a.txt")
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got.Key != "/folder/a.txt" || got.Size != 11 {
		t.Fatalf("unexpected stat meta: %+v", got)
	}

	list, err := st.List(ctx, "folder")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Key != "/folder/a.txt" {
		t.Fatalf("unexpected list: %+v", list)
	}

	rc, getMeta, err := st.Get(ctx, "folder/a.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	tmpReader, ok := rc.(*tempReadCloser)
	if !ok {
		t.Fatalf("unexpected reader type: %T", rc)
	}
	tmpPath := tmpReader.path
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read get body: %v", err)
	}
	if string(body) != "hello world" {
		t.Fatalf("unexpected get body: %s", body)
	}
	if getMeta.Key != "/folder/a.txt" {
		t.Fatalf("unexpected get meta: %+v", getMeta)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	if _, err := os.Stat(tmpPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary download file not removed: %v", err)
	}

	exists, err := st.Exists(ctx, "folder/a.txt")
	if err != nil || !exists {
		t.Fatalf("exists before delete: exists=%v err=%v", exists, err)
	}
	if err := st.Delete(ctx, "folder/a.txt"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	exists, err = st.Exists(ctx, "folder/a.txt")
	if err != nil {
		t.Fatalf("exists after delete: %v", err)
	}
	if exists {
		t.Fatalf("file should be deleted")
	}
}

func TestStorageCopyMoveAndMappings(t *testing.T) {
	st, _ := newFakeStorage(t, "baidupcs-copy")
	ctx := context.Background()

	if _, err := st.Put(ctx, "src/data.bin", strings.NewReader("copy-data")); err != nil {
		t.Fatalf("put src: %v", err)
	}
	if _, err := st.Copy(ctx, "src/data.bin", "dst/data-copy.bin"); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if _, err := st.Move(ctx, "dst/data-copy.bin", "final/data.bin"); err != nil {
		t.Fatalf("move: %v", err)
	}

	exists, err := st.Exists(ctx, "dst/data-copy.bin")
	if err != nil {
		t.Fatalf("exists old path: %v", err)
	}
	if exists {
		t.Fatalf("old path should be gone after move")
	}
	exists, err = st.Exists(ctx, "final/data.bin")
	if err != nil || !exists {
		t.Fatalf("exists new path: exists=%v err=%v", exists, err)
	}

	if _, err := st.Put(ctx, "final/data.bin", strings.NewReader("dup")); !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("put duplicate should return already exists: %v", err)
	}
	if _, err := st.Stat(ctx, "missing.bin"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("stat missing should return not found: %v", err)
	}
	if _, err := st.Stat(ctx, "../escape"); !errors.Is(err, storage.ErrInvalidParam) {
		t.Fatalf("stat traversal should return invalid param: %v", err)
	}
}

func TestStorageTimeoutAndJSONParsing(t *testing.T) {
	st, _ := newFakeStorage(t, "baidupcs-timeout")
	t.Setenv("FAKE_BAIDUPCS_SLEEP", "1")
	st.config.Timeout = 20 * time.Millisecond
	st.runner = commandRunner{
		command: st.config.Command,
		workDir: st.config.WorkDir,
		timeout: st.config.Timeout,
		args:    append([]string(nil), st.config.Args...),
	}

	_, err := st.Stat(context.Background(), "slow.txt")
	if !errors.Is(err, storage.ErrTransient) {
		t.Fatalf("timeout should map to transient: %v", err)
	}

	entries, err := parseEntries(`[{"path":"/apps/cocom/parsed.txt","size":7,"mod_time":"2026-04-12T10:00:00Z","etag":"abc"}]`)
	if err != nil {
		t.Fatalf("parse json entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != "/apps/cocom/parsed.txt" || entries[0].ETag != "abc" {
		t.Fatalf("unexpected parsed entries: %+v", entries)
	}
}
