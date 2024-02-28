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
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func COPYCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy files from the local filesystem to container",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return Cp(cmd, args)
		},
	}

	return cmd
}

func Cp(cmd *cobra.Command, args []string) error {
	store, err := utils.GetStore(cmd)
	store.GraphRoot()
	if err != nil {
		return err
	}
	option := builder.BuilderOptions{
		FromImage: args[0],
		//TODO copy的参数赋值需要定义
	}
	cpBuilder, err := builder.NewBuilder(store, option)
	if err != nil {
		return err
	}
	return cpBuilder.Copy(args)
}
