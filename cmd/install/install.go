// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/cocomhub/cocom/pkg/errwrap"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "install [component]",
	Short: "安装依赖组件",
	Long: `安装依赖组件，目前支持：
  - webp: WebP 图片格式工具（cwebp、dwebp）`,
	// 当前只支持 install webp 一个子命令，不接受多余位置参数
	Args: cobra.NoArgs,
}

var installWebpCmd = &cobra.Command{
	Use:   "webp",
	Short: "安装 WebP 工具",
	RunE: func(cmd *cobra.Command, args []string) error {
		return installWebpTools()
	},
}

func init() {
	// root registration handled in cmd/root.go
	Cmd.AddCommand(installWebpCmd)
}

func installWebpTools() error {
	// 检查是否已安装
	if exec.Command("cwebp", "-version").Run() == nil {
		fmt.Println("WebP 工具已安装")
		return nil
	}

	var err error
	switch runtime.GOOS {
	case "darwin":
		err = installWebpMacOS()
	case "linux":
		err = installWebpLinux()
	case "windows":
		err = installWebpWindows()
	default:
		return errwrap.ErrInvalidArgs.SetIErrF("不支持的操作系统: %s", runtime.GOOS)
	}

	if err != nil {
		return err
	}

	fmt.Println("WebP 工具安装完成")
	return nil
}

func installWebpMacOS() error {
	// 检查是否安装了 Homebrew
	if err := exec.Command("brew", "--version").Run(); err != nil {
		return errwrap.ErrInvalidArgs.SetIErrF("请先安装 Homebrew: https://brew.sh/")
	}

	cmd := exec.Command("brew", "install", "webp")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installWebpLinux() error {
	// 检查包管理器。sudo 的 apt-get update 失败不再被吞掉，
	// 而是立即返回错误，由 cobra RunE 以非零退出码传播。
	var cmd *exec.Cmd
	if exec.Command("apt-get", "--version").Run() == nil {
		// 以 root 运行（os.Geteuid()==0，Unix 专有）时免 sudo。
		pre := []string{}
		if os.Geteuid() != 0 {
			pre = []string{"sudo"}
		}
		updateArgs := append(pre, "apt-get", "update")
		if err := exec.Command(updateArgs[0], updateArgs[1:]...).Run(); err != nil {
			return errwrap.ErrInvalidArgs.SetIErrF("apt-get update 失败: %v", err)
		}
		installArgs := append(pre, "apt-get", "install", "-y", "webp")
		cmd = exec.Command(installArgs[0], installArgs[1:]...)
	} else if exec.Command("yum", "--version").Run() == nil {
		cmd = exec.Command("sudo", "yum", "install", "-y", "libwebp-tools")
	} else if exec.Command("dnf", "--version").Run() == nil {
		cmd = exec.Command("sudo", "dnf", "install", "-y", "libwebp-tools")
	} else {
		return errwrap.ErrInvalidArgs.SetIErrF("未找到支持的包管理器")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installWebpWindows() error {
	// 检查是否安装了 Chocolatey
	if err := exec.Command("choco", "--version").Run(); err != nil {
		return errwrap.ErrInvalidArgs.SetIErrF("请先安装 Chocolatey: https://chocolatey.org/\n" +
			"或从 https://storage.googleapis.com/downloads.webmproject.org/releases/webp/index.html 手动下载安装")
	}

	cmd := exec.Command("choco", "install", "webp")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
