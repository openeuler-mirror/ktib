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
	utils2 "gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func mount(cmd *cobra.Command, args []string, option *options.MountOption) error {
	var builders []*builder.Builder
	store, err := utils2.GetStore(cmd)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		builders, err = builder.FindAllBuilders(store)
		if err != nil {
			return err
		}
	} else {
		for _, name := range args {
			b, err := builder.FindBuilder(store, name)
			if err != nil {
				return err
			}
			err = b.Mount("")
			if err != nil {
				return err
			}
			builders = append(builders, b)
		}
	}
	if option.Json {
		return utils2.JsonFormatMountInfo(builders)
	}
	return utils2.FormatMountInfo(builders)
}

func MOUNTCmd() *cobra.Command {
	var op options.MountOption
	cmd := &cobra.Command{
		Use: "mount",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mount(cmd, args, &op)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&op.Json, "json", false, "output in JSON format")
	return cmd
}
