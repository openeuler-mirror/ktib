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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/buildah/define"
	"github.com/containers/common/pkg/auth"
	containersconfig "github.com/containers/common/pkg/config"
	auth_config "github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
)

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
	uselayers := true

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
		AddCapabilities:         flags.CapAdd,
		CommonBuildOpts:         commonBuildOpts,
		ContextDirectory:        contextDir,
		DropCapabilities:        flags.CapDrop,
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
		SystemContext:           &types.SystemContext{},
	}

	opts.SystemContext.AuthFilePath = auth.GetDefaultAuthFile()
	if flags.Insecure {
		opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}

	imagemanager.SetRegistriesConfPath(opts.SystemContext)

	credentials, err := auth_config.GetAllCredentials(opts.SystemContext)
	if err != nil || len(credentials) == 0 {
		opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolFalse
	} else {
		matchFound := false
		for _, dockerfilePath := range dockerfilePaths {
			repositories, err := ParseDockerfileFromImage(dockerfilePath)
			if err != nil {
				continue
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
			opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
		} else {
			opts.SystemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolFalse
		}
	}
	return &opts, nil
}

func ResolveDockerfiles(op *options.BuildOptions, args []string) ([]string, string, error) {
	var dockerfiles []string

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
		if args[0] != "-" {
			absDir, err := filepath.Abs(args[0])
			if err != nil {
				return nil, "", fmt.Errorf("determining path to directory %q: %w", args[0], err)
			}
			contextDir = absDir
		} else {
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
