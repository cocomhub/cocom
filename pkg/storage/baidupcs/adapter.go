// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package baidupcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	bdlib "github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	pcserr "github.com/qjfoidnh/BaiduPCS-Go/baidupcs/pcserror"
	"github.com/qjfoidnh/BaiduPCS-Go/requester"
	"github.com/qjfoidnh/BaiduPCS-Go/requester/multipartreader"
	"github.com/qjfoidnh/baidu-tools/tieba"
)

type Adapter interface {
	Meta(path string) (*bdlib.FileDirectory, error)
	List(path string) (bdlib.FileDirectoryList, error)
	Delete(paths ...string) error
	Copy(entries ...*bdlib.CpMvJSON) error
	Move(entries ...*bdlib.CpMvJSON) error
	Upload(ctx context.Context, localPath, targetPath string, overwrite bool) error
	Download(ctx context.Context, remotePath, localPath string) error
}

type libraryAdapter struct {
	pcs          *bdlib.BaiduPCS
	pcsUserAgent string
}

func newLibraryAdapter(config Config) (Adapter, error) {
	pcs, err := newLibraryClient(config)
	if err != nil {
		return nil, err
	}
	return &libraryAdapter{
		pcs:          pcs,
		pcsUserAgent: effectivePCSUserAgent(config),
	}, nil
}

func newLibraryClient(config Config) (*bdlib.BaiduPCS, error) {
	appID := config.AppID
	if appID == 0 {
		appID = 266719
	}
	if appID < 0 {
		return nil, fmt.Errorf("appID must be non-negative")
	}

	var pcs *bdlib.BaiduPCS
	if config.BDUSS == "" && config.Cookies != "" {
		re, _ := regexp.Compile(`BDUSS=(.+?);`)
		sub := re.FindSubmatch([]byte(config.Cookies))
		config.BDUSS = string(sub[1])
	}
	if config.BDUSS == "" {
		return nil, fmt.Errorf("bduss is required")
	}
	pcs = bdlib.NewPCS(appID, config.BDUSS)
	if strings.Contains(config.Cookies, "STOKEN=") && config.SToken == "" {
		// 未显式指定stoken则从cookies中读取
		pcs = bdlib.NewPCSWithCookieStr(appID, config.Cookies)
	}

	pcs.SetPCSUserAgent(effectivePCSUserAgent(config))
	if config.PanUserAgent == "" {
		config.PanUserAgent = bdlib.NetdiskUA
	}
	pcs.SetPanUserAgent(config.PanUserAgent)
	if config.SToken != "" {
		pcs.SetStoken(config.SToken)
	}
	if config.SBoxTKN != "" {
		pcs.SetSboxtkn(config.SBoxTKN)
	}
	if config.PCSAddr == "" {
		config.PCSAddr = bdlib.PCSBaiduCom
	}
	pcs.SetPCSAddr(config.PCSAddr)
	pcs.SetHTTPS(true)
	pcs.GetClient().SetUserAgent(effectivePCSUserAgent(config))

	if config.UID == 0 {
		t, err := tieba.NewUserInfoByBDUSS(config.BDUSS)
		if err != nil {
			return nil, err
		}
		config.UID = t.Baidu.UID
	}
	pcs.SetUID(config.UID)

	return pcs, nil
}

func effectivePCSUserAgent(config Config) string {
	if config.PCSUserAgent != "" {
		return config.PCSUserAgent
	}
	return requester.UserAgent
}

func (a *libraryAdapter) Meta(path string) (*bdlib.FileDirectory, error) {
	fd, pcsErr := a.pcs.FilesDirectoriesMeta(path)
	if pcsErr != nil {
		return nil, pcsErr
	}
	return fd, nil
}

func (a *libraryAdapter) List(path string) (bdlib.FileDirectoryList, error) {
	fds, pcsErr := a.pcs.FilesDirectoriesList(path, bdlib.DefaultOrderOptions)
	if pcsErr != nil {
		return nil, pcsErr
	}
	return fds, nil
}

func (a *libraryAdapter) Delete(paths ...string) error {
	return a.pcs.Remove(paths...)
}

func (a *libraryAdapter) Copy(entries ...*bdlib.CpMvJSON) error {
	return a.pcs.Copy(entries...)
}

func (a *libraryAdapter) Move(entries ...*bdlib.CpMvJSON) error {
	return a.pcs.Move(entries...)
}

func (a *libraryAdapter) Upload(ctx context.Context, localPath, targetPath string, overwrite bool) (err error) {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	policy := bdlib.SkipPolicy
	if overwrite {
		policy = bdlib.OverWritePolicy
	}

	startAt := time.Now()
	cmd := exec.CommandContext(ctx, "/usr/local/bin/BaiduPCS-Go", "upload", "--policy", policy, localPath, filepath.Dir(targetPath))
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	err = cmd.Run()
	if err != nil {
		slog.ErrorContext(ctx, "BaiduPCS-Go upload", "localPath", localPath, "targetPath", targetPath, "overwrite", overwrite, "cost", time.Since(startAt).String(), "err", err)
	} else {
		slog.InfoContext(ctx, "BaiduPCS-Go upload", "localPath", localPath, "targetPath", targetPath, "overwrite", overwrite, "cost", time.Since(startAt).String())
		return nil
	}

	body, pcsErr := a.pcs.PrepareUpload(policy, targetPath, func(uploadURL string, jar http.CookieJar) (_ *http.Response, err error) {
		startAt := time.Now()
		defer func() {
			if err != nil {
				slog.ErrorContext(ctx, "baidupcs upload", "localPath", localPath, "targetPath", targetPath, "overwrite", overwrite, "cost", time.Since(startAt).String(), "err", err)
			} else {
				slog.InfoContext(ctx, "baidupcs upload", "localPath", localPath, "targetPath", targetPath, "overwrite", overwrite, "cost", time.Since(startAt).String())
			}
		}()

		if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
			return nil, seekErr
		}

		mr := multipartreader.NewMultipartReader()
		mr.AddFormFile("uploadedfile", info.Name(), &fileReader{File: file, size: info.Size()})
		if closeErr := mr.CloseMultipart(); closeErr != nil {
			return nil, closeErr
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, mr)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", mr.ContentType())
		req.Header.Set("User-Agent", a.pcsUserAgent)
		req.ContentLength = mr.Len()

		client := &http.Client{Jar: jar, Timeout: 60 * time.Second}
		return client.Do(req)
	})
	if pcsErr != nil {
		return fmt.Errorf("prepare upload error: %w", pcsErr)
	}
	defer body.Close()

	if err := pcserr.DecodePCSJSONError(bdlib.OperationUpload, body); err != nil {
		return fmt.Errorf("upload error: %w", err)
	}
	return nil
}

func (a *libraryAdapter) Download(ctx context.Context, remotePath, localPath string) error {
	err := a.download(ctx, remotePath, localPath, true)
	if err == nil {
		return nil
	}
	return a.download(ctx, remotePath, localPath, false)
}

func (a *libraryAdapter) download(ctx context.Context, remotePath, localPath string, useLocateURL bool) error {
	return a.pcs.DownloadFile(remotePath, func(downloadURL string, jar http.CookieJar) (err error) {
		isLocateURL := false
		if useLocateURL {
			info, pcsError := a.pcs.LocateDownload(remotePath)
			if pcsError == nil {
				u := info.SingleURL(true)
				if u != nil {
					downloadURL = u.String()
					isLocateURL = true
				}
			}
		}
		slog.InfoContext(ctx, "baidupcs download url", "downloadURL", downloadURL, "isLocateURL", isLocateURL)

		startAt := time.Now()
		defer func() {
			if err != nil {
				slog.ErrorContext(ctx, "baidupcs download", "remotePath", remotePath, "localPath", localPath, "cost", time.Since(startAt).String(), "err", err)
			} else {
				slog.InfoContext(ctx, "baidupcs download", "remotePath", remotePath, "localPath", localPath, "cost", time.Since(startAt).String())
			}
		}()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", a.pcsUserAgent)

		client := &http.Client{Jar: jar}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode/100 != 2 {
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				return fmt.Errorf("download status %s: %w", resp.Status, readErr)
			}
			if pcsErr := pcserr.DecodePCSJSONError(bdlib.OperationDownloadFile, bytes.NewReader(body)); pcsErr != nil {
				return pcsErr
			}
			return fmt.Errorf("download status %s", resp.Status)
		}

		err = os.MkdirAll(filepath.Dir(localPath), 0o755)
		if err != nil {
			return fmt.Errorf("mkdir all error: %w", err)
		}

		out, err := os.Create(localPath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		return err
	})
}

type fileReader struct {
	*os.File
	size int64
}

func (r *fileReader) Len() int64 {
	return r.size
}
