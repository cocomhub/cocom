// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestHelp(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("help execute failed: %v", err)
	}
}

func TestPackAndUnpack_LocalFS_Min(t *testing.T) {
	_, err := os.Stat("C:\\Program Files\\7-Zip\\7z.exe")
	if err == nil {
		viper.Set("cocom.archive.cmd", "C:\\Program Files\\7-Zip\\7z.exe")
	} else {
		_, err2 := os.Stat("7z.exe")
		if err2 != nil {
			t.Skip("7z not found, skip integration test")
		}
		viper.Set("cocom.archive.cmd", "7z.exe")
	}
	viper.Set("cocom.archive.password", "test-password")

	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	destDir := filepath.Join(tmp, "archives")
	restoreDir := filepath.Join(tmp, "restore")
	indexRoot := filepath.Join(tmp, "data")
	_ = os.MkdirAll(src, 0o755)
	_ = os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644)
	_ = os.WriteFile(filepath.Join(src, "b.txt"), []byte("world"), 0o644)
	_ = os.MkdirAll(destDir, 0o755)
	dest := filepath.Join(destDir, "a.7z")

	{
		root := newRootCmd()
		root.SetArgs([]string{"--index-root", indexRoot, "pack", "--src", src, "--dest", dest, "--id", "1001"})
		if err := root.Execute(); err != nil {
			t.Fatalf("pack failed: %v", err)
		}
		if _, err := os.Stat(dest); err != nil {
			t.Fatalf("archive not found: %v", err)
		}
	}

	{
		root := newRootCmd()
		root.SetArgs([]string{"--index-root", indexRoot, "unpack", "--src", dest, "--out", restoreDir})
		if err := root.Execute(); err != nil {
			t.Fatalf("unpack failed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(restoreDir, "a.txt")); err != nil {
			t.Fatalf("restore missing a.txt: %v", err)
		}
		if _, err := os.Stat(filepath.Join(restoreDir, "b.txt")); err != nil {
			t.Fatalf("restore missing b.txt: %v", err)
		}
	}

	{
		root := newRootCmd()
		root.SetArgs([]string{"--index-root", indexRoot, "query", "--id", "1001"})
		if err := root.Execute(); err != nil {
			t.Fatalf("query failed: %v", err)
		}
	}

	backupRoot := filepath.Join(tmp, "backup")
	{
		root := newRootCmd()
		root.SetArgs([]string{"--index-root", indexRoot, "backup", "--id", "1001", "--to-root", backupRoot, "--backend", "backupfs", "--prefix", "archives/data"})
		if err := root.Execute(); err != nil {
			t.Fatalf("backup failed: %v", err)
		}
	}

	{
		root := newRootCmd()
		root.SetArgs([]string{"--index-root", indexRoot, "check", "--id", "1001"})
		if err := root.Execute(); err != nil {
			t.Fatalf("check failed: %v", err)
		}
	}
}
