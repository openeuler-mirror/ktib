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
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// DependencyScanner handles the analysis of binary dependencies
type DependencyScanner struct {
	Rootfs string
	// Cache for resolved paths to avoid re-scanning
	resolvedCache map[string]string
	// Standard library paths
	libPaths []string
}

// NewDependencyScanner creates a new scanner instance
func NewDependencyScanner(rootfs string) *DependencyScanner {
	scanner := &DependencyScanner{
		Rootfs:        rootfs,
		resolvedCache: make(map[string]string),
		libPaths: []string{
			"/lib", "/usr/lib", "/lib64", "/usr/lib64",
			"/usr/local/lib", "/usr/local/lib64",
		},
	}
	// Try to load additional paths from ld.so.conf
	if err := scanner.loadLdSoConf(); err != nil {
		logrus.Debugf("Failed to load ld.so.conf: %v", err)
	}
	return scanner
}

// loadLdSoConf reads /etc/ld.so.conf and adds paths to libPaths
func (s *DependencyScanner) loadLdSoConf() error {
	confPath := filepath.Join(s.Rootfs, "etc", "ld.so.conf")
	return s.parseConfFile(confPath)
}

func (s *DependencyScanner) parseConfFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "include ") {
			pattern := strings.TrimSpace(strings.TrimPrefix(line, "include "))
			// pattern might be absolute in container, e.g. /etc/ld.so.conf.d/*.conf
			// we need to glob it relative to rootfs
			// But glob pattern itself usually contains absolute path
			// So we join rootfs with the pattern
			fullPattern := filepath.Join(s.Rootfs, pattern)
			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				continue
			}
			for _, match := range matches {
				s.parseConfFile(match)
			}
			continue
		}

		// It's a directory path
		// Ensure it's absolute (it should be)
		if !strings.HasPrefix(line, "/") {
			continue
		}
		s.libPaths = append(s.libPaths, line)
	}
	return scanner.Err()
}

// ScanDependencies finds all shared library dependencies for the given entrypoints
func (s *DependencyScanner) ScanDependencies(entrypoints []string) ([]string, error) {
	requiredLibs := make(map[string]struct{})
	queue := make([]string, 0)

	// Initialize queue with entrypoints
	for _, ep := range entrypoints {
		if ep == "" {
			continue
		}
		// Resolve entrypoint path relative to rootfs
		path := ep
		// If entrypoint is not absolute, we skip it for now or could try to find it in PATH
		if !strings.HasPrefix(path, "/") {
			logrus.Debugf("Entrypoint %s is not absolute, skipping dependency check", ep)
			continue
		}

		fullPath := filepath.Join(s.Rootfs, path)
		if _, err := os.Stat(fullPath); err == nil {
			queue = append(queue, fullPath)
		} else {
			logrus.Debugf("Entrypoint %s not found at %s", path, fullPath)
		}
	}

	processed := make(map[string]struct{})

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, ok := processed[current]; ok {
			continue
		}
		processed[current] = struct{}{}

		libs, err := s.findSharedLibraries(current)
		if err != nil {
			// Not an ELF or error reading, just continue
			continue
		}

		for _, lib := range libs {
			// lib is the name, e.g., "libc.so.6"
			// We need to resolve it to a full path
			resolvedPath, err := s.resolveLibrary(lib)
			if err != nil {
				continue
			}

			// Store the path inside the container (remove rootfs prefix)
			relPath, _ := filepath.Rel(s.Rootfs, resolvedPath)
			// Ensure it starts with /
			containerPath := filepath.Join("/", relPath)
			// Fix for windows path separator if running on windows (though rootfs is linux usually)
			containerPath = filepath.ToSlash(containerPath)

			requiredLibs[containerPath] = struct{}{}

			if _, ok := processed[resolvedPath]; !ok {
				queue = append(queue, resolvedPath)
			}
		}
	}

	result := make([]string, 0, len(requiredLibs))
	for lib := range requiredLibs {
		result = append(result, lib)
	}
	return result, nil
}

// findSharedLibraries extracts DT_NEEDED from an ELF file
func (s *DependencyScanner) findSharedLibraries(path string) ([]string, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	libs, err := f.ImportedLibraries()
	if err != nil {
		return nil, err
	}
	return libs, nil
}

// resolveLibrary finds the full path of a library name in standard paths
func (s *DependencyScanner) resolveLibrary(libName string) (string, error) {
	if path, ok := s.resolvedCache[libName]; ok {
		return path, nil
	}

	for _, prefix := range s.libPaths {
		candidate := filepath.Join(s.Rootfs, prefix, libName)
		if _, err := os.Stat(candidate); err == nil {
			s.resolvedCache[libName] = candidate
			return candidate, nil
		}
	}

	return "", fmt.Errorf("library %s not found", libName)
}

// AssessFatSlim calculates potential savings
// returns: totalLibsSize, requiredLibsSize, savings, unusedLibs
func (s *DependencyScanner) AssessFatSlim(requiredLibs []string) (int64, int64, int64, []string) {
	requiredSet := make(map[string]struct{})
	for _, l := range requiredLibs {
		requiredSet[l] = struct{}{}
	}

	var totalSize int64
	var requiredSize int64
	var unusedLibs []string

	// Helper to walk paths
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		
		// Check if it's a library (simple check: contains .so)
		if strings.Contains(info.Name(), ".so") {
			size := info.Size()
			totalSize += size

			// Check if it is required
			// path is absolute on host (including rootfs)
			// requiredLibs are absolute paths inside container (e.g. /lib64/libc.so.6)

			relPath, _ := filepath.Rel(s.Rootfs, path)
			containerPath := filepath.ToSlash(filepath.Join("/", relPath))

			if _, ok := requiredSet[containerPath]; ok {
				requiredSize += size
			} else {
				unusedLibs = append(unusedLibs, containerPath)
			}
		}
		return nil
	}

	for _, prefix := range s.libPaths {
		targetDir := filepath.Join(s.Rootfs, prefix)
		// Only walk if directory exists
		if _, err := os.Stat(targetDir); err == nil {
			filepath.Walk(targetDir, walkFn)
		}
	}

	return totalSize, requiredSize, totalSize - requiredSize, unusedLibs
}
