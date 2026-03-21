// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/cmd/server/internal/comic"
	"github.com/cocomhub/cocom/cmd/server/internal/mongo"
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
		slog.ErrorContext(ctx, "query custom like list failed", slog.String("errmsg", err.Error()))
		httpwrap.ResponseFail(ctx, w, "query custom like list failed")
		return
	}

	results := make([]*migrateLikeResult, 0, len(items))
	for _, it := range items {
		unlock, lockErr := mutex.Lock(ctx, fmt.Sprintf("comic/%d", it.CID))
		if lockErr != nil {
			slog.ErrorContext(ctx, "mutex lock failed", slog.Int("cid", it.CID), slog.String("errmsg", lockErr.Error()))
			continue
		}
		func() {
			defer unlock()
			info := api.ComicInfo{}
			getErr := comic.GetComicInfo(ctx, it.CID, &info)
			if getErr != nil {
				slog.ErrorContext(ctx, "get comic info failed", slog.Int("cid", it.CID), slog.String("errmsg", getErr.Error()))
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
					slog.ErrorContext(ctx, "encode comic info failed", slog.Int("cid", it.CID), slog.String("errmsg", encErr.Error()))
					return
				}
				upErr := comic.UpdateComicInfo(ctx, it.CID, m)
				if upErr != nil {
					slog.ErrorContext(ctx, "update comic info failed", slog.Int("cid", it.CID), slog.String("errmsg", upErr.Error()))
					return
				}
				diff.Current = info.Tags
			}
			results = append(results, &migrateLikeResult{CID: it.CID, Diff: diff})
		}()
	}

	httpwrap.ResponseSucc(ctx, w, results)
}
