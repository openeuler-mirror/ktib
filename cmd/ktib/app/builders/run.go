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
	"fmt"
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
	if len(args) < 2 {
		return fmt.Errorf("at least 2 arguments are required for the run command")
	}
	builderName := args[0]
	runArgs := args[1:]

	runBuilder, err := builder.FindBuilder(store, builderName)
	if err != nil {
		logrus.Errorf("not found the builder: %s", builderName)
		return err
	}

	return runBuilder.Run(runArgs, option)
}

func RUNCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run [flags...] [builderID/builderName] [cmds...]",
		Short:   "Running commands in a new container",
		Aliases: []string{"run-builder"},
		Args:    cobra.MinimumNArgs(2),
		Long: `The 'run' command runs the command in a new container based on the specified builder. The first parameter is the builder ID or name, and the remaining parameters are the commands and parameters to be executed in the container.

flags:
  --runtime string   The runtime of the container used (default to "runc")
  --workdir string   The working directory inside the container (default to "/")

  Example:
  #Run commands in the container according to the specified builder
  ktib builders run builderID/builderName echo "Hello, World!"

  #Run commands using specific runtime
  ktib builders run --runtime crun builderID/builderName echo "Hello, World!"

  #Run commands using a specific working directory
  ktib builders run --workdir /app builderID/builderName ./app-entrypoint.sh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RUN(cmd, args, runOption)
		},
	}
	initFlags(cmd)
	return cmd
}

func initFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&runOption.Runtime, "runtime", "runc", "Runtime to use for this container")
	flags.StringVar(&runOption.Workdir, "workdir", "/", "Working directory inside the builder")
}
