// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"time"
)

type VideoInfo struct {
	Oid          string    `json:"_id" bson:"_id"`
	Vid          string    `json:"vid" bson:"vid"`
	Context      string    `json:"@context" bson:"@context"`
	Type         string    `json:"@type" bson:"@type"`
	Name         string    `json:"name" bson:"name"`
	Description  string    `json:"description" bson:"description"`
	ThumbnailUrl []string  `json:"thumbnailUrl" bson:"thumbnail_url"`
	UploadDate   time.Time `json:"uploadDate" bson:"upload_date"`
	Author       struct {
		Type string `json:"@type" bson:"@type"`
		Name string `json:"name" bson:"name"`
	} `json:"author" bson:"author"`
	ContentUrl           string `json:"contentUrl" bson:"content_url"`
	InteractionStatistic struct {
		Type            string `json:"@type" bson:"type"`
		InteractionType struct {
			Type string `json:"@type" bson:"@type"`
		} `json:"interactionType" bson:"interaction_type"`
		UserInteractionCount int `json:"userInteractionCount" bson:"user_interaction_count"`
	} `json:"interactionStatistic" bson:"interaction_statistic"`
}
