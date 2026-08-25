// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/archive"
	"github.com/cocomhub/cocom/pkg/archive/manager"
	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var arOutput string

// GetSourceDir 从 MongoDB 查询 ComicInfo 并返回源目录（可被测试覆盖）
var GetSourceDir func(ctx context.Context, cid int) (string, error)

var Cmd = &cobra.Command{
	Use:   "ar",
	Short: "对单个 cid 执行归档打包、解包、查询、备份与校验",
}

func init() {
	GetSourceDir = func(ctx context.Context, cid int) (string, error) {
		if cid == 0 {
			return "", errors.New("cid 不能为空")
		}
		coll := comicInfoCollection()
		var info api.ComicInfo
		if err := coll.FindOne(ctx, bson.M{"cid": cid}).Decode(&info); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return "", fmt.Errorf("cid=%d 的 comicInfo 不存在", cid)
			}
			return "", err
		}
		return info.SaveDir(), nil
	}

	var cid int
	Cmd.PersistentFlags().IntVar(&cid, "cid", 0, "comic ID")
	Cmd.PersistentFlags().StringVar(&arOutput, "output", "text", "输出格式：text|json")
	archivecli.Attach(Cmd, archivecli.Options{
		GetArchiveID: func(id int) (int, error) {
			if id > 0 && cid > 0 && id != cid {
				return 0, errors.New("归档ID与comic ID不匹配")
			} else if id > 0 {
				return id, nil
			} else if cid > 0 {
				return cid, nil
			}
			return 0, errors.New("缺少必要参数：--id 或 --cid")
		},
		OutputMode:      func() string { return arOutput },
		ReplicatePrefix: api.StoragePrefix,
		GetSourceDir:    func(ctx context.Context, id int) (string, error) { return GetSourceDir(ctx, id) },
		GetArchiveFilePath: func(ctx context.Context, id int, pack bool) (string, error) {
			return archiveFilePath(ctx, id, pack)
		},
	})
	// root registration handled in cmd/root.go
}

// archiveFilePath 计算归档文件路径：跟随 cocom.archive.path（server 布局 {path}/{prefix}/{id}.cocoma），
// 并在 pack 时基于索引中的历史版本递增（{id}-v{n+1}.cocoma）。
// 替代此前硬编码 /data/cocom/data/archive 的实现。
func archiveFilePath(ctx context.Context, id int, pack bool) (string, error) {
	suffix := archive.DefaultArchiveSuffix // ".cocoma"
	root := config.Get().Cocom.Archive.Path
	if root == "" {
		root = api.DefaultRootPaths.ArchiveRoot
	}
	dir := filepath.Join(root, api.StoragePrefix(id))

	meta, err := manager.Get().Get(ctx, id)
	if err != nil && !manager.IsNotFound(err) {
		return "", err
	} else if err == nil {
		// 索引存在：优先复用索引记录的路径；pack 时基于已有版本递增。
		if path := meta.Path; path != "" {
			if !pack {
				return path, nil
			}
			version := archive.ParseArchiveVersion(path)
			newPath := filepath.Join(filepath.Dir(path), fmt.Sprintf("%d-v%d%s", id, version+1, suffix))
			slog.InfoContext(ctx, "存档记录存在，基于存档文件路径生成新版本路径",
				"prev", path, "archive_path", newPath, "version", version+1)
			return newPath, nil
		}
	}

	defaultPath := filepath.Join(dir, fmt.Sprintf("%d%s", id, suffix))
	slog.InfoContext(ctx, "存档记录不存在，使用默认存档文件路径", "archive_path", defaultPath)
	return defaultPath, nil
}

func comicInfoCollection() *mongo.Collection {
	cfg := config.Get()
	dbName := cfg.Comic.Mongo.Database
	if dbName == "" {
		dbName = cfg.Mongo.Database
	}
	if dbName == "" {
		dbName = "cocom"
	}
	if err := mongowrap.Init(context.Background(), cfg.Mongo); err != nil {
		panic(fmt.Errorf("mongowrap init: %w", err))
	}
	db, err := mongowrap.DB(dbName)
	if err != nil {
		panic(fmt.Errorf("failed to get mongo db: %w", err))
	}
	collName := cfg.Comic.Mongo.Collections.ComicInfo
	if collName == "" {
		collName = "comicInfo"
	}
	return db.Collection(collName)
}
