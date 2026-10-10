// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/cocomhub/cocom/cmd/server/api"
)

func TestGenerateMockImage_WritesDecodableJPEG(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "1.jpg")
	if err := generateMockImage(path, 120); err != nil {
		t.Fatalf("generateMockImage: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open generated image: %v", err)
	}
	defer f.Close()

	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		t.Fatalf("generated file is not a decodable JPEG: %v", err)
	}
	if cfg.Width != 1 || cfg.Height != 1 {
		t.Errorf("generated image size = %dx%d, want 1x1", cfg.Width, cfg.Height)
	}
}

func TestGenerateMockImage_MissingDirFails(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "missing-dir", "1.jpg")
	if err := generateMockImage(target, 1); err == nil {
		t.Fatal("generateMockImage into missing directory: want error, got nil")
	}
}

func TestGenerateSingleImageByInfo_PagesCoverAndThumb(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	info := &api.ComicInfo{CID: 9001}
	info.Images.Pages = []api.PicInfo{{T: "j", Status: true}, {T: "j", Status: true}}
	info.Images.Cover = api.PicInfo{T: "j", Status: true}
	info.Images.Thumbnail = api.PicInfo{T: "j", Status: true}

	if err := generateSingleImageByInfo(info, root, 5, func(int) byte { return 7 }); err != nil {
		t.Fatalf("generateSingleImageByInfo: %v", err)
	}

	saveDir := filepath.Join(root, info.StoragePrefix(), info.SaveDirName())
	for _, name := range []string{"1.jpg", "2.jpg", "cover.jpg", "thumb.jpg"} {
		st, err := os.Stat(filepath.Join(saveDir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if st.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestGenerateSingleImageByInfo_NoCoverNoThumb(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	info := &api.ComicInfo{CID: 9002}
	info.Images.Pages = []api.PicInfo{{T: "j", Status: true}}

	if err := generateSingleImageByInfo(info, root, 3, func(int) byte { return 1 }); err != nil {
		t.Fatalf("generateSingleImageByInfo: %v", err)
	}

	saveDir := filepath.Join(root, info.StoragePrefix(), info.SaveDirName())
	if _, err := os.Stat(filepath.Join(saveDir, "1.jpg")); err != nil {
		t.Fatalf("stat 1.jpg: %v", err)
	}
	if _, err := os.Stat(filepath.Join(saveDir, "cover.jpg")); !os.IsNotExist(err) {
		t.Errorf("cover.jpg should not exist when Cover.Status=false (err=%v)", err)
	}
}
