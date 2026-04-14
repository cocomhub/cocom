// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	bdlib "github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	pcserror "github.com/qjfoidnh/BaiduPCS-Go/baidupcs/pcserror"

	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/baidupcs"
)

func TestIndexStoreFS_BaiduPCS(t *testing.T) {
	st := newFakeBaiduPCSStorage(t, "archive-index-baidupcs")
	store := NewIndexStoreFS(st, "index")
	ctx := context.Background()

	meta := ArchiveMeta{ID: 5001, Name: "remote-index", Path: "/tmp/remote-index.7z", Size: 12, ModTime: time.Now(), Version: 1, Type: archive.TypeSingle}
	if err := store.Create(ctx, meta); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.Get(ctx, 5001)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != meta.Name {
		t.Fatalf("unexpected meta name: %s", got.Name)
	}
	list, err := store.List(ctx, IndexFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != 5001 {
		t.Fatalf("unexpected list: %+v", list)
	}
	if err := store.Delete(ctx, 5001); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err = store.List(ctx, IndexFilter{})
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("unexpected list after delete: %+v", list)
	}
}

func TestReplicateToStorage_BaiduPCS(t *testing.T) {
	srcDir := t.TempDir()
	p := filepath.Join(srcDir, "replicated.7z")
	if err := os.WriteFile(p, []byte("replicate"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	mgr := New()
	idx := mgr.(*manager).index
	ctx := context.Background()
	meta := ArchiveMeta{ID: 5002, Name: "replicated", Path: p, Version: 1, Type: archive.TypeSingle}
	if err := idx.Create(ctx, meta); err != nil {
		t.Fatalf("create: %v", err)
	}

	dst := newFakeBaiduPCSStorage(t, "archive-replica-baidupcs")
	n, err := newHelper(mgr).ReplicateToStorage(ctx, dst, "rep", IndexFilter{ID: 5002})
	if err != nil || n != 1 {
		t.Fatalf("replicate: %v n=%d", err, n)
	}

	key := storage.MustPath("rep", filepath.Base(p))
	exists, err := dst.Exists(ctx, key)
	if err != nil || !exists {
		t.Fatalf("exists: %v", err)
	}

	m2, err := idx.Get(ctx, 5002)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	found := false
	for _, loc := range m2.Locators {
		if loc.Backend == dst.Name() && loc.Key == key {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("locator not updated for baidupcs backend")
	}
}

type fakeBaiduPCSAdapter struct {
	mu    sync.Mutex
	files map[string]*fakeBaiduPCSFile
	dirs  map[string]struct{}
}

type fakeBaiduPCSFile struct {
	data   []byte
	mtime  int64
	md5sum string
}

func newFakeBaiduPCSStorage(t *testing.T, name string) storage.Storage {
	t.Helper()

	baseDir := t.TempDir()
	adapter := &fakeBaiduPCSAdapter{
		files: make(map[string]*fakeBaiduPCSFile),
		dirs: map[string]struct{}{
			"/":        {},
			"/archive": {},
		},
	}
	st, err := baidupcs.New(name, baidupcs.Config{
		Root:    "/archive",
		TempDir: filepath.Join(baseDir, "tmp"),
		Adapter: adapter,
	})
	if err != nil {
		t.Fatalf("new baidupcs storage: %v", err)
	}
	return st
}

func (a *fakeBaiduPCSAdapter) Meta(path string) (*bdlib.FileDirectory, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if f, ok := a.files[path]; ok {
		return &bdlib.FileDirectory{
			Path:  path,
			Size:  int64(len(f.data)),
			Mtime: f.mtime,
			MD5:   f.md5sum,
		}, nil
	}
	if a.dirExistsLocked(path) {
		return &bdlib.FileDirectory{
			Path:  path,
			Mtime: 1,
			Isdir: true,
		}, nil
	}
	return nil, fakeRemoteErr("meta", 31066, "文件或目录不存在")
}

func (a *fakeBaiduPCSAdapter) List(path string) (bdlib.FileDirectoryList, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.dirExistsLocked(path) {
		return nil, fakeRemoteErr("ls", 31066, "文件或目录不存在")
	}
	prefix := path
	if prefix != "/" {
		prefix += "/"
	}
	dirs := make(map[string]struct{})
	files := make(map[string]struct{})
	out := make(bdlib.FileDirectoryList, 0)
	for remote, f := range a.files {
		if !strings.HasPrefix(remote, prefix) {
			continue
		}
		rest := strings.TrimPrefix(remote, prefix)
		seg, _, hasMore := strings.Cut(rest, "/")
		if seg == "" {
			continue
		}
		child := prefix + seg
		if hasMore {
			if _, ok := dirs[child]; ok {
				continue
			}
			dirs[child] = struct{}{}
			out = append(out, &bdlib.FileDirectory{Path: child, Mtime: 1, Isdir: true})
			continue
		}
		if _, ok := files[child]; ok {
			continue
		}
		files[child] = struct{}{}
		out = append(out, &bdlib.FileDirectory{
			Path:  remote,
			Size:  int64(len(f.data)),
			Mtime: f.mtime,
			MD5:   f.md5sum,
		})
	}
	return out, nil
}

func (a *fakeBaiduPCSAdapter) Delete(paths ...string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, p := range paths {
		delete(a.files, p)
	}
	return nil
}

func (a *fakeBaiduPCSAdapter) Copy(entries ...*bdlib.CpMvJSON) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, e := range entries {
		if e == nil {
			continue
		}
		src, ok := a.files[e.From]
		if !ok {
			return fakeRemoteErr("cp", 31066, "文件或目录不存在")
		}
		a.ensureDirLocked(path.Dir(e.To))
		a.files[e.To] = &fakeBaiduPCSFile{
			data:   append([]byte(nil), src.data...),
			mtime:  src.mtime,
			md5sum: src.md5sum,
		}
	}
	return nil
}

func (a *fakeBaiduPCSAdapter) Move(entries ...*bdlib.CpMvJSON) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, e := range entries {
		if e == nil {
			continue
		}
		src, ok := a.files[e.From]
		if !ok {
			return fakeRemoteErr("mv", 31066, "文件或目录不存在")
		}
		a.ensureDirLocked(path.Dir(e.To))
		a.files[e.To] = &fakeBaiduPCSFile{
			data:   append([]byte(nil), src.data...),
			mtime:  src.mtime,
			md5sum: src.md5sum,
		}
		delete(a.files, e.From)
	}
	return nil
}

func (a *fakeBaiduPCSAdapter) Upload(ctx context.Context, localPath, targetPath string, overwrite bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	sum := md5.Sum(data)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensureDirLocked(path.Dir(targetPath))
	a.files[targetPath] = &fakeBaiduPCSFile{
		data:   data,
		mtime:  1,
		md5sum: hex.EncodeToString(sum[:]),
	}
	return nil
}

func (a *fakeBaiduPCSAdapter) Download(ctx context.Context, remotePath, localPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	f, ok := a.files[remotePath]
	a.mu.Unlock()
	if !ok {
		return fakeRemoteErr("download", 31066, "文件或目录不存在")
	}
	return os.WriteFile(localPath, f.data, 0o644)
}

func (a *fakeBaiduPCSAdapter) dirExistsLocked(path string) bool {
	_, ok := a.dirs[path]
	return ok
}

func (a *fakeBaiduPCSAdapter) ensureDirLocked(dir string) {
	for dir != "" && dir != "." {
		if _, ok := a.dirs[dir]; ok {
			return
		}
		a.dirs[dir] = struct{}{}
		if dir == "/" {
			return
		}
		dir = path.Dir(dir)
	}
}

func fakeRemoteErr(op string, code int, msg string) error {
	info := pcserror.NewPCSErrorInfo(op)
	info.ErrCode = code
	info.ErrMsg = msg
	info.SetRemoteError()
	return info
}
