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

	"gitee.com/openeuler/ktib/pkg/types"
)

func (a *Analyzer) GenerateRecommendations(
	layers []types.LayerInfo,
	pkgs types.PackageInfo,
	fs types.FilesystemInfo,
	waste types.WasteDetection,
) []types.Recommendation {
	var recs []types.Recommendation

	// TODO: Implement configurable rules loading mechanism
	// For now, rules are temporarily empty as per requirement.
	/*
		// Rule 1: Package Manager Cache
		cacheDirs := []string{"/var/cache/yum", "/var/cache/apt", "/root/.cache/pip", "/var/cache/dnf"}
		for _, dir := range fs.TopDirectories {
			for _, target := range cacheDirs {
				if strings.HasPrefix(dir.Path, target) {
					recs = append(recs, types.Recommendation{
						Level:   "WARN",
						Code:    "RM_PKG_CACHE",
						Message: fmt.Sprintf("Found package manager cache at %s. Consider adding 'yum clean all' or equivalent.", dir.Path),
						Saving:  formatSize(dir.Size),
					})
				}
			}
		}

		// Rule 2: Documentation
		docDirs := []string{"/usr/share/doc", "/usr/share/man", "/usr/share/info"}
		totalDocSize := int64(0)
		for _, dir := range fs.TopDirectories {
			for _, target := range docDirs {
				if strings.HasPrefix(dir.Path, target) {
					totalDocSize += dir.Size
				}
			}
		}
		if totalDocSize > 10*1024*1024 { // > 10MB
			recs = append(recs, types.Recommendation{
				Level:   "INFO",
				Code:    "RM_DOCS",
				Message: "Documentation files found. Consider removing /usr/share/doc and /usr/share/man in production images.",
				Saving:  formatSize(totalDocSize),
			})
		}

		// Rule 3: Development Tools
		// Check for gcc, make, or *-devel packages
		hasDevTools := false
		for _, pkg := range pkgs.RPM {
			if strings.HasSuffix(pkg.Name, "-devel") || pkg.Name == "gcc" || pkg.Name == "make" {
				hasDevTools = true
				break
			}
		}
		if hasDevTools {
			recs = append(recs, types.Recommendation{
				Level:   "WARN",
				Code:    "RM_DEV_TOOLS",
				Message: "Development tools or headers (gcc, make, *-devel) found. Use multi-stage builds to exclude them from final image.",
				Saving:  "Variable",
			})
		}

		// Rule 4: Duplicates across layers
		if len(waste.Duplicates) > 0 {
			dupSize := int64(0)
			for _, d := range waste.Duplicates {
				dupSize += d.Size
			}
			recs = append(recs, types.Recommendation{
				Level:   "WARN",
				Code:    "MERGE_LAYERS",
				Message: fmt.Sprintf("Found %d duplicate files across layers (overwritten files). Consider merging layers.", len(waste.Duplicates)),
				Saving:  formatSize(dupSize),
			})
		}
	*/

	return recs
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
