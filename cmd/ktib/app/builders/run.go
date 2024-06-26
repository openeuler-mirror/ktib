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
		Use:     "run [builderID/builderName] [命令] [参数...]",
		Short:   "在新容器中运行命令",
		Aliases: []string{"run-builder"},
		Args:    cobra.MinimumNArgs(2),
		Long: `'run'命令根据指定的构建器在新容器中运行命令。第一个参数是构建器ID或名称,剩余参数是要在容器中执行的命令和参数。

选项:
  --runtime string   使用的容器运行时(默认为"runc")
  --workdir string   容器内的工作目录(默认为"/")  

示例:
  # 根据指定的构建器在容器中运行命令
  ktib builders run builderID/builderName echo "Hello, World!"

  # 使用特定运行时运行命令
  ktib builders run --runtime crun builderID/builderName echo "Hello, World!"

  # 使用特定工作目录运行命令
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
