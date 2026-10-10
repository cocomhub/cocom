// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cocomhub/cocom/pkg/errwrap"
)

func CreateDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o777)
	}
	return nil
}

func MustFileMD5(filename string) string {
	s, _ := FileMD5(filename)
	return s
}

func FileMD5(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return MD5(f)
}

func MD5(src io.Reader) (string, error) {
	return Hash("md5", src)
}

func MustFileSha256(filename string) string {
	s, _ := FileSha256(filename)
	return s
}

func FileSha256(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return Sha256(f)
}

func Sha256(src io.Reader) (string, error) {
	return Hash("sha256", src)
}

var (
	hashMap = map[string]func() hash.Hash{
		"md5":    md5.New,
		"sha256": sha256.New,
	}

	hashBufPool = sync.Pool{
		New: func() any {
			return make([]byte, 1024*1024)
		},
	}
)

func Hash(name string, src io.Reader) (string, error) {
	if _, ok := hashMap[name]; !ok {
		return "", errwrap.New(-1, "hash name not support").SetIErrF("name:%s", name)
	}
	dst := hashMap[name]()
	buf := hashBufPool.Get().([]byte) //nolint:staticcheck,errcheck
	defer hashBufPool.Put(buf)        //nolint:staticcheck
	_, err := io.CopyBuffer(dst, src, buf)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", dst.Sum(nil)), nil
}

func IsFileSame(f1 string, f2 string) error {
	m1, err := FileMD5(f1)
	if err != nil {
		return errwrap.New(-1, "calc md5 failed").SetIErrF("file(%s) errmsg: %v", f1, err)
	}

	m2, err := FileMD5(f2)
	if err != nil {
		return errwrap.New(-1, "calc md5 failed").SetIErrF("file(%s) errmsg: %v", f2, err)
	}

	if m1 != m2 {
		return errwrap.New(-1, "md5 not same").SetIErrF("file1(%s md5:%s) file2(%s md5:%s)", f1, m1, f2, m2)
	}
	return nil
}

func IsDirSame(d1 string, d2 string) error {
	fs1, err := os.ReadDir(d1)
	if err != nil {
		return errwrap.New(-1, "read dir failed").SetIErrF("dir(%s) errmsg: %#v", d1, err)
	}
	fs2, err := os.ReadDir(d2)
	if err != nil {
		return errwrap.New(-1, "read dir failed").SetIErrF("dir(%s) errmsg: %#v", d2, err)
	}
	if len(fs1) != len(fs2) {
		return errwrap.New(-1, "file count not equal").SetIErrF("dir1(%s len:%d) dir2(%s len:%d)",
			d1, len(fs1), d2, len(fs2))
	}

	return filepath.Walk(d1, func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		dstPath := strings.Replace(path, d1, d2, 1)
		dstInfo, err := os.Stat(dstPath)
		if err != nil {
			return errwrap.New(-1, "check dst stat failed").SetIErrF("src(%s) dst(%s) errmsg: %#v",
				path, dstPath, err)
		}

		if dstInfo.IsDir() != info.IsDir() || dstInfo.Size() != info.Size() {
			return errwrap.New(-1, "file stat noq equal").SetIErrF("src(%s isDir:%v size:%d) dst(%s isDir:%v size:%d)",
				path, info.IsDir(), info.Size(), dstPath, dstInfo.IsDir(), dstInfo.Size())
		}

		if info.IsDir() {
			return nil
		}

		return IsFileSame(path, dstPath)
	})
}

// CopyDir 递归复制目录（保留普通文件的权限位与符号链接）。
//
// 实现为纯 Go，不再调用平台命令：此前 Windows 分支用 robocopy 但把非零退出码一律当失败，
// 而 robocopy 用 0-7 表示成功（1=复制成功、2=有额外文件…），因此复制成功后反而报错；
// 同时 robocopy/cp 的存在性依赖也让行为无法在目标机器上稳定复现。
func CopyDir(source, dest string) error {
	srcInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat source %q: %w", source, err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory: %s", source)
	}
	if err := os.MkdirAll(dest, srcInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("create dest %q: %w", dest, err)
	}

	return filepath.WalkDir(source, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)

		switch {
		case d.IsDir():
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		case d.Type()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("readlink %q: %w", path, err)
			}
			return os.Symlink(link, target)
		case d.Type().IsRegular():
			return copyFileWithMode(path, target)
		default:
			// 具名管道/设备文件等：不静默跳过，明确报错避免“看似复制成功”。
			return fmt.Errorf("unsupported file type at %q (%s)", path, d.Type())
		}
	})
}

// Move 移动文件或目录。优先 os.Rename（同卷原子操作）；跨卷等失败场景
// 回退为「复制 + 删除源」，与原 mv 语义一致。
//
// 此前 Windows 分支调用 exec.Command("move", ...)，而 move 是 cmd.exe 的内建命令、
// 不是可执行文件，exec 不会经过 shell，必然失败。
func Move(source, dest string) error {
	// 与 POSIX mv 一致：dest 已存在且是目录时，把 source 移入该目录（dest/base(source)）。
	// 这一步不能省：调用方（如 restoreComic）正是把「移到父目录」当作「移到父目录下的同名目录」用。
	// 若直接 os.Rename(source, dest)，目标是已存在非空目录时会 ENOTEMPTY 失败，
	// 随后回退到 CopyDir —— 而 CopyDir 是把 source 的**内容**摊平进 dest，目录层级被丢掉。
	if info, statErr := os.Stat(dest); statErr == nil && info.IsDir() {
		dest = filepath.Join(dest, filepath.Base(source))
	}

	if err := os.Rename(source, dest); err == nil {
		return nil
	}

	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat source %q: %w", source, err)
	}
	if info.IsDir() {
		if err := CopyDir(source, dest); err != nil {
			return fmt.Errorf("move fallback (copy dir): %w", err)
		}
		return os.RemoveAll(source)
	}
	if err := copyFileWithMode(source, dest); err != nil {
		return fmt.Errorf("move fallback (copy file): %w", err)
	}
	return os.Remove(source)
}

func copyFileWithMode(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func Chtimes(name string, mtime time.Time) error {
	return os.Chtimes(name, mtime, mtime)
}
