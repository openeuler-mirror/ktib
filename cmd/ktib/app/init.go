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
	"gitee.com/openeuler/ktib/pkg/project"
	"github.com/spf13/cobra"
)

type InitOption struct {
	buildType string
}

func runInit(c *cobra.Command, args []string, option InitOption) error {
	// TODO 解析参数 构建app
	if len(args) >= 0 {
		return c.Help()
	}
	boot := project.NewBootstrap("rpm or binary", "/tmp")
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, args, option)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			//TODO init 前检查函数
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// TODO init 后检查函数（可选）
			return nil
		},
		Args: cobra.NoArgs,
	}
	flags := cmd.Flags()
	flags.StringVar(&option.buildType, "buildType", "RPM", "")
	return cmd
}
