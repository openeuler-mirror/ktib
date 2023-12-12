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
	"fmt"
	"gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"gitee.com/openeuler/ktib/cmd/ktib/app/utils"
	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
)

func removeImages(cmd *cobra.Command, imageName []string, op options.RemoveOption) error {
	// TODO images, err := runtime.RemoveImages()
	rmOptions := &libimage.RemoveImagesOptions{}
	rmOptions.Force = op.Force
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	var systemContext *types.SystemContext
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	_, errs := runtime.RemoveImages(context.Background(), imageName, rmOptions)
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func removeBuilders() error {
	return nil
}

func RemoveImagesCmd() *cobra.Command {
	var op options.RemoveOption
	cmd := &cobra.Command{
		Use:   "rmi",
		Short: "Remove one or more images",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please enter remove images")
			}
			return removeImages(cmd, args, op)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&op.Force, "force", "f", false, "Force will remove all builders from the local storage.")
	return cmd
}

func RemoveBuildersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rmb",
		Short: "Remove one or more working builder",
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeBuilders()
		},
	}
	return cmd
}
