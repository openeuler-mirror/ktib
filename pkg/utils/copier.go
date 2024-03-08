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
