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
	"os/exec"
	"path/filepath"
	"runtime"
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

func CopyDir(source, dest string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux", "darwin": // Linux/macOS
		cmd = exec.Command("cp", "-a", source, dest)
	case "windows":
		// Windows 使用 robocopy（功能更全）
		cmd = exec.Command("robocopy", source, dest, "/E", "/COPYALL", "/DCOPY:DAT")
	default:
		// 回退到自己实现
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("cmd[%s] err:%w", cmd.String(), err)
	}
	return nil
}

func Move(source, dest string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux", "darwin": // Linux/macOS
		cmd = exec.Command("mv", source, dest)
	case "windows":
		// Windows 使用 move（功能更全）
		cmd = exec.Command("move", source, dest)
	default:
		return os.Rename(source, dest)
	}

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("cmd[%s] err:%w", cmd.String(), err)
	}
	return nil
}

func Chtimes(name string, mtime time.Time) error {
	return os.Chtimes(name, mtime, mtime)
}
