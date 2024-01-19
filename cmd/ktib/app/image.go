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
	imagetool "gitee.com/openeuler/ktib/cmd/ktib/app/images"
	"github.com/spf13/cobra"
)

func newCmdImage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Run this command in order to operate images at local or remote",
		// TODO 检查container依赖文件，及软件包是否安装
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Args: cobra.NoArgs,
	}
	//TODO: 需要补充images inspect
	cmd.AddCommand(
		imagetool.ImageListCmd(),
		imagetool.LoginCmd(),
		imagetool.LogoutCmd(),
		imagetool.PullCmd(),
		imagetool.PushCmd(),
		imagetool.RemoveImagesCmd(),
		imagetool.TAGCmd())
	return cmd
}
