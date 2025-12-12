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
package utils

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ArchiveUncompress(source, destination string) error {
	// Check if source file exists
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("source file '%s' does not exist", source)
	}

	// Check if it is a compressed file
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
		// If it is not a compressed file, use the original file directly
		reader = file
	}

	// Create the destination directory (if it does not exist)
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}

	// Create the destination file
	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file content
	writer := bufio.NewWriter(destFile)
	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}

	fmt.Printf("File '%s' added to '%s'\n", source, destination)
	return nil
}
