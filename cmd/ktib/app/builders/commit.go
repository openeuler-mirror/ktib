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
	if len(args) == 2 {
		exportTo = args[1]
	}
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	cmBuilder, err := builder.FindBuilder(store, args[0])
	if err != nil {
		return err
	}

	return cmBuilder.Commit(exportTo)
}

func COMMITCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit [builderID/builderName] [newImageName]",
		Short: "从容器的更改创建新映像",
		Args:  cobra.RangeArgs(1, 2),
		Long: `'commit'命令从builder的更改创建新镜像。它需要一个builderID或builderName作为第一个参数，
还可以选择提供一个新的镜像名称作为第二个参数。

示例:
  # 从构建器的更改创建新映像
  ktib builders commit builderID/builderName newImageName`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commit(cmd, args)
		},
	}
	return cmd
}
