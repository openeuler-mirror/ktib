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

package rpm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gitee.com/openeuler/ktib/pkg/fusion/types"
	"github.com/sirupsen/logrus"
)

// DefaultReconstructor is an implementation of DBReconstructor that supports SQLite
type DefaultReconstructor struct {
	SourceRPMDBPath string // Path to the source RPM DB directory (e.g., /var/lib/rpm)
}

// NewDefaultReconstructor creates a new DefaultReconstructor
func NewDefaultReconstructor(sourcePath string) *DefaultReconstructor {
	return &DefaultReconstructor{
		SourceRPMDBPath: sourcePath,
	}
}

// Reconstruct builds a new RPM DB based on the kept packages
func (r *DefaultReconstructor) Reconstruct(sourcePath string, plan *types.FusionPlan, outputDir string) error {
	if sourcePath != "" {
		r.SourceRPMDBPath = sourcePath
	}
	logrus.Infof("Reconstructing RPM DB in %s for %d packages", outputDir, len(plan.KeptPackages))

	// 1. Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	// 2. Identify the database format and file
	// We prioritize SQLite as it is the standard for openEuler/Kylin
	candidates := []string{"rpmdb.sqlite", "Packages.db", "Packages"}
	var dbFile string
	for _, f := range candidates {
		p := filepath.Join(r.SourceRPMDBPath, f)
		if _, err := os.Stat(p); err == nil {
			dbFile = f
			break
		}
	}

	if dbFile == "" {
		return fmt.Errorf("no RPM database found in %s", r.SourceRPMDBPath)
	}

	sourceFilePath := filepath.Join(r.SourceRPMDBPath, dbFile)
	targetPath := filepath.Join(outputDir, dbFile)

	// 3. Handle by format
	if dbFile == "Packages" {
		// This is likely BerkeleyDB (BDB).
		// For now, we return an error as BDB is hard to manipulate in pure Go without CGO.
		return fmt.Errorf("BerkeleyDB (BDB) format is not supported for reconstruction yet. Please use SQLite-based images")
	}

	// Assume SQLite for other candidates
	logrus.Infof("Detected SQLite RPM DB at %s, copying to %s", sourceFilePath, targetPath)

	// 4. Copy the DB file to target
	if err := copyFile(sourceFilePath, targetPath); err != nil {
		return fmt.Errorf("failed to copy RPM DB: %w", err)
	}

	// 5. Prune the target DB
	if err := pruneSQLiteDB(targetPath, plan.KeptPackages); err != nil {
		return fmt.Errorf("failed to prune RPM DB: %w", err)
	}

	logrus.Infof("Successfully reconstructed RPM DB at %s", targetPath)
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
