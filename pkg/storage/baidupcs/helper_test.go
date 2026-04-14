// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	bdlib "github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	pcserror "github.com/qjfoidnh/BaiduPCS-Go/baidupcs/pcserror"
)

type fakeAdapter struct {
	mu sync.Mutex

	files map[string]*fakeFile // key: remote path
	calls []string

	metaHook     func(path string) (*bdlib.FileDirectory, error)
	listHook     func(path string) (bdlib.FileDirectoryList, error)
	deleteHook   func(paths ...string) error
	copyHook     func(entries ...*bdlib.CpMvJSON) error
	moveHook     func(entries ...*bdlib.CpMvJSON) error
	uploadHook   func(ctx context.Context, localPath, targetPath string, overwrite bool) error
	downloadHook func(ctx context.Context, remotePath, localPath string) error
}

type fakeFile struct {
	data   []byte
	mtime  time.Time
	md5sum string
	isDir  bool
}

func newFakeAdapter() *fakeAdapter {
	return &fakeAdapter{
		files: make(map[string]*fakeFile),
	}
}

func (a *fakeAdapter) recordCall(op string, args ...string) {
	a.calls = append(a.calls, strings.TrimSpace(op+" "+strings.Join(args, " ")))
}

func (a *fakeAdapter) Calls() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.calls...)
}

func (a *fakeAdapter) SeedFile(path string, data string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	sum := md5.Sum([]byte(data))
	a.files[path] = &fakeFile{
		data:   []byte(data),
		mtime:  time.Now().UTC(),
		md5sum: hex.EncodeToString(sum[:]),
	}
}

func (a *fakeAdapter) Meta(path string) (*bdlib.FileDirectory, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.recordCall("meta", path)
	if a.metaHook != nil {
		return a.metaHook(path)
	}

	if f, ok := a.files[path]; ok && !f.isDir {
		return &bdlib.FileDirectory{
			Path:  path,
			Size:  int64(len(f.data)),
			Mtime: f.mtime.Unix(),
			MD5:   f.md5sum,
			Isdir: false,
		}, nil
	}

	// Directory exists if any file is under it.
	if a.dirExistsLocked(path) {
		return &bdlib.FileDirectory{
			Path:  path,
			Size:  0,
			Mtime: time.Now().Unix(),
			Isdir: true,
		}, nil
	}
	return nil, remoteNotFoundErr("meta", 31066, "文件或目录不存在")
}

func (a *fakeAdapter) List(path string) (bdlib.FileDirectoryList, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.recordCall("ls", path)
	if a.listHook != nil {
		return a.listHook(path)
	}

	if !a.dirExistsLocked(path) {
		return nil, remoteNotFoundErr("ls", 31066, "文件或目录不存在")
	}

	// Return immediate children under path.
	prefix := path
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	seenDirs := make(map[string]struct{})
	seenFiles := make(map[string]struct{})
	out := make(bdlib.FileDirectoryList, 0)

	for remote, f := range a.files {
		if f.isDir {
			continue
		}
		if !strings.HasPrefix(remote, prefix) {
			continue
		}
		rest := strings.TrimPrefix(remote, prefix)
		seg, _, hasMore := strings.Cut(rest, "/")
		if seg == "" {
			continue
		}
		childPath := prefix + seg
		if hasMore {
			if _, ok := seenDirs[childPath]; ok {
				continue
			}
			seenDirs[childPath] = struct{}{}
			out = append(out, &bdlib.FileDirectory{
				Path:  childPath,
				Size:  0,
				Mtime: time.Now().Unix(),
				Isdir: true,
			})
			continue
		}
		if _, ok := seenFiles[childPath]; ok {
			continue
		}
		seenFiles[childPath] = struct{}{}
		out = append(out, &bdlib.FileDirectory{
			Path:  remote,
			Size:  int64(len(f.data)),
			Mtime: f.mtime.Unix(),
			MD5:   f.md5sum,
			Isdir: false,
		})
	}
	return out, nil
}

func (a *fakeAdapter) Delete(paths ...string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.recordCall("rm", strings.Join(paths, ","))
	if a.deleteHook != nil {
		return a.deleteHook(paths...)
	}
	for _, p := range paths {
		if _, ok := a.files[p]; !ok {
			return remoteNotFoundErr("rm", 31066, "文件或目录不存在")
		}
		delete(a.files, p)
	}
	return nil
}

func (a *fakeAdapter) Copy(entries ...*bdlib.CpMvJSON) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.copyHook != nil {
		return a.copyHook(entries...)
	}
	for _, e := range entries {
		if e == nil {
			continue
		}
		a.recordCall("cp", e.From, e.To)
		src, ok := a.files[e.From]
		if !ok || src.isDir {
			return remoteNotFoundErr("cp", 31066, "文件或目录不存在")
		}
		if _, ok := a.files[e.To]; ok {
			return remoteNotFoundErr("cp", 31061, "文件已存在")
		}
		a.files[e.To] = &fakeFile{
			data:   append([]byte(nil), src.data...),
			mtime:  time.Now(),
			md5sum: src.md5sum,
			isDir:  false,
		}
	}
	return nil
}

func (a *fakeAdapter) Move(entries ...*bdlib.CpMvJSON) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.moveHook != nil {
		return a.moveHook(entries...)
	}
	for _, e := range entries {
		if e == nil {
			continue
		}
		a.recordCall("mv", e.From, e.To)
		src, ok := a.files[e.From]
		if !ok || src.isDir {
			return remoteNotFoundErr("mv", 31066, "文件或目录不存在")
		}
		if _, ok := a.files[e.To]; ok {
			return remoteNotFoundErr("mv", 31061, "文件已存在")
		}
		a.files[e.To] = &fakeFile{
			data:   append([]byte(nil), src.data...),
			mtime:  time.Now(),
			md5sum: src.md5sum,
			isDir:  false,
		}
		delete(a.files, e.From)
	}
	return nil
}

func (a *fakeAdapter) Upload(ctx context.Context, localPath, targetPath string, overwrite bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.recordCall("upload", localPath, targetPath)
	if a.uploadHook != nil {
		return a.uploadHook(ctx, localPath, targetPath, overwrite)
	}

	if _, ok := a.files[targetPath]; ok && !overwrite {
		return remoteNotFoundErr("upload", 31061, "文件已存在")
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	sum := md5.Sum(data)
	a.files[targetPath] = &fakeFile{
		data:   data,
		mtime:  time.Now(),
		md5sum: hex.EncodeToString(sum[:]),
		isDir:  false,
	}
	return nil
}

func (a *fakeAdapter) Download(ctx context.Context, remotePath, localPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	if a.downloadHook != nil {
		a.recordCall("download", remotePath, localPath)
		hook := a.downloadHook
		a.mu.Unlock()
		return hook(ctx, remotePath, localPath)
	}
	f, ok := a.files[remotePath]
	a.recordCall("download", remotePath, localPath)
	a.mu.Unlock()

	if !ok || f.isDir {
		return remoteNotFoundErr("download", 31066, "文件或目录不存在")
	}
	return os.WriteFile(localPath, f.data, 0o644)
}

func (a *fakeAdapter) dirExistsLocked(dir string) bool {
	if dir == "" {
		return false
	}
	if dir == "/" {
		return true
	}
	prefix := dir
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for remote, f := range a.files {
		if f.isDir {
			continue
		}
		if strings.HasPrefix(remote, prefix) {
			return true
		}
	}
	return false
}

func remoteNotFoundErr(op string, code int, msg string) error {
	info := pcserror.NewPCSErrorInfo(op)
	info.ErrCode = code
	info.ErrMsg = msg
	info.SetRemoteError()
	return info
}

func newFakeStorage(t *testing.T, name string) (*Storage, *fakeAdapter) {
	t.Helper()

	baseDir := t.TempDir()
	adapter := newFakeAdapter()
	st, err := New(name, Config{
		Root:    "/apps/cocom",
		TempDir: filepath.Join(baseDir, "tmp"),
		Adapter: adapter,
	})
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	return st, adapter
}
