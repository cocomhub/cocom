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

type OneComicInfo struct {
	Comicid       string `json:"comicid"`
	Name          string `json:"name"`
	Desc          string `json:"desc"`
	Tag           string `json:"tag"`
	CoverImageUrl string `json:"cover_image_url"`
	Author        string `json:"author"`
	SourceUrl     string `json:"source_url"`
	SourceName    string `json:"source_name"`
	CrawlTime     string `json:"crawl_time"`
	Chapters      []struct {
		Title         string `json:"title"`
		ChapterNumber int    `json:"chapter_number"`
		SourceUrl     string `json:"source_url"`
	} `json:"chapters"`
	ExtChapters    []interface{} `json:"ext_chapters"`
	Status         string        `json:"status"`
	Tags           []interface{} `json:"tags"`
	Site           string        `json:"site"`
	LastUpdateTime string        `json:"last_update_time"`

	VerifyInfo `json:"verify" bson:"verify"`
}
