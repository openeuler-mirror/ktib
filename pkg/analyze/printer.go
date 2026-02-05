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

package analyze

import (
	"fmt"
	"os"
	"text/tabwriter"

	"gitee.com/openeuler/ktib/pkg/types"
	"gitee.com/openeuler/ktib/pkg/utils"
)

// PrintSummary prints the analysis report summary to stdout
func PrintSummary(report *types.AnalysisReport) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "IMAGE ANALYSIS SUMMARY")
	fmt.Fprintln(w, "======================")
	fmt.Fprintf(w, "Image Ref:\t%s\n", report.ImageInfo.Ref)
	fmt.Fprintf(w, "Architecture:\t%s\n", report.ImageInfo.Architecture)
	fmt.Fprintf(w, "OS:\t%s\n", report.ImageInfo.OS)
	fmt.Fprintf(w, "Total Size:\t%s\n", utils.FormatBytes(report.ImageInfo.Size))
	fmt.Fprintln(w, "\t")

	fmt.Fprintln(w, "CONTENT STATS")
	fmt.Fprintln(w, "-------------")
	fmt.Fprintf(w, "Layers:\t%d\n", len(report.Analysis.Layers))
	fmt.Fprintf(w, "RPM Packages:\t%d\n", len(report.Analysis.Packages.RPM))
	fmt.Fprintf(w, "Python Packages:\t%d\n", len(report.Analysis.Packages.Python))

	// Calculate waste
	var wastedBytes int64
	// Duplicates
	for _, d := range report.Analysis.WasteDetection.Duplicates {
		// If a file is in N layers, N-1 copies are waste
		count := len(d.LayerDigest)
		if count > 1 {
			wastedBytes += d.Size * int64(count-1)
		}
	}
	// Caches
	for _, c := range report.Analysis.WasteDetection.Caches {
		wastedBytes += c.Size
	}

	fmt.Fprintf(w, "Potential Waste:\t%s\n", utils.FormatBytes(wastedBytes))
	efficiency := 100.0
	if report.ImageInfo.Size > 0 {
		efficiency = 100.0 * float64(report.ImageInfo.Size-wastedBytes) / float64(report.ImageInfo.Size)
	}
	fmt.Fprintf(w, "Image Efficiency:\t%.2f%%\n", efficiency)
	fmt.Fprintln(w, "\t")

	if len(report.Recommendations) > 0 {
		fmt.Fprintln(w, "RECOMMENDATIONS")
		fmt.Fprintln(w, "---------------")
		for _, r := range report.Recommendations {
			fmt.Fprintf(w, "[%s] %s: %s (Save: %s)\n", r.Level, r.Code, r.Message, r.Saving)
		}
	}

	w.Flush()
}
