// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	_ "embed"

	"github.com/spf13/cobra"
)

const (
	DefaultFormat = `Version:    %{Version}
Branch:     %{Branch}
DirtyID:    %{DirtyID}
CommitID:   %{CommitID}
Runtime:    %{Runtime}
BuiltAt:    %{BuiltAt}
ReleaseURL: %{ReleaseURL}`
)

var (
	Version    string
	Branch     string
	DirtyID    string
	CommitID   string
	GoVersion  string
	OS         string
	Arch       string
	Runtime    string
	BuiltAt    string
	ReleaseURL string

	fmtKeys map[string]string
)

//go:embed build/dirty_info.txt
var DirtyInfo string

func init() {
	GoVersion = runtime.Version()
	OS = runtime.GOOS
	Arch = runtime.GOARCH
	Runtime = fmt.Sprintf("%s %s/%s", GoVersion, OS, Arch)

	if len(DirtyInfo) > 0 {
		DirtyID = fmt.Sprintf("%x", md5.Sum([]byte(DirtyInfo)))[:10]
	} else {
		DirtyID = "clean"
		DirtyInfo = "clean"
	}

	fmtKeys = map[string]string{
		"Version":    Version,
		"Branch":     Branch,
		"DirtyID":    DirtyID,
		"CommitID":   CommitID,
		"GoVersion":  GoVersion,
		"OS":         OS,
		"Arch":       Arch,
		"Runtime":    Runtime,
		"BuiltAt":    BuiltAt,
		"ReleaseURL": ReleaseURL,
	}
}

// PrintVersion 打印版本信息
func PrintVersion(w io.Writer, format string) (int, error) {
	if format == "" {
		format = DefaultFormat
	}

	for key, value := range fmtKeys {
		if strings.Contains(format, "%{"+key+"}") {
			format = strings.ReplaceAll(format, "%{"+key+"}", value)
		}
	}

	replaceMap := map[string]string{
		"\\n": "\n",
		"\\t": "\t",
	}
	for key, value := range replaceMap {
		format = strings.ReplaceAll(format, key, value)
	}

	return fmt.Fprintln(w, format)
}

// PrintVersionJSON 打印版本信息
func PrintVersionJSON(w io.Writer) (int, error) {
	data, _ := json.Marshal(fmtKeys)
	return fmt.Fprintln(w, string(data))
}

type versionFlags struct {
	outputFile string
	outputJSON bool
	format     string
}

var versionFlag versionFlags

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long: `显示编译版本、Git 提交 SHA、构建时间、运行环境等信息。

示例：
  cocom version                默认格式输出
  cocom version --json         JSON 格式输出
  cocom version -f "%{Version}:%{CommitID}"  自定义格式
  cocom version --output ver.txt  输出到文件`,
	Run: func(cmd *cobra.Command, args []string) {
		var w io.Writer = os.Stdout
		if versionFlag.outputFile != "" {
			f, err := os.Create(versionFlag.outputFile)
			if err != nil {
				fmt.Println("Failed to create file:", err)
				return
			}
			defer f.Close()
			w = f
		}

		if versionFlag.outputJSON {
			_, _ = PrintVersionJSON(w)
		} else {
			_, _ = PrintVersion(w, versionFlag.format)
		}
	},
}

var dirtyInfoCmd = &cobra.Command{
	Use:   "dirty-info",
	Short: "显示自上次提交以来的未提交变更",
	Long:  "输出当前工作目录中所有未提交的变更差异，用于脏构建的追踪。",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(DirtyInfo)
	},
}

func AddVersionCmd(rootCmd *cobra.Command) {
	rootCmd.AddCommand(versionCmd)

	versionCmd.AddCommand(dirtyInfoCmd)
	versionCmd.Flags().StringVarP(&versionFlag.outputFile, "output", "o", "", "Output the version information to a file")
	versionCmd.Flags().BoolVarP(&versionFlag.outputJSON, "json", "j", false, "Output version information in JSON format")
	versionCmd.Flags().StringVarP(&versionFlag.format, "format", "f", "", "Format the output")
}
