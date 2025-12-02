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
    "gitee.com/openeuler/ktib/pkg/builder"
    "gitee.com/openeuler/ktib/pkg/options"
    "gitee.com/openeuler/ktib/pkg/utils"
    "github.com/spf13/cobra"
    "strings"
)

func LABELCmd() *cobra.Command {
	var op options.IFIOptions
	cmd := &cobra.Command{
		Use:   "label",
		Args:  cobra.MinimumNArgs(2),
		Short: "Adding labels to builders",
		Long: `The 'label' command sets labels on the builder. The first parameter is the builder ID or name,
The second parameter is an equal sign separated list of key value pairs, representing the labels to be set.

Example:
  #Set a single label on the builder
  ktib builders label builderID/builderName app=myapp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return label(cmd, args, op)
		},
	}
	return cmd
}

func label(cmd *cobra.Command, args []string, op options.IFIOptions) error {
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
    builderobj, err := builder.FindBuilder(store, args[0])
    if err != nil {
        return fmt.Errorf("Not found the %s builder", args[0])
    }
	containerId := args[0]
	// 将 args[1] 解析为 map[string]string
	labels, err := parseLabels(args[1])
	if err != nil {
		return err
	}

	op.Labels = labels
	return builderobj.SetLabel(containerId, op.Labels)
}

func parseLabels(input string) (map[string]string, error) {
	labels := make(map[string]string)

	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid label format: %s", pair)
		}
		labels[kv[0]] = kv[1]
	}

	return labels, nil
}
