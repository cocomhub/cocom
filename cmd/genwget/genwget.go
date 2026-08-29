// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package genwget

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cocomhub/cocom/cmd/server/api"
	"github.com/cocomhub/cocom/internal/config"
	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/cocomhub/cocom/pkg/httpwrap"
	"github.com/cocomhub/cocom/pkg/util"
)

var DefaultConfig = &Config{Output: "stdout"}

type Config struct {
	DstRootPath string
	Input       string
	Output      string
}

type Manager struct {
	*Config
}

func NewManager(cfg ...*Config) *Manager {
	if len(cfg) == 0 {
		cfg = append(cfg, DefaultConfig)
	}
	return &Manager{Config: cfg[0]}
}

func (m *Manager) Handle(ctx context.Context) error {
	infos, err := m.GetComicInfos(ctx)
	if err != nil {
		return err
	}

	return m.GenScript(infos)
}

var domainIds = []int{3, 5, 7}

func getDomainId() int {
	return domainIds[util.Intn(len(domainIds))]
}

// httpClient 统一用于与远端服务交互，带超时防止服务无响应时命令永久挂起。
var httpClient = &http.Client{Timeout: 30 * time.Second}

func (m *Manager) GenScript(infos []*api.ComicInfo) error {
	var w io.Writer

	switch m.Output {
	case "stdout":
		w = os.Stdout
	case "stderr":
		w = os.Stderr
	default:
		f, err := os.OpenFile(m.Output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			return errwrap.New(-1, "open output file failed").SetIErr(err)
		}
		defer f.Close()
		w = f
	}

	buf := bufio.NewWriter(w)
	_, _ = buf.WriteString("#!/bin/bash\n\nset -ex\n\n")

	for _, info := range infos {
		domainID := getDomainId()
		_, _ = fmt.Fprintf(buf, "# %d\n", info.CID)
		// 用单引号包裹路径与 URL，防止标题/目录名中的空格拆参或 '、$、反引号、; 注入。
		destDir := m.DstRootPath + "/" + info.SaveDirName()
		_, _ = fmt.Fprintf(buf, "mkdir -p %s\n", util.ShellQuote(destDir))
		for i := range info.Images.Pages {
			name := info.Images.PageNameByIndex(i)
			url := fmt.Sprintf("https://i%d.nhentai.net/galleries/%s/%s", domainID, info.MediaId, name)
			_, _ = fmt.Fprintf(buf, "wget -c -T 10 -t 10 -O %s %s\n",
				util.ShellQuote(destDir+"/"+name), util.ShellQuote(url))
		}
		_, _ = fmt.Fprintf(buf, "sleep 1\n")
	}
	return buf.Flush()
}

func (m *Manager) GetComicInfos(ctx context.Context) ([]*api.ComicInfo, error) {
	// --input 语义：字面量 "input.txt" 是历史内联约定（按文本解析 CID 列表）。
	// 若用户指定了其它路径且该路径在磁盘上存在，则按文件读取其中的 CID 列表；
	// 否则仍按内联文本处理，保持向后兼容。
	if m.Input != "input.txt" {
		if _, statErr := os.Stat(m.Input); statErr == nil {
			data, readErr := os.ReadFile(m.Input)
			if readErr != nil {
				return nil, readErr
			}
			m.Input = string(data)
		}
	} else {
		if data, err := os.ReadFile(m.Input); err == nil {
			m.Input = string(data)
		}
		// "input.txt" 不存在时保持字面量，按内联文本解析（跳过非数字字段）。
	}

	var infos []*api.ComicInfo
	m.Input = strings.ReplaceAll(m.Input, "\n", ",")
	m.Input = strings.ReplaceAll(m.Input, " ", ",")
	for str := range strings.SplitSeq(m.Input, ",") {
		str = strings.TrimSpace(str)
		if len(str) == 0 {
			continue
		}

		cid, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			slog.ErrorContext(ctx, "parse cid failed", slog.String("cid", str), slog.String("errmsg", err.Error()))
			continue
		}

		info, err := m.GetComicInfo(ctx, cid)
		if err != nil {
			slog.ErrorContext(ctx, "get comic info failed", slog.Int64("cid", cid), slog.String("errmsg", err.Error()))
			continue
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func serverAddr() string {
	return config.Get().Client.ServerAddr
}

type GetComicInfoResponse struct {
	Head httpwrap.ResponseHeadInfo `json:"head"`
	Body api.ComicInfo             `json:"body"`
}

func (m *Manager) GetComicInfo(ctx context.Context, cid int64) (*api.ComicInfo, error) {
	// 绑定 ctx 并携带超时，防止服务无响应时挂死；ctx 取消优先于客户超时。
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/comic/getComicInfo?cid=%d", serverAddr(), cid), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	info := &GetComicInfoResponse{}
	err = json.NewDecoder(resp.Body).Decode(info)
	if err != nil && err != io.EOF {
		return nil, err
	}

	if info.Head.Code != 0 {
		return nil, errwrap.New(info.Head.Code, info.Head.Msg)
	}
	return &info.Body, nil
}
