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
	"encoding/json"
	"fmt"

	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	utils2 "gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func ImageInspectCmd() *cobra.Command {
	var op options.ImagesOption
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect images",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return imageInspect(cmd, args, op)
		},
		Example: `ktib images inspect [imageName]`,
	}
	return cmd
}
func imageInspect(cmd *cobra.Command, args []string, op options.ImagesOption) error {
	store, err := utils2.GetStore(cmd)
	if err != nil {
		return err
	}
	imageManager, err := imagemanager.NewImageManager(store)
	if err != nil {
		return err
	}

	// 调用 Inspect 方法获取镜像数据
	imageData, err := imageManager.Inspect(context.Background(), args[0])
	if err != nil {
		return err
	}

	// 将镜像数据转换为 JSON 并格式化输出
	jsonData, err := json.MarshalIndent(imageData, "", "    ")
	if err != nil {
		return err
	}

	// 输出格式化后的 JSON 数据
	fmt.Println(string(jsonData))
	return nil
}
