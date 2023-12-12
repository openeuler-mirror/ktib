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
	"errors"
	"gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"github.com/containers/buildah/pkg/cli"
	"github.com/containers/podman/v4/cmd/podman/common"
	"github.com/containers/podman/v4/cmd/podman/registry"
	"github.com/containers/podman/v4/cmd/podman/utils"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

func BUILDCmd() *cobra.Command {
	var op options.BuildersOption
	cmd := &cobra.Command{
		Use:   "build",
		Short: "build an image",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return build(cmd, args, op)
		},
	}
	common.DefineBuildFlags(cmd, &op.BuildFlagsWrapper)
	return cmd
}

func build(cmd *cobra.Command, args []string, op options.BuildersOption) error {
	buildOptions, err := common.ParseBuildOpts(cmd, args, &op.BuildFlagsWrapper)
	if err != nil {
		return err
	}
	imageEngine, err := registry.NewImageEngine(cmd, args)
	if err != nil {
		return err
	}
	report, err := imageEngine.Build(registry.GetContext(), buildOptions.ContainerFiles, *buildOptions)
	if err != nil {
		exitCode := cli.ExecErrorCodeGeneric
		if registry.IsRemote() {
			remoteExitCode, parseErr := utils.ExitCodeFromBuildError(err.Error())
			if parseErr == nil {
				exitCode = remoteExitCode
			}
		}
		exitError := &exec.ExitError{}
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		}

		registry.SetExitCode(exitCode)
		return err
	}
	if cmd.Flag("iidfile").Changed {
		f, err := os.Create(op.Iidfile)
		if err != nil {
			return err
		}
		if _, err := f.WriteString("sha256:" + report.ID); err != nil {
			return err
		}
	}
	return nil
}
