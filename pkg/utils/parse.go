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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	ktype "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/buildah/define"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/common/pkg/report"
	auth_config "github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
	container "github.com/containers/storage"
	"github.com/docker/go-units"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const unknownState = "<none>"

type imageReport struct {
	Name     string
	ID       string
	Digest   digest.Digest
	Size     string
	Created  string
	TopLayer string
}

type containerReport struct {
	ID      string
	Names   string
	LayerID string
	ImageID string
	Created string
}

func humanSize(s int64) string {
	if s < 1024 {
		return fmt.Sprintf("%.2fB", float64(s)/float64(1))
	} else if s < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(s)/float64(1024))
	} else if s < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(s)/float64(1024*1024))
	} else if s < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(s)/float64(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2fTB", float64(s)/float64(1024*1024*1024*1024))
	}
}

func sortImages(imgs []*imagemanager.Image) ([]imageReport, error) {
	var imgReport []imageReport
	for _, img := range imgs {
		size := img.Size
		createdAgo := units.HumanDuration(time.Since(img.OriImage.Created)) + " ago"
		topLayer := img.OriImage.TopLayer
		if len(topLayer) > 10 {
			topLayer = topLayer[:10]
		}

		imgID := img.OriImage.ID
		if len(imgID) > 10 {
			imgID = imgID[:10]
		}

		if len(img.OriImage.Names) > 0 {
			for _, name := range append(img.OriImage.Names, unknownState)[:len(img.OriImage.Names)] {
				imgReport = append(imgReport, imageReport{
					Name:     name,
					ID:       imgID,
					Digest:   img.OriImage.Digest,
					TopLayer: topLayer,
					Created:  createdAgo,
					Size:     humanSize(size),
				})
			}
		} else {
			imgReport = append(imgReport, imageReport{
				Name:     unknownState,
				ID:       imgID,
				Digest:   img.OriImage.Digest,
				TopLayer: topLayer,
				Created:  createdAgo,
				Size:     humanSize(size),
			})
		}
	}
	return imgReport, nil
}

func sortContainers(containers []container.Container) ([]containerReport, error) {
	var containerReports []containerReport
	for _, c := range containers {
		var containerName string
		if len(c.Names) > 0 {
			containerName = c.Names[0]
		} else {
			containerName = ""
		}
		containerReports = append(containerReports, containerReport{
			ID:      c.ID[:10],
			Names:   containerName,
			LayerID: c.LayerID,
			ImageID: c.ImageID,
			Created: units.HumanDuration(time.Since(c.Created)) + " ago",
		})
	}
	return containerReports, nil
}

func FormatImages(images []*imagemanager.Image, ops options.ImagesOption) error {
	//TODO 参考docker以image table format 输出
	defaultImageTableFormat := "table {{.Name}} {{.ID}}  {{.Size}} {{.TopLayer}}   {{.Created}}"
	defaultImageTableFormatWithDigest := "table {{.Name}} {{.ID}} {{.Digest}} {{.Size}} {{.TopLayer}} {{.Created}}"
	defaultQuietFormat := "table {{.ID}}"
	// defaultImageTableFormatWithDigest = "table {{.Repository}}\t{{.Tag}}\t{{.Digest}}\t{{.ID}}\t{{.CreatedSince}}\t{{.Size}}"
	// 构造所需的image结构=>sortImage
	imagesReport, err := sortImages(images)
	if err != nil {
		return err
	}
	headers := report.Headers(imageReport{}, map[string]string{
		"Name": "Name",
	})
	if ops.Quiet {
		defaultImageTableFormat = defaultQuietFormat
	} else if ops.Digests {
		defaultImageTableFormat = defaultImageTableFormatWithDigest
	} else if ops.Format != "" {
		defaultImageTableFormat = "table " + ops.Format
	}
	formater, err := report.New(os.Stdout, "format").Parse(report.OriginPodman, defaultImageTableFormat)
	if err != nil {
		return err
	}
	defer func() {
		err = formater.Flush()
		if err != nil {
			logrus.Error(err)
		}
	}()
	if !ops.Quiet {
		err = formater.Execute(headers)
		if err != nil {
			return err
		}
	}
	err = formater.Execute(imagesReport)
	if err != nil {
		return err
	}
	return nil
}

func JsonFormatImages(images []*imagemanager.Image, ops options.ImagesOption) error {
	var jsonImages []ktype.JsonImage

	for _, image := range images {
		jsonImages = append(jsonImages,
			ktype.JsonImage{
				Name:    image.OriImage.Names,
				Digest:  image.OriImage.Digest,
				ImageID: image.OriImage.ID,
				Created: image.OriImage.Created,
				Size:    image.Size,
			})
	}
	data, err := json.MarshalIndent(jsonImages, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func FormatBuilders(containers []container.Container, ops options.BuildersOption) error {
	// TODO 参考docker输出
	defaultBuilderTableFormat := "table {{.ID}}  {{.Names}} {{.LayerID}} {{.ImageID}}   {{.Created}}"
	containerReports, err := sortContainers(containers)
	if err != nil {
		return err
	}
	headers := report.Headers(containerReport{}, map[string]string{
		"Name": "Name",
	})
	formater, err := report.New(os.Stdout, "format").Parse(report.OriginPodman, defaultBuilderTableFormat)
	if err != nil {
		return err
	}
	defer func() {
		err = formater.Flush()
		if err != nil {
			logrus.Error(err)
		}
	}()
	err = formater.Execute(headers)
	if err != nil {
		return err
	}
	err = formater.Execute(containerReports)
	if err != nil {
		return err
	}
	return nil
}

func JsonFormatBuilders(containers []container.Container, ops options.BuildersOption) error {
	var jsonBuilders []ktype.JsonBuilder
	for _, b := range containers {
		jsonBuilders = append(jsonBuilders,
			ktype.JsonBuilder{
				ID:      b.ID,
				Names:   b.Names,
				ImageID: b.ImageID,
				Created: b.Created,
			})
	}
	data, err := json.MarshalIndent(jsonBuilders, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func JsonFormatMountInfo(builders []*builder.Builder) error {
	var jsonBuilders []ktype.JsonBuilder
	for _, b := range builders {
		if b.MountPoint != "" {
			jsonBuilders = append(jsonBuilders,
				ktype.JsonBuilder{
					ID:      b.ID,
					Mount:   b.MountPoint,
					ImageID: b.FromImageID,
				})
		}
	}
	data, err := json.MarshalIndent(jsonBuilders, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

func ParseBuildOptions(cmd *cobra.Command, flags *options.BuildOptions, contextDir string, dockerfilePaths []string) (*define.BuildOptions, error) {
	var output string
	var tags []string
	if cmd.Flag("tag").Changed {
		tags = flags.Tags
		if len(tags) > 0 {
			output = tags[0]
			tags = tags[1:]
		}
	}
	var stdout, stderr, reporter *os.File
	stdout = os.Stdout
	stderr = os.Stderr
	reporter = os.Stderr
	var stdin io.Reader
	if flags.In {
		stdin = os.Stdin
	}
	var format string
	flags.Format = strings.ToLower(flags.Format)
	switch {
	case strings.HasPrefix(flags.Format, define.OCI):
		format = define.OCIv1ImageManifest
	case strings.HasPrefix(flags.Format, define.DOCKER):
		format = define.Dockerv2ImageManifest
	default:
		return nil, fmt.Errorf("unrecognized image type %q", flags.Format)
	}
	var uselayers bool
	uselayers = true

	// 添加build-args处理
	var buildArgs map[string]string
	if len(flags.BuildArg) > 0 {
		buildArgs = make(map[string]string)
		for _, arg := range flags.BuildArg {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid build-arg value %q, must be in KEY=VALUE format", arg)
			}
			buildArgs[parts[0]] = parts[1]
		}
	}

	opts := define.BuildOptions{
		AdditionalTags:          tags,
		ContextDirectory:        contextDir,
		Err:                     stderr,
		ForceRmIntermediateCtrs: flags.ForceRm,
		Layers:                  uselayers,
		NoCache:                 flags.NoCache,
		RemoveIntermediateCtrs:  flags.Rm,
		Runtime:                 flags.Runtime,
		ReportWriter:            reporter,
		In:                      stdin,
		Out:                     stdout,
		Output:                  output,
		OutputFormat:            format,
		Args:                    buildArgs,
		// 添加 SystemContext 设置
		SystemContext: &types.SystemContext{},
	}

	// 设置认证文件路径，使用已登录的认证信息
	opts.SystemContext.AuthFilePath = auth.GetDefaultAuthFile()

	// 设置 TLS 验证
	if flags.Insecure {
		opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}

	// 设置 registries.conf 路径
	imagemanager.SetRegistriesConfPath(opts.SystemContext)

	// 获取已登录的认证信息
	credentials, err := auth_config.GetAllCredentials(opts.SystemContext)
	if err != nil || len(credentials) == 0 {
		// 没有认证信息时，使用默认的TLS验证设置
		opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolFalse
	} else {
		// 有认证信息时，检查Dockerfile中的镜像仓库是否匹配
		matchFound := false
		for _, dockerfilePath := range dockerfilePaths {
			repositories, err := ParseDockerfileFromImage(dockerfilePath)
			if err != nil {
				continue // 忽略解析错误，继续处理其他Dockerfile
			}

			for _, repo := range repositories {
				if _, exists := credentials[repo]; exists {
					matchFound = true
					break
				}
			}
			if matchFound {
				break
			}
		}

		if matchFound {
			// Dockerfile中使用的镜像来源于已登录的镜像仓库
			opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
		} else {
			// Dockerfile中使用的镜像不来源于已登录的镜像仓库
			opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolFalse
		}
	}
	return &opts, nil
}

func ResolveDockerfiles(op *options.BuildOptions, args []string) ([]string, string, error) {
	var dockerfiles []string

	// 收集 Dockerfile 路径
	for _, f := range op.File {
		if f == "-" {
			if len(args) == 0 {
				args = append(args, "-")
			} else {
				dockerfiles = append(dockerfiles, "/dev/stdin")
			}
		} else if op.File != nil {
			dockerfiles = append(dockerfiles, f)
		}
	}

	var contextDir string
	if len(args) > 0 {
		// 排除 `-`，确保上下文目录是有效的
		if args[0] != "-" {
			absDir, err := filepath.Abs(args[0])
			if err != nil {
				return nil, "", fmt.Errorf("determining path to directory %q: %w", args[0], err)
			}
			contextDir = absDir
		} else {
			// 如果 args 只有 `-`，可以选择使用当前工作目录
			var err error
			contextDir, err = os.Getwd()
			if err != nil {
				return nil, "", fmt.Errorf("determining current working directory: %w", err)
			}
		}
	} else {
		for i := range dockerfiles {
			absFile, err := filepath.Abs(dockerfiles[i])
			if err != nil {
				return nil, "", fmt.Errorf("determining path to file %q: %w", dockerfiles[i], err)
			}
			contextDir = filepath.Dir(absFile)
			dockerfiles[i] = absFile
			break
		}
	}

	if contextDir == "" {
		return nil, "", errors.New("no context directory and no Containerfile specified")
	}
	if !IsDir(contextDir) {
		return nil, "", fmt.Errorf("context must be a directory: %q", contextDir)
	}

	if len(dockerfiles) == 0 {
		switch {
		case FileExists(filepath.Join(contextDir, "Containerfile")):
			if IsDir(filepath.Join(contextDir, "Containerfile")) {
				return nil, "", fmt.Errorf("containerfile: cannot be path or directory")
			}
			dockerfiles = append(dockerfiles, filepath.Join(contextDir, "Containerfile"))
		case FileExists(filepath.Join(contextDir, "Dockerfile")):
			if IsDir(filepath.Join(contextDir, "Dockerfile")) {
				return nil, "", fmt.Errorf("dockerfile: cannot be path or directory")
			}
			dockerfiles = append(dockerfiles, filepath.Join(contextDir, "Dockerfile"))
		default:
			return nil, "", fmt.Errorf("no Containerfile or Dockerfile specified or found in context directory, %s: %w", contextDir, syscall.ENOENT)
		}
	}

	return dockerfiles, contextDir, nil
}

// ParseDockerfileFromImage 解析dockerfile，获取FROM的镜像的仓库地址
// 比如: cr.kylinos.cn/test/myapp:01，获取到cr.kylinos.cn
func ParseDockerfileFromImage(dockerfilePath string) ([]string, error) {
	var repositories []string

	// 读取Dockerfile内容
	file, err := os.Open(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dockerfile %s: %w", dockerfilePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read dockerfile %s: %w", dockerfilePath, err)
	}

	// 按行分割内容
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// 去除前后空格
		line = strings.TrimSpace(line)

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是FROM指令（不区分大小写）
		if strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			// 提取FROM后面的镜像名称
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				imageName := parts[1]

				// 解析镜像名称，去除tag和digest
				repository := parseImageRepository(imageName)
				if repository != "" {
					repositories = append(repositories, repository)
				}
			}
		}
	}

	return repositories, nil
}

// parseImageRepository 从完整的镜像名称中提取仓库地址
// 例如: cr.kylinos.cn/test/myapp:01 -> cr.kylinos.cn
// 例如: ubuntu:20.04 -> "" (没有明确的仓库地址)
// 例如: registry.io:5000/user/app@sha256:abc123 -> registry.io:5000
func parseImageRepository(imageName string) string {
	if imageName == "" {
		return ""
	}

	// 去除digest部分 (以@开头的部分)
	if idx := strings.Index(imageName, "@"); idx != -1 {
		imageName = imageName[:idx]
	}

	// 去除tag部分 (最后一个:后面的部分)
	if idx := strings.LastIndex(imageName, ":"); idx != -1 {
		// 检查:后面是否包含/，如果包含则说明这个:是仓库地址的一部分（端口号）
		tagPart := imageName[idx+1:]
		if !strings.Contains(tagPart, "/") {
			// 这是一个tag，去除它
			imageName = imageName[:idx]
		}
	}

	// 提取仓库地址部分
	// 如果镜像名称包含/，则第一个/之前的部分是仓库地址
	if idx := strings.Index(imageName, "/"); idx != -1 {
		return imageName[:idx]
	}

	// 如果没有/，没有明确的仓库地址
	return ""
}
