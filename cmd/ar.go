// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/internal/archivecli"
	"github.com/cocomhub/cocom/pkg/mongowrap"
	"github.com/cocomhub/cocom/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var arOutput string

var arCmd = &cobra.Command{
	Use:   "ar",
	Short: "对单个 cid 执行归档打包、解包、查询、备份与校验",
}

func init() {
	var cid int
	arCmd.PersistentFlags().IntVar(&cid, "cid", 0, "comic ID")
	arCmd.PersistentFlags().StringVar(&arOutput, "output", "text", "输出格式：text|json")
	archivecli.Attach(arCmd, archivecli.Options{
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
		GetSourceDir: func(ctx context.Context, cid int) (string, error) {
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
		},
		GetArchiveFilePath: func(ctx context.Context, cid int, pack bool) (string, error) {
			info := &api.ComicInfo{CID: cid}
			return filepath.Join(info.ArchiveDir(), info.ArchiveName()), nil
		},
	})
	rootCmd.AddCommand(arCmd)
}

func comicInfoCollection() *mongo.Collection {
	return mongowrap.DB(util.FirstNonEmpty(
		strings.TrimSpace(viper.GetString("comic.mongo.database")),
		strings.TrimSpace(viper.GetString("mongo.database")),
		"cocom",
	)).Collection(util.FirstNonEmpty(
		strings.TrimSpace(viper.GetString("comic.mongo.collections.comicInfo")),
		"comicInfo",
	))
}
