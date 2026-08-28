// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package genwget

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/pkg/util"
)

func TestGenwget_NewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager should return non-nil")
	}
	t.Logf("Manager created: %T", mgr)
}

// TestGenScript 验证生成的 bash 脚本对路径与 URL 全部用 ShellQuote 包裹，
// 含空格/单引号/$/反引号/分号的标题与目录名不会被拆参或注入。
func TestGenScript(t *testing.T) {
	// 固定 domain id，使期望 URL 确定
	oldDomain := domainIds
	domainIds = []int{3}
	defer func() { domainIds = oldDomain }()

	buildInfo := func(cid int, mediaID, title string) *api.ComicInfo {
		info := &api.ComicInfo{
			CID:     cid,
			MediaId: mediaID,
			Images:  api.ComicImages{Pages: []api.PicInfo{{T: "j", Status: true}}},
		}
		info.Title.English = title
		return info
	}

	infos := []*api.ComicInfo{
		buildInfo(123, "456789", "Comic Title"),
		buildInfo(124, "999999", "x'; rm -rf ~; echo '"),
	}

	outFile := filepath.Join(t.TempDir(), "genwget.sh")
	mgr := NewManager(&Config{DstRootPath: "/data/dst root", Output: outFile})
	if err := mgr.GenScript(infos); err != nil {
		t.Fatalf("GenScript() error = %v", err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	want := "#!/bin/bash\n\nset -ex\n\n"
	want += fmt.Sprintf("# %d\n", 123)
	want += "mkdir -p " + util.ShellQuote("/data/dst root/[123] Comic Title") + "\n"
	want += "wget -c -T 10 -t 10 -O " + util.ShellQuote("/data/dst root/[123] Comic Title/1.jpg") + " " + util.ShellQuote("https://i3.nhentai.net/galleries/456789/1.jpg") + "\n"
	want += "sleep 1\n"
	want += fmt.Sprintf("# %d\n", 124)
	want += "mkdir -p " + util.ShellQuote("/data/dst root/[124] x'; rm -rf ~; echo '") + "\n"
	want += "wget -c -T 10 -t 10 -O " + util.ShellQuote("/data/dst root/[124] x'; rm -rf ~; echo '/1.jpg") + " " + util.ShellQuote("https://i3.nhentai.net/galleries/999999/1.jpg") + "\n"
	want += "sleep 1\n"

	if string(got) != want {
		t.Errorf("GenScript() output mismatch\n got: %q\nwant: %q", string(got), want)
	}
}
