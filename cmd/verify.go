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
package cmd

import (
	"context"
	"fmt"
	"image"
	"os"
	"path"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/suixibing/cocom/pkg/clog"
)

var (
	verifyFilename string
	verifyDir      string
	verifyOutput   string
)

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify image integrity",
	Long: `Verify image integrity. For example:

  cocom verify -f [FILE] -o [OUTPUT_FILE]
  cocom verify -d [DIR]  -o [OUTPUT_FILE]`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var key string
		var files []string
		if len(verifyFilename) > 0 {
			key = verifyFilename
			err = verifyImageFile(cmd.Context(), verifyFilename)
			if err != nil {
				clog.Errorf(cmd.Context(), "verify image failed: %#v", err)
				fmt.Fprintf(os.Stderr, "verify image failed: %#v\n", err)
				files = append(files, verifyFilename)
			}
		} else {
			key = strings.ReplaceAll(verifyDir, "/", "---")
			files, err = verifyImageDir(cmd.Context(), verifyDir)
			if err != nil {
				clog.Errorf(cmd.Context(), "verify image dir failed: %#v", err)
				fmt.Fprintf(os.Stderr, "verify image dir failed: %#v\n", err)
				files = append(files, verifyDir+"/")
			}
		}

		if len(files) == 0 {
			fmt.Fprintf(os.Stderr, "not found err\n")
			return
		}

		f, err := os.CreateTemp(verifyOutput, fmt.Sprintf("verify__%s__.*.txt", key))
		if err != nil {
			clog.Errorf(cmd.Context(), "create temp file failed: %#v", err)
			fmt.Fprintf(os.Stderr, "create temp file failed: %#v\n", err)
			return
		}
		defer f.Close()

		fmt.Fprintf(f, strings.Join(files, "\n"))
		fmt.Fprintf(os.Stderr, "output err list to [%s]\n", path.Join(verifyOutput, f.Name()))
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)

	verifyCmd.Flags().StringVarP(&verifyFilename, "filename", "f", ``, "image file path")
	verifyCmd.Flags().StringVarP(&verifyDir, "dir", "d", `.`, "image file root path")
	verifyCmd.Flags().StringVarP(&verifyOutput, "output", "o", `.`, "failed file list")
}

func verifyImageFile(ctx context.Context, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return errors.Wrapf(err, "open file failed")
	}
	defer f.Close()

	_, err = imaging.Decode(f)
	if err != nil {
		if errors.Is(err, image.ErrFormat) {
			return nil
		}
		return errors.Wrapf(err, "image decode failed")
	}

	return nil
}

func verifyImageDir(ctx context.Context, dir string) ([]string, error) {
	fmt.Fprintf(os.Stdout, "verifyImageDir[%s]\n", dir)

	files, err := os.ReadDir(dir)
	if err != nil {
		clog.Errorf(ctx, "read dir failed: %#v", err)
		fmt.Fprintf(os.Stderr, "read dir failed: %#v\n", err)
		return nil, err
	}

	fileList := []string{}

	for _, file := range files {
		if file.IsDir() {
			files, err := verifyImageDir(ctx, path.Join(dir, file.Name()))
			if err != nil {
				clog.Errorf(ctx, "verifyImageDir[%s] read dir failed: %#v", dir, err)
				fmt.Fprintf(os.Stderr, "verifyImageDir[%s] read dir failed: %#v\n", dir, err)
				fileList = append(fileList, path.Join(dir, file.Name())+"/")
				continue
			}
			fileList = append(fileList, files...)
			continue
		}

		err = verifyImageFile(ctx, path.Join(dir, file.Name()))
		if err != nil {
			clog.Errorf(ctx, "verifyImageDir[%s] verify image failed: %#v", dir, err)
			fmt.Fprintf(os.Stderr, "verifyImageDir[%s] verify image failed: %#v\n", dir, err)
			fileList = append(fileList, path.Join(dir, file.Name()))
			continue
		}
	}
	return fileList, nil
}
