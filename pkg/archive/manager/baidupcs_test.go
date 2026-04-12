// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func newFakeBaiduPCSStorage(t *testing.T, name string) storage.Storage {
	t.Helper()

	baseDir := t.TempDir()
	commandPath := filepath.Join(baseDir, "fake-baidupcs")
	if err := os.WriteFile(commandPath, []byte(fakeBaiduPCSScript()), 0o755); err != nil {
		t.Fatalf("write fake command: %v", err)
	}
	t.Setenv("FAKE_BAIDUPCS_ROOT", filepath.Join(baseDir, "remote"))
	t.Setenv("FAKE_BAIDUPCS_LOG", filepath.Join(baseDir, "fake-baidupcs.log"))
	st, err := baidupcs.New(name, baidupcs.Config{
		Command: commandPath,
		Root:    "/archive",
		TempDir: filepath.Join(baseDir, "tmp"),
		Timeout: time.Second,
		Args:    []string{"--profile=test"},
	})
	if err != nil {
		t.Fatalf("new baidupcs storage: %v", err)
	}
	return st
}

func fakeBaiduPCSScript() string {
	return strings.TrimSpace(fmt.Sprintf(`
#!/bin/sh
set -eu

root="${FAKE_BAIDUPCS_ROOT:?}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --*)
      shift
      ;;
    *)
      break
      ;;
  esac
done

cmd="${1:-}"
shift

map_remote() {
  remote="$1"
  remote="${remote#/}"
  printf '%%s/%%s\n' "$root" "$remote"
}

mtime() {
  stat -f %s "$1"
}

filesize() {
  wc -c < "$1" | tr -d '[:space:]'
}

emit_file() {
  remote="$1"
  path="$(map_remote "$remote")"
  printf 'F\t%%s\t%%s\t%%s\n' "$remote" "$(filesize "$path")" "$(mtime "$path")"
}

case "$cmd" in
  upload)
    overwrite=0
    if [ "${1:-}" = "--overwrite" ]; then
      overwrite=1
      shift
    fi
    local_path="$1"
    remote="$2"
    target="$(map_remote "$remote")"
    mkdir -p "$(dirname "$target")"
    if [ -e "$target" ] && [ "$overwrite" -ne 1 ]; then
      echo "file already exists: $remote" >&2
      exit 11
    fi
    cp "$local_path" "$target"
    ;;
  download)
    remote="$1"
    local_path="$2"
    source="$(map_remote "$remote")"
    if [ ! -e "$source" ]; then
      echo "文件不存在: $remote" >&2
      exit 4
    fi
    mkdir -p "$(dirname "$local_path")"
    cp "$source" "$local_path"
    ;;
  meta)
    remote="$1"
    target="$(map_remote "$remote")"
    if [ ! -e "$target" ]; then
      echo "文件不存在: $remote" >&2
      exit 4
    fi
    if [ -d "$target" ]; then
      printf 'D\t%%s\t0\t%%s\n' "$remote" "$(mtime "$target")"
    else
      emit_file "$remote"
    fi
    ;;
  ls)
    remote="$1"
    target="$(map_remote "$remote")"
    if [ ! -e "$target" ]; then
      echo "文件不存在: $remote" >&2
      exit 4
    fi
    if [ -f "$target" ]; then
      emit_file "$remote"
      exit 0
    fi
    find "$target" -type f | sort | while IFS= read -r file; do
      rel="${file#$root}"
      printf 'F\t%%s\t%%s\t%%s\n' "$rel" "$(filesize "$file")" "$(mtime "$file")"
    done
    ;;
  rm)
    remote="$1"
    target="$(map_remote "$remote")"
    if [ ! -e "$target" ]; then
      echo "文件不存在: $remote" >&2
      exit 4
    fi
    rm -f "$target"
    ;;
  cp)
    overwrite=0
    if [ "${1:-}" = "--overwrite" ]; then
      overwrite=1
      shift
    fi
    remote_src="$1"
    remote_dst="$2"
    source="$(map_remote "$remote_src")"
    target="$(map_remote "$remote_dst")"
    if [ ! -e "$source" ]; then
      echo "文件不存在: $remote_src" >&2
      exit 4
    fi
    if [ -e "$target" ] && [ "$overwrite" -ne 1 ]; then
      echo "file already exists: $remote_dst" >&2
      exit 11
    fi
    mkdir -p "$(dirname "$target")"
    cp "$source" "$target"
    ;;
  mv)
    overwrite=0
    if [ "${1:-}" = "--overwrite" ]; then
      overwrite=1
      shift
    fi
    remote_src="$1"
    remote_dst="$2"
    source="$(map_remote "$remote_src")"
    target="$(map_remote "$remote_dst")"
    if [ ! -e "$source" ]; then
      echo "文件不存在: $remote_src" >&2
      exit 4
    fi
    if [ -e "$target" ] && [ "$overwrite" -ne 1 ]; then
      echo "file already exists: $remote_dst" >&2
      exit 11
    fi
    mkdir -p "$(dirname "$target")"
    if [ -e "$target" ] && [ "$overwrite" -eq 1 ]; then
      rm -f "$target"
    fi
    mv "$source" "$target"
    ;;
  *)
    echo "unknown command: $cmd" >&2
    exit 2
    ;;
esac
`, "%m"))
}
