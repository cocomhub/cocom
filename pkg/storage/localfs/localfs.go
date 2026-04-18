// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cocomhub/cocom/pkg/storage"
)

type FS struct {
	name string
	Root string
}

func New(name, root string) *FS {
	return &FS{name: name, Root: filepath.Clean(root)}
}

func (fs *FS) Type() string {
	return "localfs"
}

func (fs *FS) Name() string {
	if fs.name != "" {
		return fs.name
	}
	return fs.Root
}

func (fs *FS) withRoot(key string, fn func(r *os.Root, key string) error) error {
	r, err := os.OpenRoot(fs.Root)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		err = os.MkdirAll(fs.Root, 0o755)
		if err != nil {
			return err
		}
		r, err = os.OpenRoot(fs.Root)
		if err != nil {
			return err
		}
	}
	defer r.Close()
	// Block attempts that start with traversal even if Clean would collapse them.
	raw := strings.TrimLeft(key, "/\\")
	if strings.HasPrefix(filepath.ToSlash(raw), "..") {
		return fmt.Errorf("key %s is traversal blocked", key)
	}
	key = strings.TrimLeft(filepath.Clean(key), "/\\")
	if strings.HasPrefix(key, "..") {
		return fmt.Errorf("key %s is traversal blocked", key)
	}
	return fn(r, key)
}

func (fs *FS) Put(ctx context.Context, key string, r io.Reader, opts ...storage.Option) (*storage.ObjectMeta, error) {
	var po storage.PutOptions
	for _, o := range opts {
		o(&po)
	}
	var meta storage.ObjectMeta
	err := fs.withRoot(key, func(root *os.Root, key string) error {
		_ = root.MkdirAll(filepath.Dir(key), 0o755)
		if !po.Overwrite {
			if _, err := root.Stat(key); err == nil {
				return storage.ErrAlreadyExists
			} else if err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		tmp := key + fmt.Sprintf(".tmp-%d", time.Now().UnixNano())
		f, err := root.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}

		if po.Hash != nil {
			n, err := io.Copy(f, io.TeeReader(r, po.Hash))
			if err != nil {
				_ = f.Close()
				_ = root.Remove(tmp)
				return err
			}
			meta = storage.ObjectMeta{Key: storage.MustPath(key), Size: n, ETag: hex.EncodeToString(po.Hash.Sum(nil)), ModTime: time.Now()}
		} else {
			n, err := io.Copy(f, r)
			if err != nil {
				_ = f.Close()
				_ = root.Remove(tmp)
				return err
			}
			meta = storage.ObjectMeta{Key: storage.MustPath(key), Size: n, ModTime: time.Now()}
		}
		if err := f.Close(); err != nil {
			_ = root.Remove(tmp)
			return err
		}
		// Best-effort atomic replace: if overwrite allowed, remove existing first on platforms where rename won't overwrite.
		if po.Overwrite {
			_ = root.Remove(key)
		}
		if err := root.Rename(tmp, key); err != nil {
			_ = root.Remove(tmp)
			return err
		}
		return nil
	})
	return &meta, err
}

func (fs *FS) Get(ctx context.Context, key string) (io.ReadCloser, *storage.ObjectMeta, error) {
	var (
		rc   io.ReadCloser
		meta *storage.ObjectMeta
	)
	err := fs.withRoot(key, func(root *os.Root, key string) error {
		f, err := root.Open(key)
		if err != nil {
			return err
		}
		info, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return err
		}
		rc = f
		meta = fs.getMeta(key, info)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, storage.ErrNotFound
		}
		if os.IsPermission(err) {
			return nil, nil, storage.ErrPermissionDenied
		}
		return nil, nil, err
	}
	return rc, meta, nil
}

func (fs *FS) Stat(ctx context.Context, key string) (*storage.ObjectMeta, error) {
	var meta *storage.ObjectMeta
	err := fs.withRoot(key, func(root *os.Root, key string) error {
		info, err := root.Stat(key)
		if err != nil {
			return err
		}
		meta = fs.getMeta(key, info)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.ErrNotFound
		}
		if os.IsPermission(err) {
			return nil, storage.ErrPermissionDenied
		}
		return nil, err
	}
	return meta, nil
}

func (fs *FS) Exists(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := fs.withRoot(key, func(root *os.Root, key string) error {
		_, err := root.Stat(key)
		if err == nil {
			exists = true
			return nil
		}
		if os.IsNotExist(err) {
			exists = false
			return nil
		}
		return err
	})
	return exists, err
}

func (fs *FS) List(ctx context.Context, prefix string) ([]storage.ObjectMeta, error) {
	var out []storage.ObjectMeta
	err := fs.withRoot(prefix, func(root *os.Root, prefix string) error {
		start := prefix
		if start == "." || start == string(os.PathSeparator) {
			start = "."
		}
		info, err := root.Stat(start)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			out = append(out, *fs.getMeta(start, info))
			return nil
		}
		var stack []string
		stack = append(stack, start)
		for len(stack) > 0 {
			dir := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			df, err := root.Open(dir)
			if err != nil {
				return err
			}
			entries, err := df.ReadDir(-1)
			_ = df.Close()
			if err != nil {
				return err
			}
			for _, e := range entries {
				p := filepath.Join(dir, e.Name())
				if e.IsDir() {
					stack = append(stack, p)
					continue
				}
				fi, err := root.Stat(p)
				if err != nil {
					continue
				}
				out = append(out, *fs.getMeta(p, fi))
			}
		}
		return nil
	})
	return out, err
}

func (fs *FS) Delete(ctx context.Context, key string) error {
	return fs.withRoot(key, func(root *os.Root, key string) error {
		return root.Remove(key)
	})
}

func (fs *FS) Copy(ctx context.Context, srcKey, dstKey string, opts ...storage.Option) (*storage.ObjectMeta, error) {
	var meta *storage.ObjectMeta
	err := fs.withRoot(srcKey, func(root *os.Root, srcKey string) error {
		in, err := root.Open(srcKey)
		if err != nil {
			return err
		}
		defer in.Close()
		m, err := fs.Put(ctx, dstKey, in, opts...)
		if err != nil {
			return err
		}
		meta = m
		return nil
	})
	return meta, err
}

func (fs *FS) Move(ctx context.Context, srcKey, dstKey string, opts ...storage.Option) (*storage.ObjectMeta, error) {
	var meta *storage.ObjectMeta
	err := fs.withRoot(srcKey, func(root *os.Root, srcKey string) error {
		dstKey = filepath.Clean(dstKey)
		_ = root.MkdirAll(filepath.Dir(dstKey), 0o755)
		if err := root.Rename(srcKey, dstKey); err != nil {
			return err
		}
		info, err := root.Stat(dstKey)
		if err != nil {
			return err
		}
		meta = fs.getMeta(dstKey, info)
		return nil
	})
	return meta, err
}

func (fs *FS) getMeta(key string, info os.FileInfo) *storage.ObjectMeta {
	return &storage.ObjectMeta{
		Key:     storage.MustPath(key),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
}
