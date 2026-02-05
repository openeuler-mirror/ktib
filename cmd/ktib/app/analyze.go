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

package app

import (
	"encoding/json"
	"os"

	"gitee.com/openeuler/ktib/pkg/analyze"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func newCmdAnalyze() *cobra.Command {
	var outputFormat string
	cmd := &cobra.Command{
		Use:   "analyze <image>",
		Short: "Analyze an image for bloat and packages",
		Long:  `Analyze an image to find wasted space, installed packages, and provide optimization recommendations.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			store, err := utils.GetStore(cmd)
			utils.CheckErr(err)

			imageRef := args[0]
			analyzer := analyze.NewAnalyzer(store, imageRef)
			report, err := analyzer.Run(cmd.Context())
			utils.CheckErr(err)

			if outputFormat == "json" {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				err := enc.Encode(report)
				utils.CheckErr(err)
			} else {
				// TODO: Implement human readable output or default to JSON for now
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				err := enc.Encode(report)
				utils.CheckErr(err)
			}
		},
	}
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "json", "Output format (json)")
	return cmd
}
