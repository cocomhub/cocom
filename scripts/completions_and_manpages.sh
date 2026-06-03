#!/bin/sh
# Copyright 2026 The Cocomhub Authors. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

# 检查是否提供了二进制文件路径
if [ $# -ne 1 ]; then
    echo "用法: $0 <二进制文件路径>"
    echo "示例: $0 /usr/local/bin/cobra"
    exit 1
fi

BINARY_PATH="$1"
BINARY_NAME=$(basename "$BINARY_PATH")

# 检查二进制文件是否存在
if [ ! -f "$BINARY_PATH" ]; then
    echo "错误: 二进制文件 '$BINARY_PATH' 不存在"
    exit 1
fi

# 尝试执行 --help 来验证二进制是否可用
if ! "$BINARY_PATH" --help >/dev/null 2>&1; then
    echo "错误: 二进制文件 '$BINARY_PATH' 无法执行（可能是架构不匹配或文件损坏）" >&2
    exit 1
fi

# 创建基于二进制名称的输出目录
COMPLETIONS_DIR="completions"
MANPAGES_DIR="manpages"

# 创建目录
mkdir -p "$COMPLETIONS_DIR" "$MANPAGES_DIR"

echo "为 $BINARY_NAME 生成文档..."

# 为不同 shell 生成补全脚本
for sh in bash zsh fish; do
    echo "  生成 $sh 补全..."
    "$BINARY_PATH" completion "$sh" > "$COMPLETIONS_DIR/$BINARY_NAME.$sh"
done

# 生成 man 文档
echo "  生成 man 文档..."
"$BINARY_PATH" man | gzip -c -9 > "$MANPAGES_DIR/$BINARY_NAME.1.gz"

echo "完成！文档已生成"
echo "  补全脚本: $COMPLETIONS_DIR/"
echo "  man 文档: $MANPAGES_DIR/"