// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/comic"
	"github.com/cocomhub/cocom/pkg/storage"
	"github.com/cocomhub/cocom/pkg/util"
)

type VerifyInfo = comic.VerifyInfo

type ArchiveInfo struct {
	Path      string                   `json:"path,omitempty" bson:"path"`
	MD5       string                   `json:"md5,omitempty" bson:"md5"`
	Size      int64                    `json:"size,omitempty" bson:"size"`
	CreatedAt time.Time                `json:"created_at" bson:"created_at"`
	Algorithm string                   `json:"algorithm,omitempty" bson:"algorithm"`
	ByForce   bool                     `json:"by_force,omitempty" bson:"by_force"`
	Locators  []storage.StorageLocator `json:"locators,omitempty" bson:"locators"`
	storage.ReplicaHealth
}

type DownloadComicByIDRequest struct {
	Cid      int  `json:"cid"`
	MaxConn  int  `json:"max_conn"`
	MaxRetry int  `json:"max_retry"`
	Timeout  int  `json:"timeout"`
	IsSync   bool `json:"is_sync"`
	Force    bool `json:"force"`
}

type RestoreComicByIDRequest struct {
	Cid     int  `json:"cid"`
	Timeout int  `json:"timeout"`
	IsSync  bool `json:"is_sync"`
}

type ComicInfo struct {
	Oid      string      `json:"_id,omitempty" bson:"_id"`
	CID      int         `json:"cid,omitempty" bson:"cid"`
	Images   ComicImages `json:"images" bson:"images"`
	MediaId  string      `json:"media_id,omitempty" bson:"media_id"`
	NumPages int         `json:"num_pages,omitempty" bson:"num_pages"`
	Status   bool        `json:"status,omitempty" bson:"status"`
	Tags     Tags        `json:"tags,omitempty" bson:"tags"`
	Title    struct {
		English  string `json:"english,omitempty" bson:"english"`
		Japanese string `json:"japanese,omitempty" bson:"japanese"`
		Pretty   string `json:"pretty,omitempty" bson:"pretty"`
	} `json:"title" bson:"title"`
	UploadDate int64 `json:"upload_date,omitempty" bson:"upload_date"`

	VerifyInfo `json:"verify" bson:"verify"`
	Archive    *ArchiveInfo `json:"archive,omitempty" bson:"archive"`
}

func (i *ComicInfo) CheckStatus() {
	for _, page := range i.Images.Pages {
		if !page.Status {
			return
		}
	}
	i.Status = true
}

func (c *ComicInfo) saveTitle() (title string) {
	defer func() {
		title = strings.ReplaceAll(title, "/", "／")
		title = strings.ReplaceAll(title, ":", "")
		title = strings.ReplaceAll(title, "\t", "")
	}()
	if len(c.Title.Japanese) != 0 {
		return c.Title.Japanese
	}
	if len(c.Title.English) != 0 {
		return c.Title.English
	}
	if len(c.Title.Pretty) != 0 {
		return c.Title.Pretty
	}
	return fmt.Sprintf("[[unknown]]%d", c.CID)
}

func (c *ComicInfo) SaveDirName() (title string) {
	return fmt.Sprintf("[%d] %s", c.CID, c.saveTitle())
}

func (c *ComicInfo) saveDir() string {
	prefix := c.StoragePrefix()
	return fmt.Sprintf("%s/[%d] %s", prefix, c.CID, c.saveTitle())
}

func (c *ComicInfo) SaveDir() string {
	return path.Join(config.GetSaveRoot(), c.saveDir())
}

func (c *ComicInfo) StoragePrefix() string {
	return StoragePrefix(c.CID)
}

func StoragePrefix(cid int) string {
	return strings.Join(util.SplitStrRightBySize(fmt.Sprintf("%04d", cid/100), 2), "/")
}

func (c *ComicInfo) ArchiveDir() string {
	prefix := c.StoragePrefix()
	return path.Join(config.GetArchiveRoot(), prefix)
}

func (c *ComicInfo) ArchiveTempDir() string {
	return config.GetArchiveTempRoot()
}

func (c *ComicInfo) ArchiveName() string {
	return fmt.Sprintf("%d.cocoma", c.CID)
}

func (c *ComicInfo) PageSavePathByIndex(index int) string {
	return c.PageSavePathByName(c.Images.PageNameByIndex(index))
}

func (c *ComicInfo) PageSavePath(no int) string {
	return c.PageSavePathByName(c.Images.PageName(no))
}

func (c *ComicInfo) PageSavePathByName(name string) string {
	return fmt.Sprintf("%s/%s", c.SaveDir(), name)
}

var domainIds = []int{1, 2, 4}

func GetDomainId() int {
	return domainIds[util.Intn(len(domainIds))]
}

func DownloadComicOriginUrl(mediaId any, name string) string {
	return fmt.Sprintf("https://i%d.nhentai.net/galleries/%v/%s", GetDomainId(), mediaId, name)
}

func (c *ComicInfo) PageOriginUrlByIndex(index int) string {
	return c.PageOriginUrlByName(c.Images.PageNameByIndex(index))
}

func (c *ComicInfo) PageOriginUrl(no int) string {
	return c.PageOriginUrlByName(c.Images.PageName(no))
}

func (c *ComicInfo) PageOriginUrlByName(name string) string {
	return DownloadComicOriginUrl(c.MediaId, name)
}

type ComicImages struct {
	Cover     PicInfo   `json:"cover" bson:"cover"`
	Pages     []PicInfo `json:"pages,omitempty" bson:"pages"`
	Thumbnail PicInfo   `json:"thumbnail" bson:"thumbnail"`
}

func (c *ComicImages) CoverName() string {
	if !c.Cover.Status {
		return c.PageName(1)
	}
	return fmt.Sprintf("cover.%s", c.Cover.PicType())
}

func (c *ComicImages) PageNameByIndex(index int) string {
	return c.PageName(index + 1)
}

func (c *ComicImages) PageName(no int) string {
	if no < 1 || no > len(c.Pages) {
		return ""
	}
	return fmt.Sprintf("%d.%s", no, c.Pages[no-1].PicType())
}

func (c *ComicImages) PageThumbnailNameByIndex(index int) string {
	return c.PageThumbnailName(index + 1)
}

func (c *ComicImages) PageThumbnailName(no int) string {
	return c.PageName(no)
}

func (c *ComicImages) ThumbnailName() string {
	if !c.Thumbnail.Status {
		return c.PageName(1)
	}
	return fmt.Sprintf("thumb.%s", c.Thumbnail.PicType())
}

type PicInfo struct {
	H      int    `json:"h,omitempty" bson:"h"`
	T      string `json:"t,omitempty" bson:"t"`
	W      int    `json:"w,omitempty" bson:"w"`
	Status bool   `json:"status,omitempty" bson:"status"`
}

func (p PicInfo) PicType() string {
	switch p.T {
	case "j":
		return "jpg"
	case "g":
		return "gif"
	case "p":
		return "png"
	case "w":
		return "webp"
	default:
		return "jpg"
	}
}

type Tags []Tag

func (tags Tags) SubTypeTags(subType string) Tags {
	subTags := Tags{}
	for _, tag := range tags {
		if tag.Type == subType {
			subTags = append(subTags, tag)
		}
	}
	return subTags
}

func (tags Tags) IdString() string {
	buf := strings.Builder{}
	for i, tag := range tags {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(strconv.Itoa(tag.ID))
	}
	return buf.String()
}

func (tags Tags) NameString() string {
	buf := strings.Builder{}
	for i, tag := range tags {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(tag.Name)
	}
	return buf.String()
}

type Tag struct {
	Count int    `json:"count,omitempty" bson:"count"`
	ID    int    `json:"id,omitempty" bson:"id"`
	Name  string `json:"name,omitempty" bson:"name"`
	Type  string `json:"type,omitempty" bson:"type"`
	URL   string `json:"url,omitempty" bson:"url"`
}

// AutocompleteComic 自动补全中漫画的轻量信息
type AutocompleteComic struct {
	CID   int    `json:"cid"`
	Title string `json:"title"`
}

// AutocompleteResponse 自动补全响应
type AutocompleteResponse struct {
	Comics []*AutocompleteComic `json:"comics"`
	Tags   []*TagInfo           `json:"tags"`
}
