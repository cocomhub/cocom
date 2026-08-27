#!/usr/bin/env bash
# Copyright 2026 The Cocomhub Authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#
# check-test-files.sh — 验证所有 Go 包都有测试文件
# 使用方法： scripts/check-test-files.sh <packages...>
# .notestignore 中列出的包免检

set -euo pipefail

IGNORE_FILE=".notestignore"

# isIgnored — 判断包路径是否命中 .notestignore 中的 glob 条目。
# 不用 find -path 排除：find -maxdepth 0 + -quit 对存在的路径恒输出空，
# 导致排除永不生效（历史潜伏 bug）。这里直接做 shell case 匹配，最稳。
isIgnored() {
  local pkg="$1"
  local pkg_normal="${pkg#./}"   # 规整为不带 ./ 前缀，与条目文本对齐
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%%#*}"
    line="$(printf '%s' "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    [ -z "$line" ] && continue
    case "$pkg_normal" in
    "$line") return 0 ;;
    esac
  done < "$IGNORE_FILE"
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

  if isIgnored "$pkg"; then
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
