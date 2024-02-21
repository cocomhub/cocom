/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
