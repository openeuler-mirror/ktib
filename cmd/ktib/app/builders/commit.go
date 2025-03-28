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

func commit(cmd *cobra.Command, args []string) error {
	exportTo := ""
	container := ""
	if len(args) == 2 {
		container = args[0]
		exportTo = args[1]
	}
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	cmBuilder, err := builder.FindBuilder(store, container)
	if err != nil {
		return err
	}

	return cmBuilder.Commit(exportTo)
}

func COMMITCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit [builderID/builderName] [newImageName]",
		Short: "Create a new image from builder changes",
		Args:  cobra.RangeArgs(1, 2),
		Long: `The 'commit' command creates a new image from changes made to the builder. It requires a builderID or builderName as the first parameter,
You can also choose to provide a new image name as the second parameter.

Example:
  #Create a new image from changes in the builder
  ktib builders commit builderID/builderName newImageName`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commit(cmd, args)
		},
	}
	return cmd
}
