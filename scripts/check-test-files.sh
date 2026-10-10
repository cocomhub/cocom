#!/usr/bin/env bash
# Copyright 2026 The Cocomhub Authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#
# check-test-files.sh — 验证所有 Go 包都有测试文件
# 使用方法： scripts/check-test-files.sh <packages...>
# .notestignore 中列出的包免检（每行一个包路径或 glob 模式，支持 # 注释与空行）

set -euo pipefail

IGNORE_FILE=".notestignore"

IGNORE_PATTERNS=()
if [ -f "$IGNORE_FILE" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%%#*}"
    line="$(echo "$line" | xargs)"
    [ -z "$line" ] && continue
    IGNORE_PATTERNS+=("$line")
  done < "$IGNORE_FILE"
fi

# is_ignored <pkg> —— pkg 形如 ./cmd/server/internal/testutil。
# 命中 .notestignore 中任一条目（支持 glob）返回 0。
is_ignored() {
  local pkg="$1"
  local rel="${pkg#./}"
  local pat
  for pat in "${IGNORE_PATTERNS[@]}"; do
    # shellcheck disable=SC2053  # 右侧故意不加引号以启用 glob 匹配
    if [[ "$rel" == $pat || "./$rel" == $pat ]]; then
      return 0
    fi
  done
  return 1
}

exit_code=0
missing_count=0
missing_list=""

for pkg in "$@"; do
  pkg="${pkg%/}"
  [ -z "$pkg" ] || [ "$pkg" = "." ] || [ ! -d "$pkg" ] && continue

  test_files=$(find "$pkg" -maxdepth 1 -name '*_test.go' -print -quit 2>/dev/null)
  if [ -n "$test_files" ]; then
    continue
  fi

  # 无测试文件：命中免检清单才放过，否则判 FAIL。
  if is_ignored "$pkg"; then
    continue
  fi

  echo "FAIL: $pkg has no test files" >&2
  exit_code=1
  missing_count=$((missing_count + 1))
  missing_list="$missing_list $pkg"
done

if [ $exit_code -eq 0 ]; then
  echo "OK: all packages have test files"
else
  echo "FAIL: $missing_count package(s) missing test files:$missing_list" >&2
fi

exit $exit_code
