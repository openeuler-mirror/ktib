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
	"gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"gitee.com/openeuler/ktib/cmd/ktib/app/utils"
	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
)

// TODO: 当dockerfile制作镜像时，超过一层会panic；commit已有镜像名也会panic
func imageList(c *cobra.Command, args []string, ops options.ImagesOption) error {
	// 判断args长度, 按照docker设计两个imageName时会报错
	if len(args) > 1 {
		return errors.New("\"docker images\" requires at most 1 argument")
	}
	store, err := utils.GetStore(c)
	if err != nil {
		return err
	}
	var systemContext *types.SystemContext
	//systemContext.BigFilesTemporaryDir = "/tmp"
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}

	// get current context
	ctx := context.Background()
	opts := &libimage.ListImagesOptions{}

	// TODO set opts.Filters = ops.Filter impl filter images

	images, err := runtime.ListImages(ctx, args, opts)
	if ops.Json {
		return utils.JsonFormatImages(images, ops)
	}

	return utils.FormatImages(images, ops)
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
