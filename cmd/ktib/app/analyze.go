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
	"gitee.com/openeuler/ktib/pkg/i18n"
	"gitee.com/openeuler/ktib/pkg/types"
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
	var saveData string
	var fromData string
	var lang string

	cmd := &cobra.Command{
		Use:   "analyze <image>",
		Short: "Analyze an image for bloat and packages",
		Long: `Analyze an image to find wasted space, installed packages, and provide optimization recommendations.

This command supports two modes:
1. Online Mode (default): Scans a local image, collects data, and generates recommendations.
2. Offline Mode: Uses pre-collected analysis data to generate recommendations without needing the image.

Key Features:
- Layer Analysis: Detailed breakdown of file changes per layer.
- Package Scan: Detection of RPM and Python packages with metadata.
- Waste Detection: Identification of duplicate files across layers.
- Advisor: Rule-based optimization recommendations.`,
		Example: ` # 1. Standard Analysis (Scan + Recommend)
  ktib analyze myimage:latest

 # 2. Save Analysis Report to File (JSON)
  ktib analyze myimage:latest --output json --file report.json

 # 3. Separated Workflow (Useful for CI/CD or offloading analysis)
  # Step A: Collect data only (skips recommendation generation)
  ktib analyze myimage:latest --save-data raw_data.json

  # Step B: Generate recommendations from data (offline, no image needed)
  ktib analyze --from-data raw_data.json

 # 4. Advanced Options
  # Use custom rules
  ktib analyze myimage:latest --rules my_rules.yaml

  # Only run safe checks
  ktib analyze myimage:latest --level SAFE

  # Fast mode (skip heavy checksums)
  ktib analyze myimage:latest --fast`,
		Args: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("default-rules") {
				return nil
			}
			if cmd.Flags().Changed("from-data") {
				if len(args) > 0 {
					return fmt.Errorf("does not accept image argument when --from-data is used")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Initialize i18n
			i18n.SetLanguage(lang)

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

			var levelList []string
			if levels != "" {
				levelList = strings.Split(levels, ",")
			}

			// --- 1. Offline Mode: Load Data ---
			if fromData != "" {
				data, err := os.ReadFile(fromData)
				utils.CheckErr(err)

				var report types.AnalysisReport
				err = json.Unmarshal(data, &report)
				utils.CheckErr(err)

				// Initialize Analyzer with nil store for offline mode
				analyzer, err := analyze.NewAnalyzer(nil, "", rulesPath, levelList, fastMode, lang)
				utils.CheckErr(err)

				// Run Advisor (Offline)
				recs := analyzer.GenerateRecommendations(
					report.Analysis.Layers,
					report.Analysis.Packages,
					report.Analysis.Filesystem,
					report.Analysis.WasteDetection,
					"",  // No mount point
					nil, // No entrypoints (skips dependency check)
				)
				report.Recommendations = recs

				// Prune report for cleaner output
				pruneReport(&report)

				// Output
				if outputFile != "" {
					file, err := os.Create(outputFile)
					utils.CheckErr(err)
					defer file.Close()

					enc := json.NewEncoder(file)
					enc.SetEscapeHTML(false)
					enc.SetIndent("", "  ")
					err = enc.Encode(report)
					utils.CheckErr(err)
					fmt.Printf("Report with recommendations saved to %s\n", outputFile)
				}

				if outputFormat == "json" {
					enc := json.NewEncoder(os.Stdout)
					enc.SetEscapeHTML(false)
					enc.SetIndent("", "  ")
					err = enc.Encode(report)
					utils.CheckErr(err)
				} else {
					analyze.PrintRecommendations(recs)
				}
				return
			}

			// --- 2. Online Mode ---
			store, err := utils.GetStore(cmd)
			utils.CheckErr(err)

			imageRef := args[0]
			// Validate image exists before starting progress bar to avoid hang
			if _, err := store.Image(imageRef); err != nil {
				utils.CheckErr(fmt.Errorf("failed to find image %s: %w", imageRef, err))
			}

			totalSteps := 5
			if saveData != "" {
				totalSteps = 4 // Skip advisor step
			}
			progressFunc, waitFunc := analyze.NewAnalysisProgressBar(totalSteps)

			analyzer, err := analyze.NewAnalyzer(store, imageRef, rulesPath, levelList, fastMode, lang)
			utils.CheckErr(err)

			report, mountPoint, entrypoints, cleanup, err := analyzer.Analyze(cmd.Context(), func(step string, done bool, duration time.Duration) {
				if progressFunc != nil {
					progressFunc(step, done, duration)
				}
			})
			if cleanup != nil {
				defer cleanup()
			}
			// Ensure progress bar finishes cleanly (for the analyze part)
			if waitFunc != nil && saveData != "" {
				waitFunc()
			}

			utils.CheckErr(err)

			// Handle --save-data
			if saveData != "" {
				file, err := os.Create(saveData)
				utils.CheckErr(err)
				defer file.Close()

				enc := json.NewEncoder(file)
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				err = enc.Encode(report)
				utils.CheckErr(err)
				fmt.Printf("Analysis data (full) saved to %s\n", saveData)
				// Do not return here, continue to generate recommendations
			}

			// --- 3. Advisor (Standard Flow) ---
			stepName := "Advisor Generation"
			startTime := time.Now()
			if progressFunc != nil {
				progressFunc(stepName, false, 0)
			}

			recs := analyzer.GenerateRecommendations(
				report.Analysis.Layers,
				report.Analysis.Packages,
				report.Analysis.Filesystem,
				report.Analysis.WasteDetection,
				mountPoint,
				entrypoints,
			)
			report.Recommendations = recs

			if progressFunc != nil {
				progressFunc(stepName, true, time.Since(startTime))
			}

			// Ensure progress bar finishes cleanly
			if waitFunc != nil {
				waitFunc()
			}

			// Prune report for cleaner output
			pruneReport(report)

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
	cmd.Flags().StringVarP(&outputFile, "file", "f", "", "Save report to file (e.g. report.json)")
	cmd.Flags().BoolVar(&fastMode, "fast", false, "Enable fast mode (skip checksums and deep inspection)")
	cmd.Flags().StringVar(&rulesPath, "rules", "", "Path to custom rules file")
	cmd.Flags().Lookup("rules").NoOptDefVal = "/etc/ktib/default_rules.yaml"
	cmd.Flags().StringVar(&levels, "level", "", "Override run levels (comma separated, e.g. SAFE,STANDARD)")
	cmd.Flags().BoolVar(&defaultRules, "default-rules", false, "Dump default rules to default_rules.yaml")
	cmd.Flags().StringVar(&saveData, "save-data", "", "Save analysis data to JSON file (skips advisor)")
	cmd.Flags().StringVar(&fromData, "from-data", "", "Load analysis data from JSON file to generate recommendations (skips image scan)")
	cmd.Flags().StringVar(&lang, "lang", "en", "Output language (en|zh)")

	return cmd
}

func pruneReport(report *types.AnalysisReport) {
	for i := range report.Analysis.Packages.RPM {
		report.Analysis.Packages.RPM[i].Requires = nil
		report.Analysis.Packages.RPM[i].Provides = nil
		report.Analysis.Packages.RPM[i].Files = nil
	}
	for i := range report.Analysis.Packages.Python {
		report.Analysis.Packages.Python[i].Requires = nil
		report.Analysis.Packages.Python[i].Provides = nil
		report.Analysis.Packages.Python[i].Files = nil
	}
}
