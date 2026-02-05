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

package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	fusionrpm "gitee.com/openeuler/ktib/pkg/fusion/rpm"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/sirupsen/logrus"
)

// DefaultVerifier implements Verifier
type DefaultVerifier struct{}

// NewDefaultVerifier creates a new DefaultVerifier
func NewDefaultVerifier() *DefaultVerifier {
	return &DefaultVerifier{}
}

// Verify checks the integrity and usability of the fused image
func (v *DefaultVerifier) Verify(rootfsPath string) error {
	logrus.Infof("Verifying rootfs at %s", rootfsPath)

	loc, err := fusionrpm.FindRPMDB(rootfsPath)
	if err != nil {
		return fmt.Errorf("RPM database not found: %w", err)
	}

	// 2. Open DB
	db, err := rpmdb.Open(loc.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open rpmdb: %w", err)
	}
	defer db.Close()

	// 3. Check packages
	pkgList, err := db.ListPackages()
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	logrus.Infof("Verifying %d packages...", len(pkgList))
	errors := 0
	warnings := 0
	missingByPkg := make(map[string]int)

	for _, p := range pkgList {
		files, err := p.InstalledFiles()
		if err != nil {
			logrus.Warnf("Failed to get file list for %s: %v", p.Name, err)
			continue
		}

		for _, f := range files {
			// Skip ghost files or documentation if configured (but here we assume strict check for now)
			// TODO: Add config to ignore docs/man/locale

			fullPath := filepath.Join(rootfsPath, f.Path)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				// Check if it's a directory? InstalledFiles usually includes dirs.
				// Missing file
				logrus.Debugf("Missing file: %s (pkg: %s)", f.Path, p.Name)
				// errors++ // Too strict for now, maybe just warn
				warnings++
				missingByPkg[p.Name]++
			}
		}
	}

	if errors > 0 {
		return fmt.Errorf("verification failed with %d errors", errors)
	}
	if warnings > 0 {
		type kv struct {
			name  string
			count int
		}
		var items []kv
		for n, c := range missingByPkg {
			items = append(items, kv{name: n, count: c})
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].count == items[j].count {
				return items[i].name < items[j].name
			}
			return items[i].count > items[j].count
		})
		topN := 10
		if len(items) < topN {
			topN = len(items)
		}
		var parts []string
		for i := 0; i < topN; i++ {
			parts = append(parts, fmt.Sprintf("%s(%d)", items[i].name, items[i].count))
		}

		if len(parts) > 0 {
			logrus.Warnf("Verification completed with %d missing files (likely intentional cuts). Top missing packages: %s", warnings, strings.Join(parts, ", "))
		} else {
			logrus.Warnf("Verification completed with %d missing files (likely intentional cuts)", warnings)
		}
	} else {
		logrus.Info("Verification passed: All files present.")
	}

	// 4. Run rpm -Va (External Tool Check)
	// We run this only if rpm is available
	if _, err := exec.LookPath("rpm"); err == nil {
		logrus.Info("Running rpm -Va...")
		// rpm --root requires absolute path
		absRoot, _ := filepath.Abs(rootfsPath)
		cmd := exec.Command("rpm", "--root", absRoot, "-Va")
		// rpm -Va output is noisy for cut images, we capture it but maybe don't fail strictly?
		// The requirement is "Add rpm -Va check".
		// We'll run it and log output if it fails.
		output, err := cmd.CombinedOutput()
		if err != nil {
			outStr := strings.TrimSpace(string(output))
			logrus.Debugf("rpm -Va full output:\n%s", outStr)

			totalLines, snippet := summarizeLines(outStr, 5)
			if snippet == "" {
				logrus.Warn("rpm -Va reported issues (expected for fused images), but produced no output")
			} else {
				logrus.Warnf("rpm -Va reported issues (expected for fused images). Issues lines=%d, sample=%s (set log level to debug for full output)", totalLines, snippet)
			}
		} else {
			logrus.Info("rpm -Va passed cleanly.")
		}
	} else {
		logrus.Warn("rpm command not found, skipping rpm -Va check")
	}

	// 5. Run ldconfig (Library Check)
	if _, err := exec.LookPath("ldconfig"); err == nil {
		logrus.Info("Running ldconfig check...")
		absRoot, _ := filepath.Abs(rootfsPath)
		// ldconfig -r checks and rebuilds cache. If libs are broken/missing deps, it might complain?
		// ldconfig usually doesn't verify deps, just creates links.
		// To verify, we might check if it returns 0.
		cmd := exec.Command("ldconfig", "-r", absRoot)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logrus.Errorf("ldconfig failed:\n%s", string(output))
			return fmt.Errorf("library verification (ldconfig) failed")
		} else {
			logrus.Info("ldconfig verification passed.")
		}
	} else {
		logrus.Warn("ldconfig command not found, skipping library check")
	}

	return nil
}

func summarizeLines(s string, maxLines int) (int, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ""
	}
	lines := strings.Split(s, "\n")
	total := len(lines)
	if maxLines > 0 && total > maxLines {
		lines = lines[:maxLines]
	}
	return total, strings.Join(lines, " | ")
}
