// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmv

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/cocomhub/cocom/pkg/util"
)

func TestCmv_NewManager(t *testing.T) {
	mgr := NewComicMoveManager()
	if mgr == nil {
		t.Fatal("NewComicMoveManager should return non-nil")
	}
	t.Logf("ComicMoveManager created: %T", mgr)
}

// TestComicDirWriteTo 验证 WriteTo 输出的 bash 脚本对所有路径都用 ShellQuote 包裹，
// 使含空格/单引号/$/反引号/分号的目录名与文件名不会被拆参或注入。
func TestComicDirWriteTo(t *testing.T) {
	tests := []struct {
		name string
		dir  ComicDir
		want string
	}{
		{
			name: "plain paths quoted",
			dir:  ComicDir{FullPath: "/data/src/[123] comic", DstDir: "/data/dst/01/23"},
			want: fmt.Sprintf("mkdir -p %s\nmv %s %s\n",
				util.ShellQuote("/data/dst/01/23"),
				util.ShellQuote("/data/src/[123] comic"),
				util.ShellQuote("/data/dst/01/23")),
		},
		{
			name: "single quote escaped",
			dir:  ComicDir{FullPath: "/data/src/x'; rm -rf ~; echo '", DstDir: "/data/dst"},
			want: fmt.Sprintf("mkdir -p %s\nmv %s %s\n",
				util.ShellQuote("/data/dst"),
				util.ShellQuote("/data/src/x'; rm -rf ~; echo '"),
				util.ShellQuote("/data/dst")),
		},
		{
			name: "dollar and backtick preserved inside quotes",
			dir:  ComicDir{FullPath: "/data/src/$HOME/`id`;ls", DstDir: "/data/dst"},
			want: fmt.Sprintf("mkdir -p %s\nmv %s %s\n",
				util.ShellQuote("/data/dst"),
				util.ShellQuote("/data/src/$HOME/`id`;ls"),
				util.ShellQuote("/data/dst")),
		},
		{
			name: "dst exists comment line",
			dir:  ComicDir{FlagDstExist: true, FullPath: "/data/src/a", DstDir: "/data/dst/a"},
			want: "# exist same dir src(/data/src/a) dst(/data/dst/a)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tt.dir.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("WriteTo() = %q, want %q", got, tt.want)
			}
		})
	}
}
