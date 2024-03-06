/*
   Copyright (c) 2023 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package builders

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ADDCmd() *cobra.Command {
	var op options.BuildersOption
	cmd := &cobra.Command{
		Use:   "add",
		Short: "...",
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			destination := args[1]
			return add(cmd, source, destination, op)
		},
	}
	return cmd
}

func add(cmd *cobra.Command, source string, destination string, op options.BuildersOption) error {
	// 检查源文件是否存在
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("source file '%s' does not exist", source)
	}

	// 检查是否是压缩文件
	var reader io.Reader
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()

	if strings.HasSuffix(source, ".zip") {
		zipReader, err := zip.OpenReader(source)
		if err != nil {
			return err
		}
		defer zipReader.Close()

		for _, f := range zipReader.File {
			if f.Name == destination {
				src, err := f.Open()
				if err != nil {
					return err
				}
				defer src.Close()

				destFile, err := os.Create(destination)
				if err != nil {
					return err
				}
				defer destFile.Close()

				_, err = io.Copy(destFile, src)
				if err != nil {
					return err
				}

				fmt.Printf("File '%s' added to '%s'\n", source, destination)
				return nil
			}
		}

		return fmt.Errorf("file '%s' not found in '%s'", destination, source)
	} else if strings.HasSuffix(source, ".tar.gz") || strings.HasSuffix(source, ".tgz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzipReader.Close()

		tarReader := tar.NewReader(gzipReader)
		reader = tarReader
	} else {
		// 如果不是压缩文件，则直接使用原始文件
		reader = file
	}

	// 创建目标目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}

	// 创建目标文件
	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 复制文件内容
	writer := bufio.NewWriter(destFile)
	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}

	fmt.Printf("File '%s' added to '%s'\n", source, destination)
	return nil
}
