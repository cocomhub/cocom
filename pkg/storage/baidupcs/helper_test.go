// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newFakeStorage(t *testing.T, name string) (*Storage, string) {
	t.Helper()

	baseDir := t.TempDir()
	logPath := filepath.Join(baseDir, "fake-baidupcs.log")
	commandPath := filepath.Join(baseDir, "fake-baidupcs")
	if err := os.WriteFile(commandPath, []byte(fakeCommandScript()), 0o755); err != nil {
		t.Fatalf("write fake command: %v", err)
	}
	t.Setenv("FAKE_BAIDUPCS_ROOT", filepath.Join(baseDir, "remote"))
	t.Setenv("FAKE_BAIDUPCS_LOG", logPath)
	st, err := New(name, Config{
		Command: commandPath,
		Root:    "/apps/cocom",
		TempDir: filepath.Join(baseDir, "tmp"),
		Timeout: time.Second,
		Args:    []string{"--profile=test"},
	})
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	return st, logPath
}

func readFakeLog(t *testing.T, logPath string) string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake log: %v", err)
	}
	return string(data)
}

func fakeCommandScript() string {
	return strings.TrimSpace(`
#!/bin/sh
set -eu

log_file="${FAKE_BAIDUPCS_LOG:-}"
if [ -n "$log_file" ]; then
  printf '%s\n' "$*" >> "$log_file"
fi

if [ -n "${FAKE_BAIDUPCS_SLEEP:-}" ]; then
  sleep "$FAKE_BAIDUPCS_SLEEP"
fi

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
if [ -z "$cmd" ]; then
  echo "missing command" >&2
  exit 2
fi
shift

map_remote() {
  remote="$1"
  remote="${remote#/}"
  printf '%s/%s\n' "$root" "$remote"
}

mtime() {
  stat -f %m "$1"
}

filesize() {
  wc -c < "$1" | tr -d '[:space:]'
}

emit_file() {
  remote="$1"
  path="$(map_remote "$remote")"
  printf 'F\t%s\t%s\t%s\n' "$remote" "$(filesize "$path")" "$(mtime "$path")"
}

emit_dir() {
  remote="$1"
  path="$(map_remote "$remote")"
  printf 'D\t%s\t0\t%s\n' "$remote" "$(mtime "$path")"
}

copy_file() {
  src="$1"
  dst="$2"
  mkdir -p "$(dirname "$dst")"
  cp "$src" "$dst"
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
    copy_file "$local_path" "$target"
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
      emit_dir "$remote"
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
      printf 'F\t%s\t%s\t%s\n' "$rel" "$(filesize "$file")" "$(mtime "$file")"
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
    copy_file "$source" "$target"
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
`)
}
