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
	"gitee.com/openeuler/ktib/pkg/options"
	utils2 "gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func SaveCmd() *cobra.Command {
	var op options.SaveOption
	cmd := &cobra.Command{
		Use:   "save",
		Short: "save images(暂未实现，目前只是一个框架)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return imageSave(cmd, args, op)
		},
	}
	flag := cmd.Flags()
	flag.StringVarP(&op.Output, "output", "o", "", "Write to a file, instead of stdout")
	return cmd
}

func imageSave(cmd *cobra.Command, args []string, op options.SaveOption) error {
	if len(args) > 1 {
		return errors.New("\"ktib images save\" requires at most 1 argument")
	}
	store, err := utils2.GetStore(cmd)
	if err != nil {
		return err
	}
	imageManager, err := imagemanager.NewImageManager(store)
	if err != nil {
		return err
	}
	tarFileName := op.Output
	err = imageManager.SaveImage(args, store, tarFileName)
	if err != nil {
		return err
	}
	return nil
}
