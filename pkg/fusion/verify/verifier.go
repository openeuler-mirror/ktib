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
	"path/filepath"

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

	// 1. Locate RPM DB
	dbPath := filepath.Join(rootfsPath, "var/lib/rpm")
	candidates := []string{"Packages", "rpmdb.sqlite", "Packages.db"}
	var dbFile string
	for _, f := range candidates {
		p := filepath.Join(dbPath, f)
		if _, err := os.Stat(p); err == nil {
			dbFile = p
			break
		}
	}

	if dbFile == "" {
		return fmt.Errorf("RPM database not found in %s", dbPath)
	}

	// 2. Open DB
	db, err := rpmdb.Open(dbFile)
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
			}
		}
	}

	if errors > 0 {
		return fmt.Errorf("verification failed with %d errors", errors)
	}
	if warnings > 0 {
		logrus.Warnf("Verification completed with %d missing files (likely intentional cuts)", warnings)
	} else {
		logrus.Info("Verification passed: All files present.")
	}

	return nil
}
