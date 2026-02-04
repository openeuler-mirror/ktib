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
	"fmt"
	"os"
	"path/filepath"

	"gitee.com/openeuler/ktib/pkg/fusion"
	"gitee.com/openeuler/ktib/pkg/fusion/config"
	"gitee.com/openeuler/ktib/pkg/fusion/solver"
	"gitee.com/openeuler/ktib/pkg/i18n"
	"gitee.com/openeuler/ktib/pkg/types"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newCmdFusion() *cobra.Command {
	var configPath string
	var outputDir string
	var targetTag string
	var dumpConfig string
	var fromData string
	var saveData string
	var lang string

	cmd := &cobra.Command{
		Use:   "fusion <image>",
		Short: "Fusion and optimize container images",
		Long: `Fusion is a powerful tool to slim down container images by keeping only necessary dependencies.
It uses advanced dependency solving and RPM DB reconstruction to create a minimal, valid rootfs.

Example:
  ktib fusion --dump-config fusion.yaml
  ktib fusion myimage:latest --config fusion.yaml --tag myimage:slim
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("dump-config") {
				return nil
			}
			if cmd.Flags().Changed("from-data") {
				if len(args) > 1 {
					return fmt.Errorf("accepts at most one image argument when --from-data is used")
				}
				return nil
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Initialize i18n
			i18n.SetLanguage(lang)

			if cmd.Flags().Changed("dump-config") {
				cfg := config.NewExampleConfig()
				data, err := yaml.Marshal(cfg)
				utils.CheckErr(err)

				if dumpConfig == "-" {
					fmt.Print(string(data))
					return
				}

				if err := os.MkdirAll(filepath.Dir(dumpConfig), 0o755); err != nil {
					utils.CheckErr(err)
				}
				if err := os.WriteFile(dumpConfig, data, 0o644); err != nil {
					utils.CheckErr(err)
				}
				fmt.Printf("Default fusion config saved to %s\n", dumpConfig)
				return
			}

			imageRef := ""
			if len(args) > 0 {
				imageRef = args[0]
			} else if cmd.Flags().Changed("from-data") {
				ref, err := inferImageRefFromData(fromData)
				utils.CheckErr(err)
				imageRef = ref
			}
			if imageRef == "" {
				utils.CheckErr(fmt.Errorf("image reference is required (provide <image> or ensure --from-data contains image_info.ref)"))
			}

			// 1. Load Config
			cfg, err := config.LoadConfig(configPath)
			utils.CheckErr(err)

			// Get Store
			store, err := utils.GetStore(cmd)
			utils.CheckErr(err)

			// 2. Initialize Manager
			mgr := fusion.NewFusionManager(cfg, store)
			mgr.Lang = lang
			mgr.Solver = solver.NewDefaultSolverWithOptions(store, solver.Options{
				FromData: fromData,
				SaveData: saveData,
			})

			if targetTag == "" {
				utils.CheckErr(fmt.Errorf("--tag is required"))
			}

			totalSteps := 4
			if targetTag != "" {
				totalSteps++
			}
			progressFunc, waitFunc := fusion.NewFusionProgressBar(totalSteps)
			mgr.OnProgress = progressFunc

			keepOutput := cmd.Flags().Changed("output-dir") && outputDir != ""
			outputRootfs := outputDir
			tempOutput := ""
			if !keepOutput {
				tmpDir, err := os.MkdirTemp("", "ktib-fusion-output-")
				utils.CheckErr(err)
				tempOutput = tmpDir
				outputRootfs = tmpDir
			}

			err = mgr.Run(imageRef, outputRootfs, targetTag)
			waitFunc()
			if tempOutput != "" && err == nil {
				_ = os.RemoveAll(tempOutput)
			}
			if tempOutput != "" && err != nil {
				fmt.Fprintf(os.Stderr, "fusion failed; keeping temporary rootfs at %s\n", tempOutput)
			}
			utils.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to fusion configuration file")
	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Directory to output the fused rootfs (optional; if omitted, uses a temp dir and cleans up on success)")
	cmd.Flags().StringVarP(&targetTag, "tag", "t", "", "Tag for the new image (required)")
	cmd.Flags().StringVar(&dumpConfig, "dump-config", "", "Dump default fusion config to a file (use '-' for stdout)")
	cmd.Flags().Lookup("dump-config").NoOptDefVal = "fusion.yaml"
	cmd.Flags().StringVar(&saveData, "save-data", "", "Save analysis data to JSON file for reuse")
	cmd.Flags().StringVar(&fromData, "from-data", "", "Load analysis data from JSON file to skip image scan")
	cmd.Flags().StringVar(&lang, "lang", "en", "Output language (en|zh)")

	return cmd
}

func inferImageRefFromData(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read analysis data %s: %w", path, err)
	}
	var report types.AnalysisReport
	if err := json.Unmarshal(data, &report); err != nil {
		return "", fmt.Errorf("failed to parse analysis data %s: %w", path, err)
	}
	return report.ImageInfo.Ref, nil
}
