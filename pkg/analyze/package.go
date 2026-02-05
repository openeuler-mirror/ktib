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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/ktib/pkg/types"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/sirupsen/logrus"
)

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
	basePath := filepath.Join(rootfs, "var/lib/rpm")
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil, nil // No RPM db
	}

	// Try to find the actual database file
	// Common names: Packages (BDB), rpmdb.sqlite (SQLite), Packages.db (SQLite in some distros)
	candidates := []string{"Packages", "rpmdb.sqlite", "Packages.db"}
	var dbPath string
	for _, f := range candidates {
		p := filepath.Join(basePath, f)
		if _, err := os.Stat(p); err == nil {
			dbPath = p
			break
		}
	}

	if dbPath == "" {
		// If no known DB file is found, but the directory exists, we assume no packages or unknown format.
		// Returning nil is safer than failing hard.
		return nil, nil
	}

	db, err := rpmdb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rpmdb: %w", err)
	}
	defer db.Close()

	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var pkgs []types.Package
	for _, p := range pkgList {
		var filePaths []string
		files, err := p.InstalledFiles()
		if err == nil {
			for _, f := range files {
				filePaths = append(filePaths, f.Path)
			}
		} else {
			logrus.Debugf("Failed to get files for package %s: %v", p.Name, err)
		}

		pkgs = append(pkgs, types.Package{
			Name:     p.Name,
			Version:  fmt.Sprintf("%s-%s", p.Version, p.Release),
			Size:     int64(p.Size),
			License:  p.License,
			Digest:   p.SigMD5,
			Requires: p.Requires,
			Provides: p.Provides,
			Files:    filePaths,
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
				pkg := parsePythonMetadata(rootfs, path)
				if pkg.Name != "" {
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

func parsePythonMetadata(rootfs, path string) types.Package {
	pkg := types.Package{}

	// Priority: METADATA > PKG-INFO
	metaPath := filepath.Join(path, "METADATA")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		metaPath = filepath.Join(path, "PKG-INFO")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			return pkg
		}
	}

	// Calculate hash of the metadata file
	if hash, err := calculateFileHash(metaPath); err == nil {
		pkg.Digest = hash
	} else {
		logrus.Debugf("Failed to calculate hash for %s: %v", metaPath, err)
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

	// Parse Files
	pkg.Files = parsePythonFiles(rootfs, path, pkg.Name)

	return pkg
}

func parsePythonFiles(rootfs, infoPath, pkgName string) []string {
	var files []string
	baseDir := filepath.Dir(infoPath) // site-packages dir

	// 1. Try RECORD (.dist-info)
	recordPath := filepath.Join(infoPath, "RECORD")
	if _, err := os.Stat(recordPath); err == nil {
		if recFiles, err := parseRecordFile(rootfs, baseDir, recordPath); err == nil {
			return recFiles
		}
	}

	// 2. Try installed-files.txt (.egg-info)
	installedPath := filepath.Join(infoPath, "installed-files.txt")
	if _, err := os.Stat(installedPath); err == nil {
		if instFiles, err := parseInstalledFiles(rootfs, baseDir, installedPath); err == nil {
			return instFiles
		}
	}

	// 3. Try SOURCES.txt (.egg-info)
	sourcesPath := filepath.Join(infoPath, "SOURCES.txt")
	if _, err := os.Stat(sourcesPath); err == nil {
		if srcFiles, err := parseInstalledFiles(rootfs, baseDir, sourcesPath); err == nil {
			return srcFiles
		}
	}

	// 4. Fallback: top_level.txt or heuristics
	topLevelPath := filepath.Join(infoPath, "top_level.txt")
	if _, err := os.Stat(topLevelPath); err == nil {
		if topFiles, err := parseTopLevel(rootfs, baseDir, topLevelPath); err == nil {
			return topFiles
		}
	}

	// 5. Fallback: Package Name Directory
	// Convert package name to possible directory name (e.g. llama_cpp -> llama_cpp)
	// normalize: - to _
	normName := strings.ReplaceAll(pkgName, "-", "_")
	pkgDir := filepath.Join(baseDir, normName)
	if _, err := os.Stat(pkgDir); err == nil {
		if dirFiles, err := scanDirectory(rootfs, pkgDir); err == nil {
			return dirFiles
		}
	}

	return files
}

func parseRecordFile(rootfs, baseDir, recordPath string) ([]string, error) {
	f, err := os.Open(recordPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var files []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// RECORD format: path,sha256,size
		parts := strings.Split(line, ",")
		if len(parts) > 0 {
			relPath := parts[0]
			// Handle absolute paths (rare in wheels, but possible)
			if filepath.IsAbs(relPath) {
				// If absolute, it usually refers to system root, but we are in a chroot context?
				// Wheel spec says paths are relative to site-packages usually.
				// But some entries might be absolute.
				// Let's assume relative to baseDir if not absolute.
			}

			fullPath := filepath.Join(baseDir, relPath)
			
			// Resolve to container absolute path
			// rootfs might be /tmp/mount/
			// fullPath might be /tmp/mount/usr/lib/python...
			// We need /usr/lib/python...
			
			containerPath, err := filepath.Rel(rootfs, fullPath)
			if err == nil {
				if !strings.HasPrefix(containerPath, "/") {
					containerPath = "/" + containerPath
				}
				files = append(files, containerPath)
			}
		}
	}
	return files, nil
}

func parseInstalledFiles(rootfs, baseDir, path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var files []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		fullPath := filepath.Join(baseDir, line)
		containerPath, err := filepath.Rel(rootfs, fullPath)
		if err == nil {
			if !strings.HasPrefix(containerPath, "/") {
				containerPath = "/" + containerPath
			}
			files = append(files, containerPath)
		}
	}
	return files, nil
}

func parseTopLevel(rootfs, baseDir, path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var files []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		modName := strings.TrimSpace(scanner.Text())
		if modName == "" {
			continue
		}
		
		modDir := filepath.Join(baseDir, modName)
		modPy := filepath.Join(baseDir, modName+".py")
		
		if info, err := os.Stat(modDir); err == nil && info.IsDir() {
			if dirFiles, err := scanDirectory(rootfs, modDir); err == nil {
				files = append(files, dirFiles...)
			}
		}
		if _, err := os.Stat(modPy); err == nil {
			if containerPath, err := filepath.Rel(rootfs, modPy); err == nil {
				if !strings.HasPrefix(containerPath, "/") {
					containerPath = "/" + containerPath
				}
				files = append(files, containerPath)
			}
		}
	}
	return files, nil
}

func scanDirectory(rootfs, dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			containerPath, err := filepath.Rel(rootfs, path)
			if err == nil {
				if !strings.HasPrefix(containerPath, "/") {
					containerPath = "/" + containerPath
				}
				files = append(files, containerPath)
			}
		}
		return nil
	})
	return files, err
}

func calculateFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
