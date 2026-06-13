#!/usr/bin/env bash
# Copyright 2026 The Cocomhub Authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#
# check-test-files.sh — 验证所有 Go 包都有测试文件（用于 make nocover）
#
# 使用方法： scripts/check-test-files.sh <packages...>
#
# 对每个传入的包路径，检查该目录下是否有 *_test.go 文件。
# 如果某些包目前没有测试文件且已知被允许，请在 ALLOWLIST 中列出。
# 随着测试覆盖率的提升，逐步从 ALLOWLIST 中移除条目。

set -euo pipefail

# 已知暂时没有测试文件的包（逐个移除，目标：全部清空）
# 2026-06-13: 全部包已有测试文件，保留空数组以方便未来添加。
ALLOWLIST=()

# 将 ALLOWLIST 转为以换行符分隔的查找表
allowlist_keys=$(printf "%s\n" "${ALLOWLIST[@]}")

is_allowed() {
  local pkg="$1"
  # 去掉尾部斜杠以便匹配
  pkg="${pkg%/}"
  while IFS= read -r line; do
    line="${line%/}"
    if [ "$pkg" = "$line" ]; then
      return 0
    fi
  done <<< "$allowlist_keys"
  return 1
}

has_test_file() {
  local dir="$1"
  # 检查目录下是否有 *_test.go 文件（非递归）
  test_files=$(find "$dir" -maxdepth 1 -name '*_test.go' -print -quit 2>/dev/null)
  test -n "$test_files"
}

exit_code=0
missing_count=0
missing_list=""

for pkg in "$@"; do
  # 去除尾部斜杠
  pkg="${pkg%/}"

  # 跳过空字符串
  if [ -z "$pkg" ]; then
    continue
  fi

  # 跳过根目录包
  if [ "$pkg" = "." ]; then
    continue
  fi

  # 跳过不存在的目录（ALL_PKGS 可能包含不含 .go 文件的目录）
  if [ ! -d "$pkg" ]; then
    continue
  fi

  if has_test_file "$pkg"; then
    # 有测试文件 — good
    :
  elif is_allowed "$pkg"; then
    # 无测试文件但在 allowlist 中 — 发出警告但不报错
    echo "WARN: $pkg has no test files (allowed by allowlist)" >&2
  else
    echo "FAIL: $pkg has no test files" >&2
    exit_code=1
    missing_count=$((missing_count + 1))
    missing_list="$missing_list $pkg"
  fi
done

if [ $exit_code -eq 0 ]; then
  echo "OK: all packages have test files (or are allowlisted)"
else
  echo "FAIL: $missing_count package(s) missing test files:$missing_list" >&2
fi

exit $exit_code
