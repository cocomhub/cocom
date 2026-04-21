// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage_test

import (
	"strings"
	"testing"

	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/storage/localfs"
)

func TestURI(t *testing.T) {
	dir := t.TempDir()
	fs := localfs.New("testfs", dir)
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "a/b.txt",
			args: args{
				key: "a/b.txt",
			},
			want: "localfs://testfs/a/b.txt",
		},
		{
			name: "./a/../a//b.txt",
			args: args{
				key: "./a/../a//b.txt",
			},
			want: "localfs://testfs/a/b.txt",
		},
		{
			name: "./a/../a/../b.txt",
			args: args{
				key: "./a/../a/../b.txt",
			},
			want: "localfs://testfs/b.txt",
		},
		{
			name: ".././..",
			args: args{
				key: ".././..",
			},
			want: "",
		},
	}
	for _, test := range tests {
		if got, _ := storage.URI(fs, test.args.key); got != test.want {
			t.Fatalf("%s: unexpected uri: %s, want: %s", test.name, got, test.want)
		}
	}
}

// FuzzPathNormalize ensures Path never returns traversal-prefixed keys
// and always returns forward-slash absolute form when valid.
func FuzzPathNormalize(f *testing.F) {
	seeds := []string{
		"",
		"a/b/c",
		"../x",
		"/../x",
		// `C:\Windows\..\evil`,
		"正常/文件/名.txt",
		" spaced / name .bin ",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p, err := storage.Path(s)
		if err != nil {
			// Traversal or invalid is allowed to error; nothing to assert.
			return
		}
		if !strings.HasPrefix(p, "/") {
			t.Fatalf("want leading '/', got %q", p)
		}
		if strings.HasPrefix(p[1:], "..") {
			t.Fatalf("normalized path should not start with '..': %q", p)
		}
		if strings.Contains(p, `\`) {
			t.Fatalf("normalized path should not contain backslash: %q", p)
		}
	})
}
