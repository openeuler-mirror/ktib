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
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/sirupsen/logrus"
)

var execCommand = exec.Command

func (a *Analyzer) AnalyzePackages(ctx context.Context, rootfs string) (types.PackageInfo, error) {
	info := types.PackageInfo{}

	// RPM Analysis
	rpms, err := scanRPMs(rootfs)
	if err != nil {
		logrus.Debugf("RPM scan failed (non-fatal): %v", err)
	} else {
		info.RPM = rpms
	}

	// Python Analysis
	pys, err := scanPython(rootfs)
	if err != nil {
		logrus.Debugf("Python scan failed (non-fatal): %v", err)
	} else {
		info.Python = pys
	}

	return info, nil
}

func scanRPMs(rootfs string) ([]types.Package, error) {
	dbPath := filepath.Join(rootfs, "var/lib/rpm")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil // No RPM db
	}

	absDbPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}

	// Use the host rpm command
	cmd := execCommand("rpm", "--dbpath", absDbPath, "-qa", "--qf", "%{NAME}|%{VERSION}|%{RELEASE}|%{SIZE}|%{LICENSE}\n")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rpm command failed: %w", err)
	}

	var pkgs []types.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		size := int64(0)
		fmt.Sscanf(parts[3], "%d", &size)

		pkgs = append(pkgs, types.Package{
			Name:    parts[0],
			Version: parts[1] + "-" + parts[2],
			Size:    size,
			License: parts[4],
		})
	}
	return pkgs, nil
}

func scanPython(rootfs string) ([]types.Package, error) {
	var pkgs []types.Package

	// Heuristic search paths for Python site-packages
	searchPaths := []string{
		filepath.Join(rootfs, "usr", "lib"),
		filepath.Join(rootfs, "usr", "local", "lib"),
	}

	for _, base := range searchPaths {
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		}
		
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			// Look for .dist-info or .egg-info directories
			if info.IsDir() && (strings.HasSuffix(info.Name(), ".dist-info") || strings.HasSuffix(info.Name(), ".egg-info")) {
				pkg := parsePythonMetadata(path)
				if pkg.Name != "" {
					// Try to estimate size (scan directory of package?)
					// For now, keep size 0 or try to find RECORD file to sum up sizes
					pkgs = append(pkgs, pkg)
				}
				return filepath.SkipDir // Don't look inside dist-info
			}
			return nil
		})
		if err != nil {
			logrus.Debugf("Error walking %s: %v", base, err)
		}
	}
	return pkgs, nil
}

func parsePythonMetadata(path string) types.Package {
	pkg := types.Package{}
	
	// Priority: METADATA > PKG-INFO
	metaPath := filepath.Join(path, "METADATA")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		metaPath = filepath.Join(path, "PKG-INFO")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			return pkg
		}
	}

	f, err := os.Open(metaPath)
	if err != nil {
		return pkg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Name: ") {
			pkg.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name: "))
		} else if strings.HasPrefix(line, "Version: ") {
			pkg.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version: "))
		} else if strings.HasPrefix(line, "License: ") {
			pkg.License = strings.TrimSpace(strings.TrimPrefix(line, "License: "))
		}
	}
	return pkg
}
