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
	"sort"
	"strings"
	"syscall"
	"time"

	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	ktype "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/buildah/define"
	"github.com/containers/common/pkg/auth"
	containersconfig "github.com/containers/common/pkg/config"
	"github.com/containers/common/pkg/report"
	"github.com/containers/image/v5/docker/reference"
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
	Repository string
	Tag        string
	ID         string
	Digest     digest.Digest
	Size       string
	Created    string
	TopLayer   string
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

// Helper function to parse image name
func parseImageName(fullName string) (repository, tag string) {
	if fullName == "" {
		return unknownState, unknownState
	}

	// Try to parse using the standard library
	parsed, err := reference.ParseNormalizedNamed(fullName)
	if err != nil {
		// Parsing failed, try manual parsing
		return manualParseImageName(fullName)
	}

	// Get repository name
	repository = reference.FamiliarName(parsed)

	// Get tag
	if tagged, ok := parsed.(reference.Tagged); ok {
		tag = tagged.Tag()
	} else {
		// Check if it is a digest reference
		if digested, ok := parsed.(reference.Digested); ok {
			digestStr := digested.Digest().String()
			if len(digestStr) > 12 {
				tag = digestStr[:12] + "..."
			} else {
				tag = digestStr
			}
		} else {
			tag = unknownState
		}
	}

	return repository, tag
}

func manualParseImageName(fullName string) (repository, tag string) {
	// Handle special characters
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return unknownState, unknownState
	}

	// Handle common image name formats
	// Format 1: registry/repository:tag
	// Format 2: repository:tag
	// Format 3: registry:port/repository:tag

	// Find the last ":" to split the tag
	lastColon := strings.LastIndex(fullName, ":")

	if lastColon <= 0 {
		// No ":", only repository name
		return fullName, unknownState
	}

	// Check if it might be a port number (e.g., localhost:5000/image)
	// Check if there is a digit before ":" (simple check)
	if lastColon > 0 {
		// Check if the character before ":" is a digit (port number)
		charBeforeColon := fullName[lastColon-1]
		if charBeforeColon >= '0' && charBeforeColon <= '9' {
			// Might be a port number, try to find the previous ":"
			prevColon := strings.LastIndex(fullName[:lastColon], ":")
			if prevColon > 0 {
				repository = fullName[:prevColon]
				tag = fullName[prevColon+1:]
				return repository, tag
			}
		}
	}

	// Normal splitting
	repository = fullName[:lastColon]
	tag = fullName[lastColon+1:]

	// If the tag is empty
	if tag == "" {
		tag = unknownState
	}

	return repository, tag
}

func sortImages(imgs []*imagemanager.Image, ops options.ImagesOption) ([]imageReport, error) {
	var imgReport []imageReport

	for _, img := range imgs {
		size := img.Size
		createdAgo := units.HumanDuration(time.Since(img.OriImage.Created)) + " ago"

		topLayer := img.OriImage.TopLayer
		if len(topLayer) > 10 {
			topLayer = topLayer[:10]
		}

		imgID := img.OriImage.ID
		if !ops.NoTrunc {
			// NoTrunc=false (default): truncate to 10 characters
			if len(imgID) > 10 {
				imgID = imgID[:10]
			}
		} else {
			// NoTrunc=true: truncate to 12 characters
			if len(imgID) > 12 {
				imgID = imgID[:12]
			}
		}

		if len(img.OriImage.Names) > 0 {
			for _, name := range img.OriImage.Names {
				repository, tag := parseImageName(name)

				imgReport = append(imgReport, imageReport{
					Repository: repository,
					Tag:        tag,
					ID:         imgID,
					Digest:     img.OriImage.Digest,
					TopLayer:   topLayer,
					Created:    createdAgo,
					Size:       humanSize(size),
				})
			}
		} else {
			// Case with no names
			imgReport = append(imgReport, imageReport{
				Repository: unknownState,
				Tag:        unknownState,
				ID:         imgID,
				Digest:     img.OriImage.Digest,
				TopLayer:   topLayer,
				Created:    createdAgo,
				Size:       humanSize(size),
			})
		}
	}

	// Sort by Repository
	sort.Slice(imgReport, func(i, j int) bool {
		return imgReport[i].Repository < imgReport[j].Repository
	})
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
	// Define output format
	defaultImageTableFormat := "table {{.Repository}} {{.Tag}} {{.ID}} {{.Size}} {{.Created}}"
	defaultImageTableFormatWithDigest := "table {{.Repository}} {{.Tag}} {{.ID}} {{.Digest}} {{.Size}} {{.Created}}"
	defaultQuietFormat := "table {{.ID}}"
	// defaultImageTableFormatWithDigest = "table {{.Repository}}\t{{.Tag}}\t{{.Digest}}\t{{.ID}}\t{{.CreatedSince}}\t{{.Size}}"
	// Construct the required image structure => sortImage
	imagesReport, err := sortImages(images, ops)
	if err != nil {
		return err
	}

	// Define table header mapping
	headers := report.Headers(imageReport{}, map[string]string{
		"Repository": "REPOSITORY",
		"Tag":        "TAG",
		"ID":         "IMAGE ID",
		"Size":       "SIZE",
		"Created":    "CREATED",
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
	// TODO Refer to docker output
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

	// Add build-args handling
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

	commonBuildOpts := &define.CommonBuildOptions{}
	if _, err := os.Stat(containersconfig.SeccompOverridePath); err == nil {
		commonBuildOpts.SeccompProfilePath = containersconfig.SeccompOverridePath
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	} else if _, err := os.Stat(containersconfig.SeccompDefaultPath); err == nil {
		commonBuildOpts.SeccompProfilePath = containersconfig.SeccompDefaultPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	opts := define.BuildOptions{
		AdditionalTags:          tags,
		CommonBuildOpts:         commonBuildOpts,
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
		// Add SystemContext setting
		SystemContext: &types.SystemContext{},
	}

	// Set authentication file path, using logged-in authentication information
	opts.SystemContext.AuthFilePath = auth.GetDefaultAuthFile()

	// Set TLS verification
	if flags.Insecure {
		opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}

	// Set registries.conf path
	imagemanager.SetRegistriesConfPath(opts.SystemContext)

	// Get logged-in authentication information
	credentials, err := auth_config.GetAllCredentials(opts.SystemContext)
	if err != nil || len(credentials) == 0 {
		// If there is no authentication information, use the default TLS verification setting
		opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolFalse
	} else {
		// If there is authentication information, check if the repositories in the Dockerfile match the logged-in ones
		matchFound := false
		for _, dockerfilePath := range dockerfilePaths {
			repositories, err := ParseDockerfileFromImage(dockerfilePath)
			if err != nil {
				continue // Ignore parsing errors, continue processing other Dockerfiles
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
			// The image used in the Dockerfile comes from a logged-in registry
			opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
		} else {
			// The image used in the Dockerfile does not come from a logged-in registry
			opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolFalse
		}
	}
	return &opts, nil
}

func ResolveDockerfiles(op *options.BuildOptions, args []string) ([]string, string, error) {
	var dockerfiles []string

	// Collect Dockerfile paths
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
		// Exclude `-` to ensure the context directory is valid
		if args[0] != "-" {
			absDir, err := filepath.Abs(args[0])
			if err != nil {
				return nil, "", fmt.Errorf("determining path to directory %q: %w", args[0], err)
			}
			contextDir = absDir
		} else {
			// If args only contains `-`, choose to use the current working directory
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

// ParseDockerfileFromImage parses the Dockerfile to get the repository addresses of the FROM images
// For example: cr.kylinos.cn/test/myapp:01, gets cr.kylinos.cn
func ParseDockerfileFromImage(dockerfilePath string) ([]string, error) {
	var repositories []string

	// Read Dockerfile content
	file, err := os.Open(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dockerfile %s: %w", dockerfilePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read dockerfile %s: %w", dockerfilePath, err)
	}

	// Split content by line
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// Trim leading and trailing spaces
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if it is a FROM instruction (case-insensitive)
		if strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			// Extract the image name after FROM
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				imageName := parts[1]

				// Parse the image name to get the repository
				repository := parseImageRepository(imageName)
				if repository != "" {
					repositories = append(repositories, repository)
				}
			}
		}
	}

	return repositories, nil
}

// parseImageRepository extracts the repository address from the full image name
// Example: cr.kylinos.cn/test/myapp:01 -> cr.kylinos.cn
// Example: ubuntu:20.04 -> "" (no explicit repository address)
// Example: registry.io:5000/user/app@sha256:abc123 -> registry.io:5000
func parseImageRepository(imageName string) string {
	if imageName == "" {
		return ""
	}

	// Remove the digest part (the part starting with @)
	if idx := strings.Index(imageName, "@"); idx != -1 {
		imageName = imageName[:idx]
	}

	// Remove the tag part (the part after the last :)
	if idx := strings.LastIndex(imageName, ":"); idx != -1 {
		// Check if the part after : contains /, if it does, it means this : is part of the repository address (port number)
		tagPart := imageName[idx+1:]
		if !strings.Contains(tagPart, "/") {
			// This is a tag, remove it
			imageName = imageName[:idx]
		}
	}

	// Extract the repository address part
	// If the image name contains /, the part before the first / is the repository address
	if idx := strings.Index(imageName, "/"); idx != -1 {
		return imageName[:idx]
	}

	// If there is no /, there is no explicit repository address
	return ""
}
