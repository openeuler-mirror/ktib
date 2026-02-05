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
	"strings"
	"text/tabwriter"

	"gitee.com/openeuler/ktib/pkg/i18n"
	"gitee.com/openeuler/ktib/pkg/types"
	"gitee.com/openeuler/ktib/pkg/utils"
)

// PrintSummary prints the analysis report summary to stdout
func PrintSummary(report *types.AnalysisReport) {
	PrintAnalysisStats(report)
	PrintRecommendations(report.Recommendations)
	fmt.Println("\n" + i18n.T("Tip: Use '-o json' or '-f <file>' for detailed report."))
}

// PrintAnalysisStats prints the statistical part of the analysis report
func PrintAnalysisStats(report *types.AnalysisReport) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, i18n.T("IMAGE ANALYSIS SUMMARY"))
	fmt.Fprintln(w, "======================")
	fmt.Fprintf(w, "%s\t%s\n", i18n.T("Image Ref:"), report.ImageInfo.Ref)
	fmt.Fprintf(w, "%s\t%s\n", i18n.T("Architecture:"), report.ImageInfo.Architecture)
	fmt.Fprintf(w, "%s\t%s\n", i18n.T("OS:"), report.ImageInfo.OS)
	fmt.Fprintf(w, "%s\t%s\n", i18n.T("Total Size:"), utils.FormatBytes(report.ImageInfo.Size))
	fmt.Fprintln(w, "\t")

	fmt.Fprintln(w, i18n.T("CONTENT STATS"))
	fmt.Fprintln(w, "-------------")
	fmt.Fprintf(w, "%s\t%d\n", i18n.T("Layers:"), len(report.Analysis.Layers))
	fmt.Fprintf(w, "%s\t%d\n", i18n.T("RPM Packages:"), len(report.Analysis.Packages.RPM))
	fmt.Fprintf(w, "%s\t%d\n", i18n.T("Python Packages:"), len(report.Analysis.Packages.Python))

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

	fmt.Fprintf(w, "%s\t%s\n", i18n.T("Potential Waste:"), utils.FormatBytes(wastedBytes))
	efficiency := 100.0
	if report.ImageInfo.Size > 0 {
		efficiency = 100.0 * float64(report.ImageInfo.Size-wastedBytes) / float64(report.ImageInfo.Size)
	}
	fmt.Fprintf(w, "%s\t%.2f%%\n", i18n.T("Image Efficiency:"), efficiency)
	fmt.Fprintln(w, "\t")
	w.Flush()
}

// PrintRecommendations prints the recommendations part of the analysis report
func PrintRecommendations(recs []types.Recommendation) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, i18n.T("RECOMMENDATIONS"))
	fmt.Fprintln(w, "---------------")

	if len(recs) == 0 {
		fmt.Fprintln(w, i18n.T("No recommendations found for current level. Use '--level ALL' to see more potential optimizations."))
	} else {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", i18n.T("LEVEL"), i18n.T("ID"), i18n.T("SAVINGS"), i18n.T("DESCRIPTION"), i18n.T("COMMAND"))
		for _, r := range recs {
			msg := r.Message
			if len(r.MatchedItems) > 0 {
				limit := 3
				var displayItems []string
				if len(r.MatchedItems) > limit {
					displayItems = r.MatchedItems[:limit]
					displayItems = append(displayItems, fmt.Sprintf("... +%d more", len(r.MatchedItems)-limit))
				} else {
					displayItems = r.MatchedItems
				}
				msg = fmt.Sprintf("%s (%s)", msg, strings.Join(displayItems, ", "))
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.Level, r.Code, r.Saving, msg, r.Command)
		}
	}
	w.Flush()
}
