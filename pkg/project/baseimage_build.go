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

package project

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/buildah/imagebuildah"
)

// BuildImage 方法用于构建容器镜像
func (b *Bootstrap) BuildImage(imageName, tag string) error {
	// 检查 rootfs 目录是否存在
	rootfsDir := filepath.Join(b.DestinationDir, "rootfs")
	if _, err := os.Stat(rootfsDir); os.IsNotExist(err) {
		return fmt.Errorf("rootfs 目录不存在，请先运行 'ktib project build-rootfs' 命令")
	}

	// 创建临时目录用于构建
	buildDir, err := ioutil.TempDir("", "ktib-build-")
	if err != nil {
		return fmt.Errorf("创建临时构建目录失败: %v", err)
	}
	defer os.RemoveAll(buildDir)

	// 创建 rootfs.tar 文件
	rootfsTarPath := filepath.Join(buildDir, "rootfs.tar")
	if err := createTarFromDirectory(rootfsDir, rootfsTarPath); err != nil {
		return fmt.Errorf("创建 rootfs.tar 文件失败: %v", err)
	}

	// 创建 Dockerfile
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	dockerfileContent := `FROM scratch
ADD rootfs.tar /
CMD ["/bin/bash"]
`
	if err := ioutil.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("创建 Dockerfile 失败: %v", err)
	}

	// 复制 files 目录中的文件到构建目录（如果需要）
	filesDir := filepath.Join(b.DestinationDir, "files")
	if _, err := os.Stat(filesDir); err == nil {
		entries, err := ioutil.ReadDir(filesDir)
		if err != nil {
			return fmt.Errorf("读取 files 目录失败: %v", err)
		}

		for _, entry := range entries {
			srcPath := filepath.Join(filesDir, entry.Name())
			dstPath := filepath.Join(buildDir, entry.Name())

			if entry.IsDir() {
				// 如果是目录，创建对应的目录
				if err := os.MkdirAll(dstPath, 0755); err != nil {
					return fmt.Errorf("创建目录 %s 失败: %v", dstPath, err)
				}
			} else {
				// 如果是文件，复制文件内容
				srcFile, err := os.Open(srcPath)
				if err != nil {
					return fmt.Errorf("打开源文件 %s 失败: %v", srcPath, err)
				}
				defer srcFile.Close()

				dstFile, err := os.Create(dstPath)
				if err != nil {
					return fmt.Errorf("创建目标文件 %s 失败: %v", dstPath, err)
				}
				defer dstFile.Close()

				if _, err = io.Copy(dstFile, srcFile); err != nil {
					return fmt.Errorf("复制文件内容失败: %v", err)
				}
			}
		}
	}

	// 使用 ktib 内部构建接口构建镜像
	imageTag := fmt.Sprintf("%s:%s", imageName, tag)

	// 直接使用底层的 buildah 接口构建镜像，避免 cobra.Command 初始化问题
	// 创建构建选项
	buildOptions := &options.BuildOptions{
		Tags:    []string{imageTag},
		Format:  utils.DefaultFormat(),
		Rm:      true,
		ForceRm: true,
	}

	// 设置构建参数
	args := []string{buildDir}

	// 解析 Dockerfile 路径和上下文目录
	dockerfiles, contextDir, err := utils.ResolveDockerfiles(buildOptions, args)
	if err != nil {
		return fmt.Errorf("解析 Dockerfile 失败: %v", err)
	}

	// 获取存储
	store, err := utils.GetStore(nil)
	if err != nil {
		return fmt.Errorf("获取存储失败: %v", err)
	}

	// 手动创建 buildah 构建选项
	buildahOptions := &imagebuildah.BuildOptions{
		ContextDirectory:        contextDir,
		PullPolicy:              imagebuildah.PullIfMissing,
		Compression:             imagebuildah.Gzip,
		Output:                  imageTag,
		AdditionalTags:          []string{},
		Out:                     os.Stdout,
		Err:                     os.Stderr,
		ReportWriter:            os.Stderr,
		ForceRmIntermediateCtrs: true,
		RemoveIntermediateCtrs:  true,
	}

	// 构建镜像
	ctx := context.Background()
	_, _, err = imagebuildah.BuildDockerfiles(ctx, store, *buildahOptions, dockerfiles...)
	if err != nil {
		return fmt.Errorf("构建镜像失败: %v", err)
	}

	fmt.Printf("成功构建镜像: %s\n", imageTag)
	return nil
}

// createTarFromDirectory 创建一个 tar 文件，包含指定目录中的所有内容
func createTarFromDirectory(sourceDir, tarFilePath string) error {
	tarFile, err := os.Create(tarFilePath)
	if err != nil {
		return fmt.Errorf("创建 tar 文件失败: %v", err)
	}
	defer tarFile.Close()

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("获取相对路径失败: %v", err)
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		// 创建 tar 头信息
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("创建文件头信息失败: %v", err)
		}

		// 修改头信息中的名称为相对路径
		header.Name = relPath

		// 处理软链接
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("读取软链接目标失败: %v", err)
			}
			header.Linkname = linkTarget
		}

		// 写入头信息
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("写入文件头信息失败: %v", err)
		}

		// 如果是常规文件，写入文件内容
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("打开文件失败: %v", err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("写入文件内容失败: %v", err)
			}
		}

		return nil
	})
}

// createDefaultDockerfile 创建默认的 Dockerfile
func createDefaultDockerfile(projectDir string) error {
	dockerfileDir := filepath.Join(projectDir, "dockerfile")
	if err := os.MkdirAll(dockerfileDir, 0755); err != nil {
		return err
	}

	dockerfilePath := filepath.Join(dockerfileDir, "Dockerfile")
	content := `FROM scratch
ADD rootfs.tar /
CMD ["/bin/bash"]
`
	return ioutil.WriteFile(dockerfilePath, []byte(content), 0644)
}
