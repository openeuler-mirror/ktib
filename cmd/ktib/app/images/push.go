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
	"context"
	"errors"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
)

func push(cmd *cobra.Command, args []string) error {
	// TODO images, err := runtime.Push()
	pushOptions := &libimage.PushOptions{}
	store, err := utils.GetStore(cmd)
	imageName := args[0]
	destination := args[len(args)-1]
	var systemContext *types.SystemContext
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	_, err = runtime.Push(context.Background(), imageName, destination, pushOptions)
	if err != nil {
		return err
	}
	return nil
}
func PushCmd() *cobra.Command {
	var op options.PushOption
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push an images or a repository to a registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("")
			}
			return push(cmd, args)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&op.SignBy, "sign-by", "", "If non-empty, asks for a signature to be added during the copy, and specifies a key ID.")
	return cmd
}
