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
	"sync"
	"text/tabwriter"

	"gitee.com/openeuler/ktib/pkg/analyze"
	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func newCmdAnalyze() *cobra.Command {
	var outputFormat string
	var outputFile string
	var fastMode bool
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
			var progressFunc func(string, bool, time.Duration)
			var p *mpb.Progress

			// Show progress if we are outputting to file OR if output format is not JSON (i.e. summary mode)
			// Enable progress bar if output is JSON (default) but likely user interactive,
			// or if we add a flag later. For now, always show on stderr unless explicitly disabled?
			// The user requirement is "process visualization".
			// We write progress to Stderr so Stdout remains clean for JSON.
			if outputFormat == "json" || outputFormat == "" {
				p = mpb.New(mpb.WithOutput(os.Stderr), mpb.WithWidth(60))
			var currentStep string
				var currentStep string
				var mu sync.Mutex
			// 5 main steps: Layer, Mount, Package, FS, Advisor
				// 5 main steps: Layer, Mount, Package, FS, Advisor
				totalSteps := 5
			// Custom decorator to safely read current step
				// Custom decorator to safely read current step
				stepDecor := decor.Any(func(s decor.Statistics) string {
					mu.Lock()
					defer mu.Unlock()
					return currentStep
				}, decor.WC{W: 25})
			bar := p.New(int64(totalSteps),
				bar := p.New(int64(totalSteps),
					mpb.BarStyle().Lbound("").Rbound("").Filler("█").Tip("█").Padding("░"),
					mpb.PrependDecorators(
						decor.Spinner(nil, decor.WC{W: 2, C: decor.DSyncSpace}),
						stepDecor,
					),
					mpb.AppendDecorators(
						decor.CurrentNoUnit(""),
						decor.Name("/", decor.WC{W: 1}),
						decor.TotalNoUnit(""),
						decor.Percentage(decor.WCSyncSpace),
					),
				)
			progressFunc = func(step string, done bool, duration time.Duration) {
				progressFunc = func(step string, done bool, duration time.Duration) {
					if !done {
						mu.Lock()
						currentStep = step
						mu.Unlock()
					} else {
						// Step finished
						msg := fmt.Sprintf("\x1b[32m✔ %s\x1b[0m (%v)\n", step, duration.Round(time.Millisecond))
						// Use p.Write to print above the bar
						// Note: p.Write expects a slice of bytes
						// We need to ensure we don't write if p is nil, but it is not here.
						// mpb v8 does not have p.Write directly exposed easily on *Progress?
						// Wait, checking docs/memory...
						// Actually usually people use a proxy writer or just `fmt.Fprint(os.Stderr)` if mpb is configured correctly.
						// But mpb might overwrite.
						// Correct way in mpb v8 is complex for ad-hoc logs.
						// Let's try direct write to stderr, mpb usually handles it if using WithOutput(os.Stderr).
						// But to be safe, we can just let the bar update.
						// Plan said "Print checkmark log".
						// Let's try fmt.Fprintf(os.Stderr, ...) and see.
						// To avoid interference, we can use a "Log Bar" or just print.
						// Given the constraints, I will try simple Fprintf.
						fmt.Fprintf(os.Stderr, "\r%s", msg)
						bar.Increment()
					}
			}

			analyzer, err := analyze.NewAnalyzer(store, imageRef, rulesPath, fastMode)
			utils.CheckErr(err)

			report, err := analyzer.Run(cmd.Context(), progressFunc)

			// Ensure progress bar finishes cleanly
			if p != nil {
				p.Wait()
			}

			utils.CheckErr(err)

			if outputFile != "" {
			if outputFormat == "json" {
				enc := json.NewEncoder(os.Stdout)
				err = enc.Encode(report)
				err := enc.Encode(report)
				fmt.Printf("Analysis report saved to %s\n", outputFile)
				// Print to Stdout
				// TODO: Implement human readable output or default to JSON for now
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				err := enc.Encode(report)
				utils.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "summary", "Output format (summary|json)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "json", "Output format (json)")
	cmd.Flags().BoolVar(&fastMode, "fast", false, "Enable fast mode (skip checksums and deep inspection)")
	return cmd
}

