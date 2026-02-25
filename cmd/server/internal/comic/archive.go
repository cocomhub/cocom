package comic

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/cmd/server/config"
	"github.com/suixibing/cocom/pkg/archive"
	"github.com/suixibing/cocom/pkg/util"
)

func archiveComic(ctx context.Context, info *api.ComicInfo) error {
	if info == nil {
		return nil
	}
	if !info.VerifyInfo.IsValid() {
		return nil
	}
	if info.Archive != nil {
		return nil
	}
	password := config.GetArchivePassword()
	if password == "" {
		return nil
	}

	if err := os.MkdirAll(info.ArchiveDir(), 0o755); err != nil {
		return err
	}

	archivePath := info.ArchiveFilePath()
	cmdPath := config.GetArchiveCmd()
	algo := config.GetArchiveAlgorithm()
	var t archive.Type
	switch algo {
	case string(archive.TypeDouble):
		t = archive.TypeDouble
	default:
		t = archive.TypeSingle
	}
	cfg := archive.Config{CmdPath: cmdPath, Password: password}
	if err := archive.New(t).Archive(ctx, info.SaveDir(), archivePath, cfg, info.CID); err != nil {
		return err
	}

	stat, err := os.Stat(archivePath)
	if err != nil {
		return err
	}
	md5, err := util.FileMD5(archivePath)
	if err != nil {
		return err
	}
	info.Archive = &api.ArchiveInfo{
		Path:      archivePath,
		MD5:       md5,
		Size:      stat.Size(),
		CreatedAt: time.Now(),
		Algorithm: string(t),
	}
	return nil
}

func restoreComic(ctx context.Context, info *api.ComicInfo) error {
	if info == nil {
		return nil
	}
	if info.Archive == nil || info.Archive.Path == "" {
		return nil
	}
	password := config.GetArchivePassword()
	if password == "" {
		return nil
	}
	if err := os.MkdirAll(info.SaveDir(), 0o755); err != nil {
		return err
	}
	cmdPath := config.GetArchiveCmd()
	var t archive.Type
	if info.Archive.Algorithm == string(archive.TypeDouble) {
		t = archive.TypeDouble
	} else if info.Archive.Algorithm == string(archive.TypeSingle) {
		t = archive.TypeSingle
	} else {
		algo := config.GetArchiveAlgorithm()
		if algo == string(archive.TypeDouble) {
			t = archive.TypeDouble
		} else {
			t = archive.TypeSingle
		}
	}
	cfg := archive.Config{CmdPath: cmdPath, Password: password}
	return archive.New(t).Restore(ctx, info.Archive.Path, filepath.Dir(info.SaveDir()), cfg, info.CID)
}

func RestoreComicByID(ctx context.Context, cid int) error {
	info := &api.ComicInfo{}
	if err := GetComicInfo(ctx, cid, info); err != nil {
		return err
	}
	return restoreComic(ctx, info)
}
