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
	"gitee.com/openeuler/ktib/pkg/project"
	"github.com/spf13/cobra"
	"os/exec"
	"strings"
)

var PackagesToCheck = []string{"containers-common", "another-packages"}

func runInit(c *cobra.Command, args []string, option project.InitOption) error {
	// TODO 解析参数 构建app, dir = args[0], imageName = args[1]
	if len(args) < 2 {
		return c.Help()
	}
	boot := project.NewBootstrap(args[0], args[1])
	boot.InitWorkDir(option)
	boot.AddDockerfile()
	boot.AddScript()
	boot.AddTestcase()
	boot.AddChangeInfo()
	return nil
}

func newCmdInit() *cobra.Command {
	var option project.InitOption
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run this command in order to create an empty project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, args, option)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			//TODO init 前检查函数，检查相关rpm包是否安装：containers-common
			for _, packageName := range PackagesToCheck {
				err := checkRpmPackageInstalled(packageName)
				if err != nil {
					return err
				}
			}
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// TODO init 后检查函数（可选）
			return nil
		},
		Args: cobra.NoArgs,
	}
	flags := cmd.Flags()
	flags.StringVar(&option.BuildType, "buildType", "RPM", "")
	return cmd
}

func checkRpmPackageInstalled(packageName string) error {
	// 运行 rpm 命令来检查包是否已安装
	cmd := exec.Command("rpm", "-q", packageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 运行命令时出错
		return err
	}
	// 检查包是否已安装
	if !strings.Contains(string(output), packageName) {
		return fmt.Errorf("%s 包未安装", packageName)
	}
	return nil
}
