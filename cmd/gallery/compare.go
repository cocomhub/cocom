// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package gallery

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cocomhub/cocom/pkg/conv"
	"github.com/cocomhub/cocom/pkg/imaging"
	"github.com/cocomhub/cocom/pkg/util"
	"github.com/spf13/cobra"
)

type fileDiff struct {
	Filename  string `json:"filename"`
	LocalMD5  string `json:"local_md5"`
	RemoteMD5 string `json:"remote_md5"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

var nhCompareFlags = struct {
	removeSame               bool
	checkRemoveNotSame       bool
	localRootDir             string
	remoteNotExistAutoVerify bool
}{}

var nhCompareCmd = &cobra.Command{
	Use:   "compare [dirPath]",
	Short: "比较本地目录与远端存储的文件差异",
	Long: `从目录名提取 CID，通过服务 API 解析远端存储目录，比较本地与远端的文件差异。

示例：
  # 比较单个目录
  cocom gallery compare "[123456] example"

  # 批量比较本地根目录下的所有子目录
  cocom gallery compare --local-root-dir .

  # 在没有差异时自动删除本地目录
  cocom gallery compare --local-root-dir . --remove-same`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if nhCompareFlags.localRootDir == "" {
			if len(args) < 1 {
				return fmt.Errorf("缺少参数: 目录路径或使用 --local-root-dir")
			}
			compareDir(ctx, args[0])
			return nil
		}

		err := filepath.Walk(nhCompareFlags.localRootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				relPath, err := filepath.Rel(nhCompareFlags.localRootDir, path)
				if err != nil {
					return err
				}
				if relPath == "." {
					return nil
				}
				compareDir(ctx, relPath)
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			fmt.Printf("遍历本地根目录失败: %s", err)
		}
		return nil
	},
}

func init() {
	Cmd.AddCommand(nhCompareCmd)

	nhCompareCmd.Flags().BoolVar(&nhCompareFlags.removeSame, "remove-same", false, "无差异时自动删除本地目录")
	nhCompareCmd.Flags().BoolVar(&nhCompareFlags.checkRemoveNotSame, "check-remove-not-same", false, "检查并删除有差异的目录")
	nhCompareCmd.Flags().StringVar(&nhCompareFlags.localRootDir, "local-root-dir", "", "批量比较的本地根目录")
	nhCompareCmd.Flags().BoolVar(&nhCompareFlags.remoteNotExistAutoVerify, "remote-not-exist-auto-verify", false, "远程目录不存在时自动复制本地目录并启动验证")
}

func compareDir(ctx context.Context, dirPath string) {
	fmt.Println("========================================")
	defer fmt.Println("========================================")

	fmt.Printf("Extracted Dir: %s\n", dirPath)

	cid, err := extractCID(dirPath)
	if err != nil {
		fmt.Printf("Failed to extract CID: %v\n", err)
		return
	}
	fmt.Printf("Extracted CID: %s\n", cid)

	remoteDir, err := getRemoteStorageDir(cid)
	if err != nil {
		fmt.Printf("Failed to get remote storage directory: %v\n", err)
		return
	}
	fmt.Printf("Remote storage directory: %s\n", remoteDir)

	diffs, err := compareDirectories(ctx, cid, dirPath, remoteDir)
	if err != nil {
		if errors.Is(err, ErrRemoteNotExistAutoVerify) {
			fmt.Printf("[WARN] Remote directory not found, auto cp local and start verify: %v\n", err)
			return
		}
		fmt.Printf("Failed to compare directories: %v\n", err)
		return
	}

	if len(diffs) == 0 && nhCompareFlags.removeSame {
		os.RemoveAll(dirPath)
	}

	printDiffResults(dirPath, diffs)
}

func extractCID(dirPath string) (string, error) {
	dirName := filepath.Base(dirPath)
	re := regexp.MustCompile(`^\[(\d+)\].*$`)
	matches := re.FindStringSubmatch(dirName)
	if len(matches) < 2 {
		return "", fmt.Errorf("failed to extract CID from directory name: %s", dirName)
	}
	return matches[1], nil
}

func getRemoteStorageDir(cid string) (string, error) {
	url := fmt.Sprintf("%s/v2/api/nhcomic/%s/cover", serverAddr(), cid)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	storageDir := filepath.Dir(string(body))
	return storageDir, nil
}

func verifyRemoteDir(_ context.Context, cid string) error {
	url := fmt.Sprintf("%s/v2/api/nhcomic/verify", serverAddr())

	body := map[string]any{
		"autoFix": true,
		"id":      cid,
		"limit":   1,
	}

	resp, err := http.Post(url, "application/json", bytes.NewBufferString(conv.JSON(body)))
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

func calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

var ErrRemoteNotExistAutoVerify = errors.New("remote directory not found, auto cp local and start verify")

func compareDirectories(ctx context.Context, cid, localDir, remoteDir string) ([]fileDiff, error) {
	var diffs []fileDiff

	localInfo, err := os.Stat(localDir)
	if err != nil {
		return nil, fmt.Errorf("local directory not found: %w", err)
	}
	if !localInfo.IsDir() {
		return nil, fmt.Errorf("local path is not a directory: %s", localDir)
	}

	remoteInfo, err := os.Stat(remoteDir)
	if err != nil {
		if os.IsNotExist(err) && nhCompareFlags.remoteNotExistAutoVerify {
			// 自动复制本地目录到远程目录
			if err = util.CopyDir(localDir, remoteDir); err != nil {
				return nil, fmt.Errorf("failed to copy local directory to remote: %w", err)
			}
			if err = verifyRemoteDir(ctx, cid); err != nil {
				return nil, fmt.Errorf("failed to verify remote directory: %w", err)
			}
			return nil, ErrRemoteNotExistAutoVerify
		}
		return nil, fmt.Errorf("remote directory not found: %w", err)
	}
	if !remoteInfo.IsDir() {
		return nil, fmt.Errorf("remote path is not a directory: %s", remoteDir)
	}

	localFiles, err := getFileList(localDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list local files: %w", err)
	}
	remoteFiles, err := getFileList(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote files: %w", err)
	}

	localFileMap := make(map[string]string)
	for _, file := range localFiles {
		localFileMap[file] = filepath.Join(localDir, file)
	}
	remoteFileMap := make(map[string]string)
	for _, file := range remoteFiles {
		remoteFileMap[file] = filepath.Join(remoteDir, file)
	}

	for remoteFile, remotePath := range remoteFileMap {
		if localPath, exists := localFileMap[remoteFile]; exists {
			localMD5, err := calculateMD5(localPath)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate local MD5 for %s: %w", remoteFile, err)
			}
			remoteMD5, err := calculateMD5(remotePath)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate remote MD5 for %s: %w", remoteFile, err)
			}
			if localMD5 != remoteMD5 {
				_, err1 := imaging.VerifyImage(ctx, remotePath)
				_, err2 := imaging.VerifyImage(ctx, localPath)
				diffs = append(diffs, fileDiff{
					Filename:  remoteFile,
					LocalMD5:  localMD5,
					RemoteMD5: remoteMD5,
					Status:    "different",
					Message:   fmt.Sprintf("local(%v) != remote(%v)", err2, err1),
				})
			}
		} else {
			diffs = append(diffs, fileDiff{
				Filename: remoteFile,
				Status:   "localMissing",
			})
		}
	}

	for localFile := range localFileMap {
		if _, exists := remoteFileMap[localFile]; !exists {
			diffs = append(diffs, fileDiff{
				Filename: localFile,
				Status:   "remoteMissing",
			})
		}
	}

	return diffs, nil
}

func getFileList(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	return files, err
}

func printDiffResults(dirPath string, diffs []fileDiff) {
	if len(diffs) == 0 {
		fmt.Printf("%s 所有文件都匹配，没有差异！\n", dirPath)
		return
	}

	var localErr, remoteErr, unknownErr int
	fmt.Printf("\n发现 %d 个文件差异：\n", len(diffs))
	for i, diff := range diffs {
		fmt.Printf("\n%d. 文件名: %s\n", i+1, diff.Filename)
		fmt.Printf("   状态: %s\n", diff.Status)
		if diff.Status == "different" {
			fmt.Printf("   本地MD5: %s\n", diff.LocalMD5)
			fmt.Printf("   远程MD5: %s\n", diff.RemoteMD5)
			fmt.Printf("   消息: %s\n", diff.Message)
			if diff.Message == "local(<nil>) != remote(<nil>)" {
				unknownErr++
			}
			if !strings.Contains(diff.Message, "local(<nil>)") {
				localErr++
			}
			if !strings.Contains(diff.Message, "remote(<nil>)") {
				remoteErr++
			}
		}
	}
	fmt.Printf("差异统计：\n")
	fmt.Printf("  - 本地缺失文件: %d 个\n", countByStatus(diffs, "localMissing"))
	fmt.Printf("  - 远程缺失文件: %d 个\n", countByStatus(diffs, "remoteMissing"))
	fmt.Printf("  - MD5不同: %d 个\n", countByStatus(diffs, "different"))
	fmt.Printf("      - 本地异常图片: %d 个\n", localErr)
	fmt.Printf("      - 远程异常图片: %d 个\n", remoteErr)
	fmt.Printf("      - 未知异常: %d 个\n", unknownErr)

	if nhCompareFlags.checkRemoveNotSame {
		var input string
		if countByStatus(diffs, "remoteMissing") == 0 && remoteErr == 0 {
			fmt.Printf("【自动删除】因为远程缺失文件为0，远程异常图片为0\n")
			input = "y"
		} else {
			fmt.Printf("是否确认删除有差异的目录？(y/n) ")
			fmt.Scanln(&input)
		}
		if input == "y" {
			os.RemoveAll(dirPath)
			fmt.Printf("%s 目录删除操作完成。\n", dirPath)
		}
	}
}

func countByStatus(diffs []fileDiff, status string) int {
	count := 0
	for _, diff := range diffs {
		if diff.Status == status {
			count++
		}
	}
	return count
}
