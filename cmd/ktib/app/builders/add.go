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
	"gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"github.com/spf13/cobra"
)

func ADDCmd() *cobra.Command {
	var op options.BuildersOption
	cmd := &cobra.Command{
		Use:   "add",
		Short: "...",
		RunE: func(cmd *cobra.Command, args []string) error {
			return add(cmd, args, op)
		},
	}
	return cmd
}

func add(cmd *cobra.Command, args []string, op options.BuildersOption) error {
	return nil
}
