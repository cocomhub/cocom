// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"fmt"
	"net/http"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
	"github.com/cocomhub/cocom/pkg/clog"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/mutex"
)

type migrateLikeResult struct {
	CID  int     `json:"cid"`
	Diff tagDiff `json:"diff"`
}

type customItem struct {
	CID  int  `bson:"cid"`
	Like bool `bson:"like"`
}

func CustomLikeToTag(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	items := make([]*customItem, 0, 128)
	err := mongo.ComicInfoCustom().
		Filters("like", true).
		NoLimit().
		All(ctx, &items)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		clog.Errorf(ctx, "query custom like list failed. errmsg: %#v", err)
		httpwrap.ResponseFail(ctx, w, "query custom like list failed")
		return
	}

	results := make([]*migrateLikeResult, 0, len(items))
	for _, it := range items {
		unlock, lockErr := mutex.MutexLock(fmt.Sprintf("comic/%d", it.CID))
		if lockErr != nil {
			clog.Errorf(ctx, "mutex lock failed. cid[%d] errmsg: %s", it.CID, lockErr)
			continue
		}
		func() {
			defer unlock()
			info := api.ComicInfo{}
			getErr := comic.GetComicInfo(ctx, it.CID, &info)
			if getErr != nil {
				clog.Errorf(ctx, "get comic info failed. cid[%d] errmsg: %s", it.CID, getErr)
				return
			}
			like := api.Tag{Type: "custom", Name: "like", URL: "/custom/like/", ID: 99999, Count: 1}
			diff := tagDiff{Current: info.Tags}
			updated := false
			for i, t := range info.Tags {
				if t.Type == "custom" && t.Name == "like" {
					if t.ID != like.ID || t.URL != like.URL || t.Count != like.Count {
						info.Tags[i] = like
						updated = true
					}
				}
			}
			if !updated {
				exists := false
				for _, t := range info.Tags {
					if t.Type == "custom" && t.Name == "like" {
						exists = true
						break
					}
				}
				if !exists {
					info.Tags = append(info.Tags, like)
					diff.Added = append(diff.Added, like)
					updated = true
				}
			}
			if updated {
				m, encErr := info.ToMapInfo()
				if encErr != nil {
					clog.Errorf(ctx, "encode comic info failed. cid[%d] errmsg: %s", it.CID, encErr)
					return
				}
				upErr := comic.UpdateComicInfo(ctx, it.CID, m)
				if upErr != nil {
					clog.Errorf(ctx, "update comic info failed. cid[%d] errmsg: %s", it.CID, upErr)
					return
				}
				diff.Current = info.Tags
			}
			results = append(results, &migrateLikeResult{CID: it.CID, Diff: diff})
		}()
	}

	httpwrap.ResponseSucc(ctx, w, results)
}
