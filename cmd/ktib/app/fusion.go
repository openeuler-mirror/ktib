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
	"fmt"
	"strings"

	"gitee.com/openeuler/ktib/pkg/fusion"
	"gitee.com/openeuler/ktib/pkg/fusion/config"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func newCmdFusion() *cobra.Command {
	var configPath string
	var outputDir string
	var targetTag string

	cmd := &cobra.Command{
		Use:   "fusion <image>",
		Short: "Fusion and optimize container images",
		Long: `Fusion is a powerful tool to slim down container images by keeping only necessary dependencies.
It uses advanced dependency solving and RPM DB reconstruction to create a minimal, valid rootfs.

Example:
  ktib fusion myimage:latest --config fusion.yaml --output-dir ./output
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			imageRef := args[0]

			// 1. Load Config
			cfg, err := config.LoadConfig(configPath)
			utils.CheckErr(err)

			// Get Store
			store, err := utils.GetStore(cmd)
			utils.CheckErr(err)

			// 2. Initialize Manager
			mgr := fusion.NewFusionManager(cfg, store)

			// 3. Execute Fusion
			// If outputDir is not specified, use a default one?
			if outputDir == "" {
				outputDir = fmt.Sprintf("fusion_output_%s", strings.ReplaceAll(imageRef, ":", "_"))
			}

			err = mgr.Run(imageRef, outputDir)
			utils.CheckErr(err)

			if targetTag != "" {
				fmt.Printf("Building new image %s from %s is not yet implemented.\n", targetTag, outputDir)
			}
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to fusion configuration file")
	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Directory to output the fused rootfs")
	cmd.Flags().StringVarP(&targetTag, "tag", "t", "", "Tag for the new image (optional)")

	return cmd
}
