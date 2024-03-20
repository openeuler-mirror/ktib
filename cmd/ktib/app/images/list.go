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

// TODO: 当dockerfile制作镜像时，超过一层会panic；commit已有镜像名也会panic
func imageList(cmd *cobra.Command, args []string, ops options.ImagesOption) error {
	// 判断args长度, 按照docker设计两个imageName时会报错
	if len(args) > 1 {
		return errors.New("\"docker images\" requires at most 1 argument")
	}
	store, err := utils2.GetStore(cmd)
	if err != nil {
		return err
	}
	imageManager, err := imagemanager.NewImageManager(store)
	if err != nil {
		return err
	}
	images, err := imageManager.ListImage(args,store)
	if ops.Json {
		return utils2.JsonFormatImages(images, ops)
	}
	return utils2.FormatImages(images, ops)
}

func ImageListCmd() *cobra.Command {
	var op options.ImagesOption
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List images",
		RunE: func(cmd *cobra.Command, args []string) error {
			return imageList(cmd, args, op)
		},
		Example: `ktib images images
ktib images images [imageName]
ktib images images --format '{{.ID}} {{.Name}} {{.Size}}'`,
	}
	flag := cmd.Flags()
	flag.BoolVarP(&op.Quiet, "quiet", "q", false, "Only show numeric IDs")
	flag.BoolVar(&op.Digests, "digests", false, "show info include digests")
	flag.BoolVar(&op.Truncate, "no-trunc", false, "show info include images IDs")
	flag.BoolVar(&op.Json, "json", false, "output in JSON format")
	return cmd
}
