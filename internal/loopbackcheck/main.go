// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Command loopbackcheck 静态扫描 Go 源码中硬编码的通配监听地址（0.0.0.0 / :: / 省略 host）。
//
// 与裸 grep 不同，它基于 Go AST 只检查「真正作为监听地址使用」的字符串字面量，
// 因此不会把注释、文档样例或"警告不要用 0.0.0.0"的代码误判为违规。
//
// 用法： go run ./internal/loopbackcheck [root]
// 退出码：0 = 未发现；1 = 发现不安全监听地址；2 = 扫描失败。
package main

import (
	"fmt"
	"os"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	findings, err := CheckDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check-loopback: 扫描失败: %v\n", err)
		os.Exit(2)
	}
	if len(findings) == 0 {
		fmt.Println("OK: no unsafe loopback addresses found")
		return
	}

	fmt.Fprintln(os.Stderr, "FAIL: found potential unsafe listen addresses (0.0.0.0 / :: / empty host)")
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "  %s:%d  %q\n", f.File, f.Line, f.Addr)
	}
	os.Exit(1)
}
