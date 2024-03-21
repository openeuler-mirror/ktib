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
	"errors"
	"fmt"

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
			if len(args) < 3 {
				return errors.New("requires exactly 3 arguments")
			}
			name := args[0]
			args = tail(args)
			source := args[:len(args)-1]
			destination := args[len(args)-1]
			return Cp(cmd, name, destination, source)
		},
	}

	return cmd
}

func Cp(cmd *cobra.Command, name, destination string, source []string) error {
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	builderobj, err := builder.FindBuilder(store, name)
	if err != nil {
		return errors.New(fmt.Sprintf("Not found the %s builder", name))
	}
	return builderobj.Add(destination, source, false)
}
