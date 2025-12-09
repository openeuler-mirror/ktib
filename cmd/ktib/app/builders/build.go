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
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/buildah/imagebuildah"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

func BUILDCmd() *cobra.Command {
	var op options.BuildOptions
	cmd := &cobra.Command{
		Use:   "build",
		Short: "build an image",
		Long: `The 'build' command builds a Docker image from a Dockerfile.
		
		Example:
		 ktib builders build ./context-dir --file Dockerfile --tag my-image:latest
		 ktib builders build -f Dockerfile -t my-image:latest . 
		
		Arguments:
		 context-dir: The directory containing the Dockerfile and other files needed for the build.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return build(cmd, args, &op)
		},
	}
	flags := cmd.Flags()
	flags.StringArrayVarP(&op.File, "file", "f", nil, "Name of the Dockerfile (Default is 'PATH/Dockerfile')")
	flags.StringSliceVarP(&op.Tags, "tag", "t", nil, "tagged name to apply to the built image")
	flags.BoolVar(&op.NoCache, "no-cache", false, "Do not use cache when building the image. (default false)")
	flags.BoolVar(&op.Rm, "rm", true, "Remove intermediate containers after a successful build. (default true)")
	flags.BoolVar(&op.ForceRm, "force-rm", true, "Always remove intermediate containers. (default true)")
	flags.BoolVar(&op.In, "stdin", false, "pass stdin into builders. (default false)")
	flags.StringVar(&op.Runtime, "runtime", "runc", "Runtime to use for build")
	flags.StringVar(&op.Format, "format", utils.DefaultFormat(), "`format` of the built image's manifest and metadata. Use BUILDAH_FORMAT environment variable to override.")
	flags.StringArrayVarP(&op.BuildArg, "build-arg", "", []string{}, "Set build-time variables")
	flags.BoolVar(&op.TLSVerify, "tls-verify", true, "Require HTTPS and verify certificates (default true)")
	flags.BoolVar(&op.Insecure, "insecure", false, "Allow insecure HTTP connections or HTTPS connections with invalid certificates")

	return cmd
}

func build(cmd *cobra.Command, args []string, op *options.BuildOptions) error {
	dockerfiles, contextDir, err := utils.ResolveDockerfiles(op, args)
	if err != nil {
		return err
	}

	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}

	ctx := context.Background()
	buildahBuildOptions, err := utils.ParseBuildOptions(cmd, op, contextDir, dockerfiles)
	if err != nil {
		return err
	}

	_, _, err = imagebuildah.BuildDockerfiles(ctx, store, *buildahBuildOptions, dockerfiles...)
	if err != nil {
		return err
	}
	return nil
}
