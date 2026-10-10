// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Finding 描述一处硬编码的通配监听地址。
type Finding struct {
	File string
	Line int
	Addr string
}

// skipDirs 为扫描时跳过的目录名（构建产物、依赖、历史评审归档等）。
var skipDirs = map[string]bool{
	".git":         true,
	".claude":      true,
	".trae":        true,
	".cursor":      true,
	".review":      true,
	"_review_tmp":  true,
	"vendor":       true,
	"node_modules": true,
	"build":        true,
	"testdata":     true,
	"fixtures":     true,
	"dist":         true,
	"__pycache__":  true,
}

// unsafeAddr 判断监听地址是否把所有网络接口暴露出去。
// 仅看 host 部分：""（形如 ":8080"）、"0.0.0.0"、"::" 视为不安全；
// 空字符串整体（非地址）与 "/tmp/x.sock"（unix socket）视为安全。
func unsafeAddr(addr string) bool {
	if addr == "" {
		return false
	}
	host := ""
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	} else {
		// 无端口的裸主机名形式："0.0.0.0" / "::"（裸 IPv6 无法被 SplitHostPort 解析）。
		host = strings.Trim(addr, "[]")
	}
	switch host {
	case "0.0.0.0", "::":
		return true
	case "":
		// 形如 ":8080"：省略 host 等价于监听所有接口。
		return strings.Contains(addr, ":")
	}
	return false
}

// listenAddrArgIndex 返回该调用中「监听地址」参数的下标。
// 覆盖 net.Listen / net.ListenPacket / (*net.ListenConfig).Listen(Packet) /
// http.ListenAndServe / http.ListenAndServeTLS。非监听类调用返回 ok=false。
func listenAddrArgIndex(call *ast.CallExpr) (int, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return 0, false
	}
	// 形如 ":8080" 之外的三个参数形态即 ListenConfig.Listen(ctx, network, addr)。
	threeArg := len(call.Args) >= 3
	switch sel.Sel.Name {
	case "Listen", "ListenPacket":
		if threeArg {
			return 2, true
		}
		if len(call.Args) >= 2 {
			return 1, true
		}
		return 0, false
	case "ListenAndServe", "ListenAndServeTLS":
		if len(call.Args) >= 1 {
			return 0, true
		}
	}
	return 0, false
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// scanSource 解析单个 Go 源文件，返回其中硬编码的不安全监听地址。
func scanSource(filename string, src []byte) ([]Finding, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return nil, err
	}

	var out []Finding
	report := func(pos token.Pos, addr string) {
		out = append(out, Finding{File: filename, Line: fset.Position(pos).Line, Addr: addr})
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			idx, ok := listenAddrArgIndex(node)
			if !ok || len(node.Args) <= idx {
				return true
			}
			if s, ok := stringLiteral(node.Args[idx]); ok && unsafeAddr(s) {
				report(node.Args[idx].Pos(), s)
			}
		case *ast.CompositeLit:
			// http.Server{Addr: "0.0.0.0:8080"} 这类结构体字面量。
			for _, elt := range node.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok || key.Name != "Addr" {
					continue
				}
				if s, ok := stringLiteral(kv.Value); ok && unsafeAddr(s) {
					report(kv.Value.Pos(), s)
				}
			}
		}
		return true
	})
	return out, nil
}

// CheckDir 递归扫描 root 下所有 Go 源文件（含 _test.go），返回不安全监听地址。
func CheckDir(root string) ([]Finding, error) {
	var out []Finding
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		findings, scanErr := scanSource(filepath.ToSlash(path), src)
		if scanErr != nil {
			// 语法不可解析的文件跳过（不把解析失败当成不安全监听）。
			return nil
		}
		out = append(out, findings...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
