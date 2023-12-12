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
	"gitee.com/openeuler/ktib/cmd/ktib/app/utils"
	"github.com/spf13/cobra"
)

func listBuilders(c *cobra.Command, args []string, ops options.BuildersOption) error {
	store, err := utils.GetStore(c)
	if err != nil {
		return err
	}
	containers, err := store.Containers()
	if err != nil {
		return err
	}
	if ops.Json {
		return utils.JsonFormatBuilders(containers, ops)
	}
	return utils.FormatBuilders(containers, ops)
}

func ListBuildersCmd() *cobra.Command {
	var op options.BuildersOption
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List working builder and their base images",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listBuilders(cmd, args, op)
		},
	}
	flag := cmd.Flags()
	flag.BoolVar(&op.Json, "json", false, "output in JSON format")
	return cmd
}
