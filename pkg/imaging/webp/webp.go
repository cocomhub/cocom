// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package webp

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cocomhub/cocom/pkg/errwrap"
)

// 添加到文件开头的常量定义部分
const (
	// DefaultQuality WebP 编码默认质量
	DefaultQuality = 100

	// InstallScriptEndpoint WebP 工具安装脚本接口
	InstallScriptEndpoint = "/api/util/webp/install"
)

// HasWebPUtil 检查是否安装了 WebP 工具
func HasWebPUtil() bool {
	return exec.Command("cwebp", "-version").Run() == nil
}

// ConvertWebP 转换为 WebP 格式
func ConvertWebP(ctx context.Context, img image.Image) ([]byte, error) {
	dstPath := filepath.Join(os.TempDir(), "tmp.webp")
	defer os.Remove(dstPath)

	if err := SaveWebP(ctx, img, dstPath); err != nil {
		return nil, err
	}

	return os.ReadFile(dstPath)
}

// SaveWebP 保存为 WebP 格式
func SaveWebP(ctx context.Context, img image.Image, dstPath string) error {
	// 检查是否安装了 WebP 工具
	if !HasWebPUtil() {
		return errwrap.ErrImageFormat.SetIErrF("未安装 WebP 工具，请运行 'cocom install webp' 安装")
	}

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return errwrap.ErrImageDir.SetIErrF("创建目标目录失败: %v", err)
	}

	// 创建临时 PNG 文件
	tmpFile := dstPath + ".tmp.png"
	defer os.Remove(tmpFile)

	// 先保存为 PNG
	if err := SavePNG(img, tmpFile); err != nil {
		return err
	}

	// 使用 cwebp 转换为 WebP
	cmd := exec.Command("cwebp", "-q", "100", tmpFile, "-o", dstPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errwrap.ErrImageConv.SetIErrF("转换 WebP 失败: %v\n%s", err, out)
	}

	slog.DebugContext(ctx, "保存图片",
		slog.String("path", dstPath),
		slog.String("format", ".webp"))
	return nil
}

// SavePNG 保存为 PNG 格式
func SavePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return errwrap.ErrImageSave.SetIErr(err)
	}
	defer f.Close()

	return png.Encode(f, img)
}

// GetInstallEndpoint 获取安装脚本接口地址
func GetInstallEndpoint(baseURL string) string {
	return baseURL + InstallScriptEndpoint
}

// HandleWebPInstall 处理 WebP 工具安装脚本请求
func HandleWebPInstall(w http.ResponseWriter, req *http.Request) {
	osType := req.URL.Query().Get("os")
	script := GetInstallScript(osType)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(script))
}

// GetInstallScript 获取安装脚本
func GetInstallScript(osType string) string {
	if osType == "" {
		osType = runtime.GOOS
	}

	switch strings.ToLower(osType) {
	case "linux", "ubuntu", "debian":
		return `#!/bin/bash
set -e
sudo apt-get update
sudo apt-get install -y webp
cwebp -version`
	case "centos", "redhat", "fedora":
		return `#!/bin/bash
set -e
sudo yum install -y libwebp-tools
cwebp -version`
	case "darwin", "macos":
		return `#!/bin/bash
set -e
if ! command -v brew &> /dev/null; then
    echo "Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi
brew install webp
cwebp -version`
	case "windows":
		return `# PowerShell Script
if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
    Write-Host "Installing Chocolatey..."
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
}
choco install webp -y
cwebp -version`
	default:
		return fmt.Sprintf("Unsupported OS: %s", osType)
	}
}
