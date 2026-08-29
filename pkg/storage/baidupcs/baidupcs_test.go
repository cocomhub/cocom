// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bdlib "github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	pcserror "github.com/qjfoidnh/BaiduPCS-Go/baidupcs/pcserror"

	"github.com/cocomhub/cocom/pkg/storage"
)

func TestStorageObjectLifecycle(t *testing.T) {
	st, adapter := newFakeStorage(t, "baidupcs-test")
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

	calls := strings.Join(adapter.Calls(), "\n")
	if !strings.Contains(calls, "upload") {
		t.Fatalf("expected upload call, got %s", calls)
	}
	if !strings.Contains(calls, "/apps/cocom/folder/a.txt") {
		t.Fatalf("remote root not applied: %s", calls)
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
	if closeErr := rc.Close(); closeErr != nil {
		t.Fatalf("close reader: %v", closeErr)
	}
	if _, statErr := os.Stat(tmpPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("temporary download file not removed: %v", statErr)
	}

	exists, err := st.Exists(ctx, "folder/a.txt")
	if err != nil || !exists {
		t.Fatalf("exists before delete: exists=%v err=%v", exists, err)
	}
	if delErr := st.Delete(ctx, "folder/a.txt"); delErr != nil {
		t.Fatalf("delete: %v", delErr)
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

func TestStorageErrorMappingAndBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*fakeAdapter)
		run     func(context.Context, *Storage) error
		wantErr error
	}{
		{
			name: "meta deadline maps transient",
			setup: func(a *fakeAdapter) {
				a.metaHook = func(path string) (*bdlib.FileDirectory, error) {
					return nil, context.DeadlineExceeded
				}
			},
			run: func(ctx context.Context, st *Storage) error {
				_, err := st.Stat(ctx, "slow.txt")
				return err
			},
			wantErr: storage.ErrTransient,
		},
		{
			name: "upload already exists maps already exists",
			setup: func(a *fakeAdapter) {
				a.uploadHook = func(ctx context.Context, localPath, targetPath string, overwrite bool) error {
					info := pcserror.NewPCSErrorInfo("upload")
					info.ErrCode = 31061
					info.SetRemoteError()
					return info
				}
			},
			run: func(ctx context.Context, st *Storage) error {
				_, err := st.Put(ctx, "dup.txt", strings.NewReader("dup"))
				return err
			},
			wantErr: storage.ErrAlreadyExists,
		},
		{
			name: "download permission message maps permission denied",
			setup: func(a *fakeAdapter) {
				a.SeedFile("/apps/cocom/noaccess.txt", "secret")
				a.downloadHook = func(ctx context.Context, remotePath, localPath string) error {
					return errors.New("cookie expired, please login again")
				}
			},
			run: func(ctx context.Context, st *Storage) error {
				_, _, err := st.Get(ctx, "noaccess.txt")
				return err
			},
			wantErr: storage.ErrPermissionDenied,
		},
		{
			name: "delete not found maps not found",
			setup: func(a *fakeAdapter) {
				a.deleteHook = func(paths ...string) error {
					info := pcserror.NewPCSErrorInfo("rm")
					info.ErrCode = 31066
					info.SetRemoteError()
					return info
				}
			},
			run: func(ctx context.Context, st *Storage) error {
				return st.Delete(ctx, "missing.txt")
			},
			wantErr: storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, adapter := newFakeStorage(t, "baidupcs-errors")
			tt.setup(adapter)
			err := tt.run(context.Background(), st)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestStorageListAndMetaBoundaries(t *testing.T) {
	st, adapter := newFakeStorage(t, "baidupcs-list")
	adapter.SeedFile("/apps/cocom/folder/a.txt", "a")
	adapter.SeedFile("/apps/cocom/folder/sub/b.txt", "bb")

	list, err := st.List(context.Background(), "folder")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("unexpected list length: %d", len(list))
	}

	adapter.metaHook = func(path string) (*bdlib.FileDirectory, error) {
		return &bdlib.FileDirectory{Path: "/outside/root.txt", Size: 1, Mtime: time.Now().Unix()}, nil
	}
	if _, err := st.Stat(context.Background(), "folder/a.txt"); !errors.Is(err, storage.ErrInvalidParam) {
		t.Fatalf("stat outside root should be invalid param: %v", err)
	}
}

// TestNewLibraryClient_MissingBDUSSInCookies 验证 Cookies 不含 BDUSS 字段时
// 返回可读错误而非对 nil 解引用 panic。
func TestNewLibraryClient_MissingBDUSSInCookies(t *testing.T) {
	_, err := newLibraryClient(Config{BDUSS: "", Cookies: "STOKEN=xyz; PAN=abc;"})
	if err == nil {
		t.Fatal("expected error for cookies without BDUSS")
	}
	if !strings.Contains(err.Error(), "BDUSS") {
		t.Errorf("expected BDUSS-related error, got %v", err)
	}
}

// TestPut_ETagMismatchBoundedRetry 验证 Put 上传后 ETag 复核不匹配时
// 有界重试（最多 3 次）而非无限循环。
func TestPut_ETagMismatchBoundedRetry(t *testing.T) {
	st, adapter := newFakeStorage(t, "baidupcs-put-retry")
	ctx := context.Background()

	// 模拟 Baidu PCS 分片上传 ETag 与本地 MD5 语义不一致：上传后内容被改写。
	adapter.uploadHook = func(ctx context.Context, localPath, targetPath string, overwrite bool) error {
		sum := md5.Sum([]byte("different-content"))
		adapter.files[targetPath] = &fakeFile{
			data:   []byte("different-content"),
			mtime:  time.Now(),
			md5sum: hex.EncodeToString(sum[:]),
			isDir:  false,
		}
		return nil
	}

	_, err := st.Put(ctx, "k.txt", strings.NewReader("hello world"), storage.WithMD5())
	if err == nil {
		t.Fatal("expected ETag mismatch error")
	}
	if !strings.Contains(err.Error(), "ETag") {
		t.Errorf("expected ETag mismatch error, got %v", err)
	}

	uploads := 0
	for _, c := range adapter.Calls() {
		if strings.Contains(c, "upload") {
			uploads++
		}
	}
	if uploads > 3 {
		t.Fatalf("upload calls should be bounded by 3, got %d", uploads)
	}
}

func TestTempReadCloserClose(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "tmp-read.txt")
	if err := os.WriteFile(tmp, []byte("x"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	fd, err := os.Open(tmp)
	if err != nil {
		t.Fatalf("open temp file: %v", err)
	}
	rc := &tempReadCloser{ReadCloser: fd, path: tmp, remove: true}
	if err := rc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := os.Stat(tmp); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary file should be removed: %v", err)
	}
}
