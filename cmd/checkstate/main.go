package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/suixibing/cocom/cmd/server/api"
	"github.com/suixibing/cocom/pkg/httpwrap"
)

var (
	path      = flag.String("path", "", "path to check")
	autoClean = flag.Bool("clean", false, "auto clean")
	host      = flag.String("host", "http://localhost:15456", "host to cocom")

	// “[cid] comic title”
	regexComicDir = regexp.MustCompile(`\[(\d+)\] .*`)
)

func main() {
	flag.Parse()

	if *path == "" {
		flag.Usage()
		return
	}

	cleanDir := filepath.Join(*path, "clean")
	notfoundDir := filepath.Join(*path, "notfound")

	// 遍历目录下所有目录名, 根据regexComicDir解析出cid编号
	filepath.WalkDir(*path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path == cleanDir || path == notfoundDir {
			return nil
		}

		if got := regexComicDir.FindStringSubmatch(filepath.Base(d.Name())); len(got) > 1 {
			fmt.Println(path, ":", got[1])
			info, err := GetComicInfo(*host, got[1])
			if err != nil {
				fmt.Printf("[%s] get comic info err:%s", got[1], err)
				return nil
			}
			newPath := cleanDir
			if !info.IsValid() {
				newPath = notfoundDir
				data, _ := json.Marshal(info)
				fmt.Printf("[%s] invalid: %s : %s", got[1], path, data)
			} else {
				if *autoClean {
					_ = os.RemoveAll(path)
					return nil
				}
			}
			os.MkdirAll(newPath, 0o766)
			newPath = newPath + "/" + filepath.Base(d.Name())
			err = os.Rename(path, newPath)
			if errors.Is(err, os.ErrExist) {
				err = os.Rename(path, newPath+"_"+fmt.Sprint(rand.Int()))
			}
			if err != nil {
				fmt.Printf("[%s] [ERROR] rename err: %s", got[1], err)
				return nil
			}
			fmt.Printf("[%s] rename succ: %s", got[1], newPath)
		}
		return nil
	})
}

func GetComicInfo(host, cid string) (*api.ComicInfo, error) {
	// /api/comic/getComicInfo?id=605674
	resp, err := http.Get(fmt.Sprintf("%s/api/comic/getComicInfo?id=%s", host, cid))
	if err != nil {
		return nil, fmt.Errorf("http err:%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("statusCode = %d status = %s", resp.StatusCode, resp.Status)
	}

	response := &httpwrap.ResponseInfo[api.ComicInfo]{}
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		return nil, fmt.Errorf("decode err:%w", err)
	}
	if response.Head.Code != 0 {
		return nil, fmt.Errorf("bizCode = %d msg = %s requestId = %s", response.Head.Code, response.Head.Msg, response.Head.RequestID)
	}
	return &response.Body, nil
}
