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

package images

import (
	"errors"
	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

// TODO: 有bug，执行ktib images tag image tagimage 后无报错；执行ktib images list查看只能看到tag后的名字tagimage，查看不到原镜像。
func tag(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		err := errors.New("requires exactly 2 arguments")
		return err
	}
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	imageManager, err := imagemanager.NewImageManager(store)
	if err != nil {
		return err
	}
	return imageManager.Tag(store, args)
}

func TAGCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tag(cmd, args)
		},
	}
	return cmd
}
