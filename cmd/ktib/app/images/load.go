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
	"fmt"
	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
	"strings"
)

func LoadCmd() *cobra.Command {
	var op options.LoadOption
	cmd := &cobra.Command{
		Use:   "load",
		Short: "load images",
		Args:  utils.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return imageLoad(cmd, op)
		},
	}
	flag := cmd.Flags()
	flag.StringVarP(&op.Input, "input", "i", "", "Read from specified archive file")

	return cmd
}

func imageLoad(cmd *cobra.Command, op options.LoadOption) error {
	if len(op.Input) > 0 {
		if err := utils.Exists(op.Input); err != nil {
			return err
		}
	} else {
		return errors.New("no input, please specify the package to load")
	}
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	imageManager, err := imagemanager.NewImageManager(store)
	if err != nil {
		return err
	}
	response, err := imageManager.LoadImage(context.Background(), op)
	if err != nil {
		return err
	}
	fmt.Println("Loaded image: " + strings.Join(response.Names, "\nLoaded image: "))
	return nil
}
