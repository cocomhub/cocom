// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func TestUnsafeAddr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		addr string
		want bool
	}{
		{"0.0.0.0:8080", true},
		{"0.0.0.0", true},
		{":8080", true},
		{"[::]:8080", true},
		{"::", true},
		{"127.0.0.1:8080", false},
		{"127.0.0.1:0", false},
		{"[::1]:8080", false},
		{"localhost:8080", false},
		{"/tmp/cocom.sock", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := unsafeAddr(tc.addr); got != tc.want {
			t.Errorf("unsafeAddr(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

func TestScanSource(t *testing.T) {
	t.Parallel()

	const src = `package demo

import (
	"net"
	"net/http"
)

func a() {
	net.Listen("tcp", "0.0.0.0:8080")
	net.Listen("tcp", ":9090")
	http.ListenAndServe("[::]:7070", nil)
	net.ListenPacket("udp", "0.0.0.0:0")

	// 安全：显式回环。
	net.Listen("tcp", "127.0.0.1:0")
	http.ListenAndServeTLS("127.0.0.1:443", "c.pem", "k.pem", nil)
	net.Listen("unix", "/tmp/cocom.sock")

	// 变量地址：静态不可判定，不报。
	addr := "0.0.0.0:1"
	net.Listen("tcp", addr)

	// 非监听调用：不报。
	net.Dial("tcp", "0.0.0.0:1")
}

var srv = &http.Server{Addr: "0.0.0.0:6060"}

// 注释中的 0.0.0.0:1 不是违规。
// 文档样例 "0.0.0.0:8080" 在字符串里但不是监听参数，不报。
var docSample = "export ADDR=0.0.0.0:8080"
`

	findings, err := scanSource("demo.go", []byte(src))
	if err != nil {
		t.Fatalf("scanSource: %v", err)
	}

	want := map[int]string{
		9:  "0.0.0.0:8080",
		10: ":9090",
		11: "[::]:7070",
		12: "0.0.0.0:0",
		27: "0.0.0.0:6060",
	}
	if len(findings) != len(want) {
		t.Fatalf("got %d findings %+v, want %d", len(findings), findings, len(want))
	}
	for _, f := range findings {
		wa, ok := want[f.Line]
		if !ok {
			t.Errorf("unexpected finding at line %d: %q", f.Line, f.Addr)
			continue
		}
		if f.Addr != wa {
			t.Errorf("line %d addr = %q, want %q", f.Line, f.Addr, wa)
		}
		delete(want, f.Line)
	}
	for line, addr := range want {
		t.Errorf("missing finding at line %d (%s)", line, addr)
	}
}

func TestScanSourceParseError(t *testing.T) {
	t.Parallel()

	if _, err := scanSource("bad.go", []byte("package")); err == nil {
		t.Fatal("scanSource with invalid source: want error, got nil")
	}
}
