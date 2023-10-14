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
package comic

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/cmd/server/internal/errs"
	"github.com/suixibing/cocom/pkg/clog"
	"github.com/suixibing/cocom/pkg/download"
	"github.com/suixibing/cocom/pkg/errwrap"
	"github.com/suixibing/cocom/pkg/mutex"
	"github.com/suixibing/cocom/pkg/util"
)

func getComicTitle(info *api.ComicInfo) string {
	if len(info.Title.Japanese) != 0 {
		return info.Title.Japanese
	}
	if len(info.Title.Japanese) != 0 {
		return info.Title.English
	}
	if len(info.Title.Japanese) != 0 {
		return info.Title.Pretty
	}
	return fmt.Sprintf("[[unknown]]%s", info.ComicUrl)
}

func getComicDir(info *api.ComicInfo) string {
	return fmt.Sprintf("[%s] %s", info.ComicId, strings.ReplaceAll(getComicTitle(info), "/", "／"))
}

var domainIds = []int{3, 5, 7}

func getDomainId() int {
	return domainIds[util.Intn(len(domainIds))]
}

func getPicType(t string) string {
	switch t {
	case "j":
		return "jpg"
	case "g":
		return "gif"
	case "p":
		return "png"
	default:
		return "jpg"
	}
}

func CreateDownloadTaskWithLock(ctx context.Context, cid, maxConn, maxRetry int) (failed int, err error) {
	unlock, err := mutex.MutexLock(fmt.Sprintf("comic/%d", cid))
	if err != nil {
		clog.Errorf(ctx, "mutex lock failed. errmsg: %s", err)
		return -1, err
	}
	defer unlock()

	taskFailed, err := CreateDownloadTask(ctx, cid, maxConn, maxRetry)
	if err != nil {
		clog.Errorf(ctx, "download comic task failed[%d]. errmsg: %s", taskFailed, err)
	}
	return taskFailed, err
}

func CreateDownloadTasks(ctx context.Context, cids []int, maxConn, maxRetry int) (int, error) {
	wg := sync.WaitGroup{}
	wg.Add(len(cids))
	errCh := make(chan error, len(cids))
	for _, cid := range cids {
		go func(cid int) {
			defer wg.Done()

			_, err := CreateDownloadTask(ctx, cid, maxConn, maxRetry)
			if err != nil {
				errCh <- err
			}
		}(cid)
	}

	wg.Wait()
	close(errCh)

	errWrap := errwrap.NewWrap()
	for err := range errCh {
		errWrap.Add(err)
	}
	return errWrap.Count(), errWrap.Err()
}

func CreateDownloadTask(ctx context.Context, cid, maxConn, maxRetry int) (failed int, err error) {
	if maxRetry == 0 {
		maxRetry = 1
	}

	for i := 0; i < maxRetry; i++ {
		failed, err = createDownloadTask(ctx, cid, maxConn)
		//clog.Debugf(ctx, "CreateDownloadTask failed[%d] err[%v] retry[%v]", failed, err, i)
		if err != nil && err != errs.ErrComicAlreadyDownloaded {
			return
		}
		if failed == 0 {
			return
		}
	}
	err = errs.ErrComicDownloadRetryOver
	return
}

func createDownloadTask(ctx context.Context, cid, maxConn int) (int, error) {
	info := api.ComicInfo{}
	err := GetComicInfo(ctx, cid, &info)
	if err != nil {
		return 0, err
	}
	//clog.Debugf(ctx, "CreateDownloadTask cid[%d] info[%+v]", cid, info)

	if info.Status {
		return 0, errs.ErrComicAlreadyDownloaded
	}
	var tasks []*download.Task
	domainId := getDomainId()
	downloadDir := getComicDir(&info)
	for i, page := range info.Images.Pages {
		if page.Status {
			continue
		}

		name := fmt.Sprintf("%d.%s", i+1, getPicType(page.T))
		url := fmt.Sprintf("https://i%d.nhentai.net/galleries/%s/%s", domainId, info.MediaId, name)
		tasks = append(tasks, &download.Task{
			Dir:    downloadDir,
			Name:   name,
			Url:    url,
			Status: &page.Status,
		})
		clog.Debugf(ctx, "comicId[%s] dir[%s] name[%s] url[%s]", info.ComicId, downloadDir, name, url)
	}

	resultCh, err := download.DoBatch(maxConn, tasks...)
	if err != nil {
		return 0, err
	}

	errWrap := errwrap.NewWrap()
	for result := range resultCh {
		if result.Response.Err() != nil {
			errWrap.Add(fmt.Errorf("comicId[%s] dir[%s] name[%s] url[%s] download failed. errmsg: %s",
				info.ComicId, result.Task.Dir, result.Task.Name, result.Task.Url, result.Response.Err()))
			continue
		}
		if result.Task.Status != nil {
			*result.Task.Status = true
		}
	}

	info2, err := info.ToMapInfo()
	if err != nil {
		errWrap.Add(fmt.Errorf("get downloaded comic info failed. errmsg: %s", err))
	} else {
		err = UpdateComicInfo(ctx, cid, info2)
		if err != nil {
			errWrap.Add(fmt.Errorf("update downloaded comic info failed. errmsg: %s", err))
		}
	}
	return errWrap.Count(), errWrap.Err()
}
