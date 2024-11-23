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
package version

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"

	_ "embed"
)

const (
	DefaultFormat = `Version:   %{Version}
Branch:    %{Branch}
DirtyID:   %{DirtyID}
CommitID:  %{CommitID}
Runtime:   %{Runtime}
BuiltAt:   %{BuiltAt}`
)

var (
	Version   string
	Branch    string
	DirtyID   string
	CommitID  string
	GoVersion string
	OS        string
	Arch      string
	Runtime   string
	BuiltAt   string

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
		"Version":   Version,
		"Branch":    Branch,
		"DirtyID":   DirtyID,
		"CommitID":  CommitID,
		"GoVersion": GoVersion,
		"OS":        OS,
		"Arch":      Arch,
		"Runtime":   Runtime,
		"BuiltAt":   BuiltAt,
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
