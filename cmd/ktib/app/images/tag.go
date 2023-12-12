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
	"gitee.com/openeuler/ktib/cmd/ktib/app/utils"
	"github.com/spf13/cobra"
)

func tag(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		err := errors.New("requires exactly 2 arguments")
		return err
	}
	name := args[0]
	store, err := utils.GetStore(cmd)
	if !store.Exists(name) {
		err := errors.New("image not exist")
		return err
	}
	if err != nil {
		return err
	}
	err = store.AddNames(name, args[1:])
	if err != nil {
		return err
	}
	return nil
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
