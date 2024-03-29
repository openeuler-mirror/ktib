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

package app

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
	"strings"

	"gitee.com/openeuler/ktib/pkg/project"
	"github.com/spf13/cobra"
)

type InitOption struct {
	BuildType string
}

var PackagesToCheck = []string{"containers-common"}

func runInit(c *cobra.Command, args []string, option InitOption) error {
	// TODO 解析参数 构建app, dir = args[0], imageName = args[1]
	if len(args) < 2 {
		logrus.Println("The number of parameters passed in is incorrect")
		return c.Help()
	}
	boot := project.NewBootstrap(args[0], args[1])
	boot.InitWorkDir(option.BuildType)
	boot.AddDockerfile()
	boot.AddScript()
	boot.AddTestcase()
	boot.AddChangeInfo()
	return nil
}

func newCmdInit() *cobra.Command {
	var option InitOption
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run this command in order to create an empty project",
		Long: `Init command helps you create an empty project with specified options. 
It creates the necessary directory structure and files to kickstart your project.`,
		Example: `  # Create a project with default options
  ktib init /path/to/project my-image
  # Create a project with source build type
  ktib init --buildType source /path/to/project my-image`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, args, option)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			//TODO init 前检查函数，检查相关rpm包是否安装：containers-common
			for _, packageName := range PackagesToCheck {
				err := checkRpmPackageInstalled(packageName)
				if err != nil {
					return fmt.Errorf("check rpm failed")
				}
			}
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// TODO init 后检查函数（可选）
			return nil
		},
		Args: cobra.ExactArgs(2),
	}
	flags := cmd.Flags()
	flags.StringVar(&option.BuildType, "buildType", "rpm", "")
	return cmd
}

func checkRpmPackageInstalled(packageName string) error {
	// 运行 rpm 命令来检查包是否已安装
	cmd := exec.Command("rpm", "-q", packageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 运行命令时出错
		return fmt.Errorf("running cmd failed")
	}
	// 检查包是否已安装
	if !strings.Contains(string(output), packageName) {
		return fmt.Errorf("%s 包未安装", packageName)
	}
	return nil
}
