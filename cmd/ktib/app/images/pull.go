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
	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
)

func Pull(cmd *cobra.Command, imageName string, ops options.PullOption) error {
	// TODO images, err := runtime.Pull()
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	var systemContext *types.SystemContext
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}
	ctx := context.Background()
	pullPolicy, err := config.ParsePullPolicy("always")
	if err != nil {
		return err
	}
	pullOptions := &libimage.PullOptions{}
	images, err := runtime.Pull(ctx, imageName, pullPolicy, pullOptions)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", images[0].ID())
	return nil
}

func PullCmd() *cobra.Command {
	var op options.PullOption
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull an images or a repository from a registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			op.Remote = args[0]
			return Pull(cmd, op.Remote, op)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&op.Platform, "platform", "", "Set platform if server is multi-platform capable")
	return cmd
}
