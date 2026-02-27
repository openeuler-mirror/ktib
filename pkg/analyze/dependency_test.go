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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveLibrary(t *testing.T) {
	tmpDir := t.TempDir()
	scanner := NewDependencyScanner(tmpDir)

	// Create dummy lib
	libDir := filepath.Join(tmpDir, "lib64")
	err := os.MkdirAll(libDir, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(libDir, "libc.so.6"), []byte("data"), 0644)
	assert.NoError(t, err)

	// Test finding it
	path, err := scanner.resolveLibrary("libc.so.6")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(libDir, "libc.so.6"), path)

	// Test missing
	_, err = scanner.resolveLibrary("missing.so")
	assert.Error(t, err)
}

func TestAssessFatSlim(t *testing.T) {
	tmpDir := t.TempDir()
	scanner := NewDependencyScanner(tmpDir)

	libDir := filepath.Join(tmpDir, "lib64")
	os.MkdirAll(libDir, 0755)

	// Create 3 libs
	// lib1: 100 bytes (required)
	// lib2: 200 bytes (not required)
	// lib3: 300 bytes (required)
	
	createLib := func(name string, size int) {
		data := make([]byte, size)
		os.WriteFile(filepath.Join(libDir, name), data, 0644)
	}

	createLib("lib1.so", 100)
	createLib("lib2.so", 200)
	createLib("lib3.so", 300)
	createLib("notalib.txt", 50)

	required := []string{
		"/lib64/lib1.so",
		"/lib64/lib3.so",
	}

	total, reqSize, saving, unused := scanner.AssessFatSlim(required)

	// Total libs size = 100 + 200 + 300 = 600 (notalib.txt ignored)
	// Required size = 100 + 300 = 400
	// Saving = 200

	assert.Equal(t, int64(600), total)
	assert.Equal(t, int64(400), reqSize)
	assert.Equal(t, int64(200), saving)
	assert.Len(t, unused, 1)
	// Check if unused contains the expected lib. 
	// Note: The path separator might need handling if strictly testing cross-platform behavior of the test itself, 
	// but AssessFatSlim normalizes to forward slashes.
	assert.Contains(t, unused, "/lib64/lib2.so")
}

func TestLoadLdSoConf(t *testing.T) {
	tmpDir := t.TempDir()

	// Create /etc/ld.so.conf
	etcDir := filepath.Join(tmpDir, "etc")
	err := os.MkdirAll(etcDir, 0755)
	assert.NoError(t, err)

	confContent := `
/usr/local/custom/lib
include /etc/ld.so.conf.d/*.conf
# This is a comment
`
	err = os.WriteFile(filepath.Join(etcDir, "ld.so.conf"), []byte(confContent), 0644)
	assert.NoError(t, err)

	// Create /etc/ld.so.conf.d/
	confD := filepath.Join(etcDir, "ld.so.conf.d")
	err = os.MkdirAll(confD, 0755)
	assert.NoError(t, err)

	// Create a conf file in conf.d
	extraConf := `/opt/lib`
	err = os.WriteFile(filepath.Join(confD, "extra.conf"), []byte(extraConf), 0644)
	assert.NoError(t, err)

	// Initialize scanner which should auto-load the conf
	scanner := NewDependencyScanner(tmpDir)

	// Check if paths are present
	assert.Contains(t, scanner.libPaths, "/usr/local/custom/lib")
	assert.Contains(t, scanner.libPaths, "/opt/lib")

	// Check standard paths are still there
	assert.Contains(t, scanner.libPaths, "/lib64")
}

func TestFindAllELFs(t *testing.T) {
	tmpDir := t.TempDir()
	scanner := NewDependencyScanner(tmpDir)

	binDir := filepath.Join(tmpDir, "bin")
	err := os.MkdirAll(binDir, 0755)
	assert.NoError(t, err)

	// Create fake ELF
	fakeELF := filepath.Join(binDir, "fakeapp")
	// ELF...
	header := []byte{0x7f, 0x45, 0x4c, 0x46}
	err = os.WriteFile(fakeELF, header, 0755)
	assert.NoError(t, err)

	// Create non-ELF
	err = os.WriteFile(filepath.Join(binDir, "script.sh"), []byte("#!/bin/sh"), 0755)
	assert.NoError(t, err)

	elfs, err := scanner.FindAllELFs()
	assert.NoError(t, err)
	assert.Contains(t, elfs, "/bin/fakeapp")
	assert.NotContains(t, elfs, "/bin/script.sh")
}
