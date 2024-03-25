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
	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/podman/v4/cmd/podman/common"
	"github.com/containers/podman/v4/cmd/podman/registry"
	podUtils "github.com/containers/podman/v4/cmd/podman/utils"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	runRmi       bool
	runOption    options.RUNOption
	createOption options.CreateOption
)

func RUN(cmd *cobra.Command, args []string) error {
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	runBuilder, err := builder.FindBuilder(store, args[0])
	if err != nil {
		logrus.Errorf("not found the builder: %s", args[0])
		return err
	}
	return runBuilder.Run(args[1:], runOption)
}

func RUNCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Aliases: []string{"run-builder"},
		Args:    cobra.MinimumNArgs(2),
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
	//flags.BoolVar(&runOption.Detach, "detach", false, "set to run builder in background ")
	flags.StringVar(&runOption.Runtime, "runtime", "runc", "set runtime (runc, crun, youki)")
	flags.StringVar(&runOption.Workdir, "work-dir", "/", "set builder work-directory")
}
