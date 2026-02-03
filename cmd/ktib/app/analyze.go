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
	"time"

	"strings"

	"gitee.com/openeuler/ktib/pkg/analyze"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func newCmdAnalyze() *cobra.Command {
	var outputFormat string
	var outputFile string
	var fastMode bool
	var rulesPath string
	var levels string
	var defaultRules bool

	cmd := &cobra.Command{
		Use:   "analyze <image>",
		Short: "Analyze an image for bloat and packages",
		Long:  `Analyze an image to find wasted space, installed packages, and provide optimization recommendations.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("default-rules") {
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if defaultRules {
				// Dump to default system path (e.g. /etc/ktib/default_rules.yaml)
				// If rulesPath is provided, we could use it, but flag says dump default rules.
				// We pass empty string to use default path defined in advisor.go
				if err := analyze.DumpDefaultRules(""); err != nil {
					// Fallback: try dumping to local directory if system path fails (e.g. permission denied)
					fmt.Fprintf(os.Stderr, "Failed to dump to default system path: %v. Trying current directory...\n", err)
					if err := analyze.DumpDefaultRules("default_rules.yaml"); err != nil {
						utils.CheckErr(err)
					}
					fmt.Println("Default rules dumped to default_rules.yaml")
				} else {
					fmt.Printf("Default rules dumped to %s\n", analyze.DefaultRulesPath)
				}
				return
			}

			store, err := utils.GetStore(cmd)
			utils.CheckErr(err)

			imageRef := args[0]
			// Validate image exists before starting progress bar to avoid hang
			if _, err := store.Image(imageRef); err != nil {
				utils.CheckErr(fmt.Errorf("failed to find image %s: %w", imageRef, err))
			}

			// Setup progress visualization
			// Only show progress bar if we are not outputting JSON to stdout (to avoid mixing output)
			// If output is summary (default) or we are writing to file, show progress.
			// If output is json and NO file is specified (meaning json goes to stdout), suppress progress or send to stderr.
			// The progress bar inside NewAnalysisProgressBar writes to stderr, so it's generally safe.
			// However, if we strictly want to suppress it for json stdout, we might want to control it.
			// But the original code just initialized it always to Stderr.
			// "p = mpb.New(mpb.WithOutput(os.Stderr), mpb.WithWidth(60))"
			// So we can just use the new function.

			totalSteps := 5
			progressFunc, waitFunc := analyze.NewAnalysisProgressBar(totalSteps)

			// If user wants JSON to stdout and no file, maybe we should not show progress?
			// The original code comment said:
			// "If output is json and NO file is specified (meaning json goes to stdout), suppress progress or send to stderr. We send to stderr so it's fine."
			// So it implies it's fine to show it on stderr even if json is on stdout.
			// We'll stick to that behavior.

			var levelList []string
			if levels != "" {
				levelList = strings.Split(levels, ",")
			}

			analyzer, err := analyze.NewAnalyzer(store, imageRef, rulesPath, levelList, fastMode)
			utils.CheckErr(err)

			report, err := analyzer.Run(cmd.Context(), func(step string, done bool, duration time.Duration) {
				if progressFunc != nil {
					progressFunc(step, done, duration)
				}
			})

			// Ensure progress bar finishes cleanly
			if waitFunc != nil {
				waitFunc()
			}

			utils.CheckErr(err)

			// 1. Handle File Output
			if outputFile != "" {
				file, err := os.Create(outputFile)
				utils.CheckErr(err)
				defer file.Close()

				enc := json.NewEncoder(file)
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				err = enc.Encode(report)
				utils.CheckErr(err)
				fmt.Printf("Analysis report saved to %s\n", outputFile)
			}

			// 2. Handle Stdout Output
			if outputFormat == "json" {
				// If user explicitly asked for JSON to stdout (and maybe also to file if they set both)
				enc := json.NewEncoder(os.Stdout)
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				err = enc.Encode(report)
				utils.CheckErr(err)
			} else {
				// Default to summary
				analyze.PrintSummary(report)
			}
		},
	}
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "summary", "Output format (summary|json)")
	cmd.Flags().StringVarP(&outputFile, "file", "f", "", "Output report to file")
	cmd.Flags().BoolVar(&fastMode, "fast", false, "Enable fast mode (skip checksums and deep inspection)")
	cmd.Flags().StringVar(&rulesPath, "rules", "", "Path to custom rules file")
	cmd.Flags().StringVar(&levels, "level", "", "Override run levels (comma separated, e.g. SAFE,STANDARD)")
	cmd.Flags().BoolVar(&defaultRules, "default-rules", false, "Dump default rules to default_rules.yaml")

	return cmd
}
