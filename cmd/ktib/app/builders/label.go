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
	"fmt"
	"gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"gitee.com/openeuler/ktib/cmd/ktib/app/utils"
	"github.com/containers/buildah"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"strings"
)

func LABELCmd() *cobra.Command {
	var op options.IFIOptions
	cmd := &cobra.Command{
		Use:   "label",
		Args:  cobra.MinimumNArgs(2),
		Short: "Executes a command as described by a container image label.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return label(cmd, args, op)
		},
	}
	return cmd
}

func label(cmd *cobra.Command, args []string, op options.IFIOptions) error {
	op.ImportFromImageOptions.Image = args[0]
	var labels = []string{}
	makeLabels := make(map[string]string)
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	var ctx context.Context
	builder, err := buildah.ImportBuilderFromImage(ctx, store, op.ImportFromImageOptions)
	if err != nil {
		return err
	}
	for i := 1; i < len(args); i++ {
		keyValue := strings.SplitN(args[i], "=", 2)
		makeLabels[keyValue[0]] = ""
		if len(keyValue) > 1 {
			makeLabels[keyValue[0]] = keyValue[1]
		}
	}
	for k, v := range makeLabels {
		labels = append(labels, fmt.Sprintf(" %q=%q", k, v))
		builder.SetLabel(k, v)
	}
	return nil
}
