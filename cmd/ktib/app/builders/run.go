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
	"fmt"
	"gitee.com/openeuler/ktib/cmd/ktib/app/images"
	"gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/podman/v4/cmd/podman/common"
	"github.com/containers/podman/v4/cmd/podman/containers"
	"github.com/containers/podman/v4/cmd/podman/registry"
	podUtils "github.com/containers/podman/v4/cmd/podman/utils"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/containers/podman/v4/pkg/specgenutil"
	"github.com/containers/storage/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"os"
)

var (
	runRmi       bool
	runOption    options.RUNOption
	createOption options.CreateOption
)

func RUN(cmd *cobra.Command, args []string) error {
	if runRmi {
		if cmd.Flags().Changed("rm") && !createOption.Rm {
			return errors.New("the --rmi option does not work without --rm")
		}
		createOption.Rm = true
	}
	if createOption.TTY && createOption.Interactive && !term.IsTerminal(int(os.Stdin.Fd())) {
		logrus.Warnf("The input device is not a TTY. The --tty and --interactive flags might not work properly")
	}
	if cmd.Flags().Changed("authfile") {
		if err := auth.CheckAuthFile(createOption.Authfile); err != nil {
			return err
		}
	}
	runOption.CIDFile = createOption.CIDFile
	runOption.Rm = createOption.Rm
	containerCreateOptions, err := containers.CreateInit(cmd, createOption.ContainerCreateOptions, false)
	if err != nil {
		return err
	}
	imgName := args[0]
	var poolop options.PullOption
	if !containerCreateOptions.RootFS {
		err := images.Pull(cmd, imgName, poolop)
		if err != nil {
			return err
		}
	}
	runOption.OutputStream = os.Stdout
	runOption.InputStream = os.Stdin
	runOption.ErrorStream = os.Stderr

	if !containerCreateOptions.Interactive {
		runOption.InputStream = nil
	}

	containerCreateOptions.PreserveFDs = runOption.PreserveFDs
	specGenerator := specgen.NewSpecGenerator(imgName, containerCreateOptions.RootFS)
	if err := specgenutil.FillOutSpecGen(specGenerator, &containerCreateOptions, args); err != nil {
		return err
	}
	ctx := registry.GetContext()
	specGenerator.RawImageName = imgName
	specGenerator.ImageOS = containerCreateOptions.OS
	specGenerator.ImageArch = containerCreateOptions.Arch
	specGenerator.ImageVariant = containerCreateOptions.Variant
	specGenerator.Passwd = &runOption.Passwd
	runOption.Spec = specGenerator
	if err != nil {
		return err
	}

	containerEngine, err := registry.NewContainerEngine(cmd, args)
	if err != nil {
		return err
	}
	report, err := containerEngine.ContainerRun(ctx, runOption.ContainerRunOptions)
	// report.ExitCode is set by ContainerRun even it returns an error
	if report != nil {
		registry.SetExitCode(report.ExitCode)
	}
	if err != nil {
		return err
	}
	if runOption.Detach {
		fmt.Println(report.Id)
		return nil
	}
	if runRmi {
		_, rmErrors := registry.ImageEngine().Remove(registry.GetContext(), []string{imgName}, entities.ImageRemoveOptions{})
		for _, err := range rmErrors {
			// ImageUnknown would be a super-unlikely race
			if !errors.Is(err, types.ErrImageUnknown) {
				// Typical case: ErrImageUsedByContainer
				logrus.Warn(err)
			}
		}
	}
	return nil
}

func RUNCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Aliases: []string{"run-builder"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RUN(cmd, args)
		},
	}
	initFlags(cmd)
	return cmd
}

func initFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SetInterspersed(false)
	common.DefineCreateDefaults(&createOption.ContainerCreateOptions)
	common.DefineCreateFlags(cmd, &createOption.ContainerCreateOptions, entities.CreateMode)
	common.DefineNetFlags(cmd)
	flags.SetNormalizeFunc(podUtils.AliasFlags)
	flags.BoolVar(&runRmi, "rmi", false, "Remove image unless used by other containers, implies --rm")
	flags.BoolVarP(&runOption.ContainerRunOptions.Detach, "detach", "d", false, "Run container in background and print container ID")
	flags.BoolVar(&runOption.ContainerRunOptions.Passwd, "passwd", true, "add entries to /etc/passwd and /etc/group")
	if registry.IsRemote() {
		_ = flags.MarkHidden("preserve-fds")
		_ = flags.MarkHidden("conmon-pidfile")
		_ = flags.MarkHidden("pidfile")
	}
	flags.IntVar(&runOption.Runtime, "runtime", 1, "set runtime (1:runc, 2:crun, 3:youki)")
}
