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

	"gitee.com/openeuler/ktib/pkg/analyze"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func newCmdAnalyze() *cobra.Command {
	var outputFormat string
	var outputFile string
	var fastMode bool
	var rulesPath string

	cmd := &cobra.Command{
		Use:   "analyze <image>",
		Short: "Analyze an image for bloat and packages",
		Long:  `Analyze an image to find wasted space, installed packages, and provide optimization recommendations.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			store, err := utils.GetStore(cmd)
			utils.CheckErr(err)

			imageRef := args[0]

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

			analyzer, err := analyze.NewAnalyzer(store, imageRef, rulesPath, fastMode)
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
				enc.SetIndent("", "  ")
				err = enc.Encode(report)
				utils.CheckErr(err)
				fmt.Printf("Analysis report saved to %s\n", outputFile)
			}

			// 2. Handle Stdout Output
			if outputFormat == "json" {
				// If user explicitly asked for JSON to stdout (and maybe also to file if they set both)
				enc := json.NewEncoder(os.Stdout)
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
	// Hidden rules flag for now or just exposed? User didn't ask for it, but code needs it.
	// I'll leave it as internal var initialized to "" (default rules).
	// If I don't register it, it stays empty string. Perfect.
	return cmd
}
