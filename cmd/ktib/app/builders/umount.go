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
	"gitee.com/openeuler/ktib/cmd/ktib/app/utils"
	"gitee.com/openeuler/ktib/pkg/builder"
	"github.com/spf13/cobra"
)

func umount(cmd *cobra.Command, args []string) error {
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		builders, err := builder.FindAllBuilders(store)
		if err != nil {
			return err
		}
		for _, b := range builders {
			err = b.UMount()
			if err != nil {
				return err
			}
			fmt.Print(b.ContainerID)
		}
	} else {
		for _, name := range args {
			b, err := builder.FindBuilder(store, name)
			if err != nil {
				return err
			}
			err = b.UMount()
			if err != nil {
				return err
			}
			fmt.Print(b.ContainerID)
		}
	}
	return nil
}

func UMOUNTCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use: "umount",
		RunE: func(cmd *cobra.Command, args []string) error {
			return umount(cmd, args)
		},
	}
	return cmd
}
