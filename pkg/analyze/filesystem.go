/*
   Copyright (c) 2026 KylinSoft Co., Ltd.
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
	"context"
	"debug/elf"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/sirupsen/logrus"
)

func (a *Analyzer) AnalyzeFilesystem(ctx context.Context, rootfs string) (types.FilesystemInfo, string, error) {
	dirSizes := make(map[string]int64)
	fileTypeStats := make(map[string]*types.FileTypeInfo)
	archStats := make(map[string]int)

	err := filepath.Walk(rootfs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		// Calculate Directory Sizes (Aggregate up to rootfs)
		size := info.Size()
		dir := filepath.Dir(path)

		// Relativize path to rootfs for reporting
		relPath, _ := filepath.Rel(rootfs, dir)
		if relPath == "." {
			relPath = "/"
		} else {
			relPath = "/" + filepath.ToSlash(relPath)
		}

		// Add size to this directory and all parents up to root
		// But "Top Directories" usually means leaf or specific high-level dirs?
		// "Aggregate calculate size of each level directory"
		// Simple approach: Add to direct parent. Post-process to sum up?
		// Or Add to ALL parents.
		// Let's add to the direct parent key in map for now, using the relative path.

		// For the report "Top Directories", we usually want to see things like "/usr/lib", "/var/cache".
		// So we need to accumulate size to `relPath` and all its parents.

		currentDir := relPath
		for {
			dirSizes[currentDir] += size
			if currentDir == "/" || currentDir == "." {
				break
			}
			currentDir = filepath.Dir(currentDir)
			// filepath.Dir("/") returns "/"
			// filepath.Dir("/usr") returns "/"
		}

		// File Type Identification
		ftype := identifyFileType(path, info, a.Fast)
		if ftype == "ELF Binary" {
			if arch, err := getELFArch(path); err == nil {
				archStats[arch]++
			}
		}

		if _, ok := fileTypeStats[ftype]; !ok {
			fileTypeStats[ftype] = &types.FileTypeInfo{Type: ftype}
		}
		fileTypeStats[ftype].Count++
		fileTypeStats[ftype].Size += size

		return nil
	})

	if err != nil {
		logrus.Errorf("Filesystem walk failed: %v", err)
		return types.FilesystemInfo{}, "", err
	}

	// Process Top Directories
	var topDirs []types.TopDirectory
	for path, size := range dirSizes {
		topDirs = append(topDirs, types.TopDirectory{Path: path, Size: size})
	}
	sort.Slice(topDirs, func(i, j int) bool {
		return topDirs[i].Size > topDirs[j].Size
	})
	if len(topDirs) > 20 {
		topDirs = topDirs[:20]
	}

	// Process File Types
	var fileTypes []types.FileTypeInfo
	for _, stat := range fileTypeStats {
		fileTypes = append(fileTypes, *stat)
	}
	sort.Slice(fileTypes, func(i, j int) bool {
		return fileTypes[i].Size > fileTypes[j].Size
	})

	// Determine primary architecture
	primaryArch := "unknown"
	maxCount := 0
	for arch, count := range archStats {
		if count > maxCount {
			maxCount = count
			primaryArch = arch
		}
	}
	// Append architecture stats to file types or handle separately?
	// The current requirement asked for Architecture in ImageInfo (we added it to Schema).
	// But we return FilesystemInfo here.
	// Let's attach it to FilesystemInfo via a hack or change signature?
	// Changing signature is cleaner.

	return types.FilesystemInfo{
		TopDirectories: topDirs,
		FileTypes:      fileTypes,
	}, primaryArch, nil
}

func getELFArch(path string) (string, error) {
	f, err := elf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return f.Machine.String(), nil
}

func identifyFileType(path string, info os.FileInfo, fast bool) string {
	if info.Mode()&os.ModeSymlink != 0 {
		return "Symlink"
	}
	
	// Prevent blocking on special files (FIFO, Device, etc.)
	if !info.Mode().IsRegular() {
		return "Special File"
	}

	ext := strings.ToLower(filepath.Ext(path))

	// Fast mode: Trust extensions for common types
	if fast {
		switch ext {
		case ".jar":
			return "Java Jar"
		case ".whl":
			return "Python Wheel"
		case ".py", ".pyc":
			return "Python Source/Bytecode"
		case ".js":
			return "JavaScript"
		case ".go":
			return "Go Source"
		case ".c", ".h", ".cpp":
			return "C/C++ Source"
		case ".json", ".yaml", ".yml", ".xml":
			return "Config/Data"
		case ".sh", ".bash":
			return "Script"
		case ".so":
			return "ELF Binary" // Approximation
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return "Unknown"
	}
	defer f.Close()

	// Read magic bytes
	header := make([]byte, 262)
	n, _ := f.Read(header)

	if n < 4 {
		return "Empty/Small"
	}

	// Check Magic Numbers
	if header[0] == 0x7f && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
		return "ELF Binary"
	}
	if header[0] == '#' && header[1] == '!' {
		return "Script"
	}
	if header[0] == 0x1f && header[1] == 0x8b {
		return "Gzip Archive"
	}
	if header[0] == 0x50 && header[1] == 0x4b && header[2] == 0x03 && header[3] == 0x04 {
		// Could be Zip, Jar, Wheel
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jar" {
			return "Java Jar"
		}
		if ext == ".whl" {
			return "Python Wheel"
		}
		return "Zip Archive"
	}

	// Extension based fallback
	// ext is already calculated above
	switch ext {
	case ".py", ".pyc":
		return "Python Source/Bytecode"
	case ".js":
		return "JavaScript"
	case ".go":
		return "Go Source"
	case ".c", ".h", ".cpp":
		return "C/C++ Source"
	case ".json", ".yaml", ".yml", ".xml":
		return "Config/Data"
	}

	// Check if text
	if isText(header[:n]) {
		return "Text"
	}

	return "Binary Data"
}

func isText(b []byte) bool {
	// Simple heuristic: if it contains valid UTF-8 and no null bytes (except maybe at end)
	// and control characters are rare (except \n, \r, \t)
	for i, w := 0, 0; i < len(b); i += w {
		runeValue, width := utf8.DecodeRune(b[i:])
		if runeValue == utf8.RuneError {
			return false
		}
		w = width
		if runeValue == 0 {
			return false
		}
		if runeValue < 32 && runeValue != '\n' && runeValue != '\r' && runeValue != '\t' {
			return false
		}
	}
	return true
}
