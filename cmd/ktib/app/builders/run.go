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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var runOption options.RUNOption

func RUN(cmd *cobra.Command, args []string, option options.RUNOption) error {
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	runBuilder, err := builder.FindBuilder(store, args[0])
	if err != nil {
		logrus.Errorf("not found the builder: %s", args[0])
		return err
	}
	return runBuilder.Run(args, option)
}

func RUNCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a command in a new container",
		Aliases: []string{"run-builder"},
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RUN(cmd, args, runOption)
		},
	}
	initFlags(cmd)
	return cmd
}

func initFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.BoolVar(&runOption.Rm, "rm", false, "Remove image unless used by other containers, implies --rm")
	flags.BoolVarP(&runOption.Detach, "detach", "d", false, "Run container in background and print container ID")
	flags.BoolVarP(&runOption.TTY, "tty", "t", false, "Allocate a pseudo-TTY")
	flags.StringVar(&runOption.Runtime, "runtime", "runc", "Runtime to use for this container")
	flags.StringVar(&runOption.Workdir, "workdir", "/", "Working directory inside the builder")
	flags.BoolVarP(&runOption.Interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
}
