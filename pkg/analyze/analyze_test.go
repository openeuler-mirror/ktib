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
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/storage"
	"github.com/stretchr/testify/assert"
)

// MockStore embeds storage.Store to satisfy the interface,
// but we only implement methods we need.
type MockStore struct {
	storage.Store
	images map[string]*storage.Image
	layers map[string]*storage.Layer
	diffs  map[string]io.ReadCloser
}

func (m *MockStore) Image(id string) (*storage.Image, error) {
	if img, ok := m.images[id]; ok {
		return img, nil
	}
	return nil, fmt.Errorf("image not found")
}

func (m *MockStore) Layer(id string) (*storage.Layer, error) {
	if l, ok := m.layers[id]; ok {
		return l, nil
	}
	return nil, fmt.Errorf("layer not found")
}

func (m *MockStore) Diff(from, to string, opts *storage.DiffOptions) (io.ReadCloser, error) {
	// analyze calls Diff("", layerID, ...)
	if rc, ok := m.diffs[to]; ok {
		// Return a new reader if needed, but for one-pass test it's fine
		// actually we should probably return a fresh reader or buffer
		// But here we store ReadCloser.
		return rc, nil
	}
	return nil, fmt.Errorf("diff not found")
}

func TestAnalyzeLayers(t *testing.T) {
	// Setup mock data
	layer1ID := "layer1"
	layer2ID := "layer2"
	imageID := "myimage"

	mockStore := &MockStore{
		images: map[string]*storage.Image{
			imageID: {
				ID:       imageID,
				TopLayer: layer2ID,
			},
		},
		layers: map[string]*storage.Layer{
			layer2ID: {ID: layer2ID, Parent: layer1ID},
			layer1ID: {ID: layer1ID, Parent: ""},
		},
		diffs: make(map[string]io.ReadCloser),
	}

	// Layer 1 content
	files1 := map[string]string{"file1": "content1"}
	r1, _ := createTarStream(files1)
	mockStore.diffs[layer1ID] = ioutil.NopCloser(r1)

	// Layer 2 content
	files2 := map[string]string{"file2": "content2"}
	r2, _ := createTarStream(files2)
	mockStore.diffs[layer2ID] = ioutil.NopCloser(r2)

	analyzer := &Analyzer{
		Store:    mockStore,
		ImageRef: imageID,
	}

	layers, waste, err := analyzer.AnalyzeLayers(context.Background())
	assert.NoError(t, err)

	assert.Len(t, layers, 2)
	// Base layer (layer1) should be first
	assert.Equal(t, layer1ID, layers[0].Digest)
	assert.Equal(t, 1, layers[0].AddedFileCount)

	// Top layer (layer2)
	assert.Equal(t, layer2ID, layers[1].Digest)
	assert.Equal(t, 1, layers[1].AddedFileCount)

	assert.Empty(t, waste.Duplicates)
}

// Helper to create a tar stream in memory
func createTarStream(files map[string]string) (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

func TestProcessLayerTar_Basic(t *testing.T) {
	files := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}
	r, err := createTarStream(files)
	assert.NoError(t, err)

	size, added, deleted, topFiles, hashes, err := processLayerTar(r, false)

	assert.NoError(t, err)
	assert.Equal(t, int64(16), size) // 8 + 8
	assert.Equal(t, 2, added)
	assert.Equal(t, 0, deleted)
	assert.Len(t, topFiles, 2)
	assert.Len(t, hashes, 2)
}

func TestProcessLayerTar_Whiteout(t *testing.T) {
	files := map[string]string{
		"file1.txt":     "content1",
		".wh.file2.txt": "", // Whiteout for file2.txt
		".wh..wh.opq":   "", // Opaque whiteout
	}
	r, err := createTarStream(files)
	assert.NoError(t, err)

	size, added, deleted, _, _, err := processLayerTar(r, false)

	assert.NoError(t, err)
	assert.Equal(t, int64(8), size) // Only file1.txt has content
	assert.Equal(t, 1, added)       // Only file1.txt is added file
	assert.Equal(t, 2, deleted)     // 2 whiteouts
}

func TestProcessLayerTar_Hashing(t *testing.T) {
	// Layer 1
	files1 := map[string]string{
		"common.txt": "shared content",
	}
	r1, err := createTarStream(files1)
	assert.NoError(t, err)

	_, _, _, _, hashes1, err := processLayerTar(r1, false)
	assert.NoError(t, err)
	assert.Len(t, hashes1, 1)
	hash1 := hashes1["/common.txt"]
	assert.NotEmpty(t, hash1)

	// Layer 2 adds same file
	files2 := map[string]string{
		"common.txt": "shared content",
		"unique.txt": "unique content",
	}
	r2, err := createTarStream(files2)
	assert.NoError(t, err)

	_, _, _, _, hashes2, err := processLayerTar(r2, false)
	assert.NoError(t, err)
	assert.Len(t, hashes2, 2)
	hash2 := hashes2["/common.txt"]

	// Check that hashes match
	assert.Equal(t, hash1, hash2)
}

func TestAnalyzePackages_RPM(t *testing.T) {
	// This test relies on the container environment having a valid RPM DB at /var/lib/rpm
	// or skips if not present.
	dbPath := "/var/lib/rpm"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("Skipping RPM test: /var/lib/rpm not found")
	}

	pkgs, err := scanRPMs("/")
	assert.NoError(t, err)

	// If the container is minimal, it might have very few packages, but should have some.
	// If scanRPMs returns 0 packages but no error, it might be valid but suspicious for a full OS container.
	// For now, we just assert no error.
	if len(pkgs) > 0 {
		t.Logf("Found %d packages", len(pkgs))
		// Check that Digest is populated if we have packages
		// Note: Not all RPMs might have SigMD5, but most do.
		// Let's just log one.
		t.Logf("First package: %+v", pkgs[0])
	}
}

func TestAnalyzeFilesystem_ELF(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fs-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a dummy ELF file
	elfPath := filepath.Join(tmpDir, "test-binary")
	f, err := os.Create(elfPath)
	assert.NoError(t, err)

	// Minimal 64-bit ELF Header (x86-64)
	header := []byte{
		0x7f, 'E', 'L', 'F',
		2, // 64-bit
		1, // Little Endian
		1, // Version
		0, 0, 0, 0, 0, 0, 0, 0, 0,
		2, 0, // e_type (EXEC)
		62, 0, // e_machine (x86-64 = 62)
		1, 0, 0, 0, // e_version
	}
	// Pad to 64 bytes (size of Elf64_Ehdr)
	padding := make([]byte, 64-len(header))
	f.Write(header)
	f.Write(padding)
	f.Close()

	analyzer := &Analyzer{}
	fsInfo, arch, err := analyzer.AnalyzeFilesystem(context.Background(), tmpDir)
	assert.NoError(t, err)

	foundELF := false
	for _, ft := range fsInfo.FileTypes {
		if ft.Type == "ELF Binary" {
			foundELF = true
			assert.Equal(t, 1, ft.Count)
		}
	}
	assert.True(t, foundELF, "Did not identify ELF Binary")
	assert.Equal(t, "EM_X86_64", arch)
}

func TestAnalyzeFilesystem_OtherTypes(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fs-types-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	files := map[string]string{
		"script.sh":   "#!/bin/bash\necho hello",
		"config.yaml": "key: value",
		"src.go":      "package main",
		"empty":       "",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		err := ioutil.WriteFile(path, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// Symlink
	os.Symlink("script.sh", filepath.Join(tmpDir, "link.sh"))

	analyzer := &Analyzer{}
	fsInfo, _, err := analyzer.AnalyzeFilesystem(context.Background(), tmpDir)
	assert.NoError(t, err)

	expectedTypes := map[string]bool{
		"Script":      true,
		"Config/Data": true,
		"Go Source":   true,
		"Empty/Small": true,
		"Symlink":     true,
	}

	for _, ft := range fsInfo.FileTypes {
		if expectedTypes[ft.Type] {
			delete(expectedTypes, ft.Type)
		}
	}
	assert.Empty(t, expectedTypes, "Missing expected file types")
}

func TestScanPython(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "py-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Setup directory structure
	// 1. .dist-info
	distDir := filepath.Join(tmpDir, "usr/lib/python3.9/site-packages/pkg1-1.0.0.dist-info")
	assert.NoError(t, os.MkdirAll(distDir, 0755))

	metaContent := "Name: pkg1\nVersion: 1.0.0\nLicense: MIT\n"
	assert.NoError(t, ioutil.WriteFile(filepath.Join(distDir, "METADATA"), []byte(metaContent), 0644))

	// 2. .egg-info (PKG-INFO)
	eggDir := filepath.Join(tmpDir, "usr/local/lib/python3.9/site-packages/pkg2-2.0.0.egg-info")
	assert.NoError(t, os.MkdirAll(eggDir, 0755))

	pkgInfoContent := "Name: pkg2\nVersion: 2.0.0\nLicense: Apache-2.0\n"
	assert.NoError(t, ioutil.WriteFile(filepath.Join(eggDir, "PKG-INFO"), []byte(pkgInfoContent), 0644))

	// 3. Ignored dir
	assert.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "usr/lib/other"), 0755))

	// Run scanPython (it's not exported, but we are in same package)
	// scanPython takes rootfs string
	pkgs, err := scanPython(tmpDir)
	assert.NoError(t, err)

	assert.Len(t, pkgs, 2)

	// Convert to map for easy checking
	pkgMap := make(map[string]types.Package)
	for _, p := range pkgs {
		pkgMap[p.Name] = p
	}

	assert.Contains(t, pkgMap, "pkg1")
	assert.Equal(t, "1.0.0", pkgMap["pkg1"].Version)
	assert.Equal(t, "MIT", pkgMap["pkg1"].License)
	assert.NotEmpty(t, pkgMap["pkg1"].Digest, "Digest should not be empty")

	assert.Contains(t, pkgMap, "pkg2")
	assert.Equal(t, "2.0.0", pkgMap["pkg2"].Version)
	assert.Equal(t, "Apache-2.0", pkgMap["pkg2"].License)
	assert.NotEmpty(t, pkgMap["pkg2"].Digest, "Digest should not be empty")
}

func TestGenerateRecommendations_Legacy(t *testing.T) {
	analyzer := &Analyzer{}

	fs := types.FilesystemInfo{
		TopDirectories: []types.TopDirectory{
			{Path: "/var/cache/yum/x86_64", Size: 100 * 1024 * 1024}, // 100MB
			{Path: "/usr/share/doc", Size: 20 * 1024 * 1024},         // 20MB
		},
	}

	pkgs := types.PackageInfo{
		RPM: []types.Package{
			{Name: "gcc", Version: "9.0.0"},
			{Name: "glibc-devel", Version: "2.30"},
		},
	}

	waste := types.WasteDetection{
		Duplicates: []types.DuplicateFile{
			{Path: "/bin/bash", Size: 1024 * 1024},
		},
	}

	recs := analyzer.GenerateRecommendations(nil, pkgs, fs, waste, "", nil)

	assert.NotEmpty(t, recs)
	assert.Equal(t, "MERGE_LAYERS", recs[0].Code)
}

func TestFormatSize(t *testing.T) {
	assert.Equal(t, "500 B", formatSize(500))
	assert.Equal(t, "1.0 KB", formatSize(1024))
	assert.Equal(t, "1.5 MB", formatSize(1.5*1024*1024))
}

func TestProcessLayerTar_OtherTypes(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Directory
	hdr := &tar.Header{
		Name:     "dir/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}
	assert.NoError(t, tw.WriteHeader(hdr))

	// Symlink
	hdr = &tar.Header{
		Name:     "link",
		Typeflag: tar.TypeSymlink,
		Linkname: "target",
		Mode:     0777,
	}
	assert.NoError(t, tw.WriteHeader(hdr))

	assert.NoError(t, tw.Close())

	size, added, deleted, topFiles, _, err := processLayerTar(&buf, false)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), size)
	assert.Equal(t, 0, added) // Dirs and symlinks don't count as "added files" in our logic currently
	assert.Equal(t, 0, deleted)
	assert.Empty(t, topFiles)
}

func TestAnalyzePackages_Empty(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "empty-pkg-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// No RPM db, no Python libs

	// We can't easily test AnalyzePackages directly because it's a method on Analyzer
	// and we might not want to construct a full Analyzer if it has dependencies.
	// But AnalyzePackages only uses rootfs string.
	analyzer := &Analyzer{}
	info, err := analyzer.AnalyzePackages(context.Background(), tmpDir)
	assert.NoError(t, err)

	assert.Empty(t, info.RPM)
	assert.Empty(t, info.Python)
}

func TestAnalyzePackages_RPMError(t *testing.T) {
	// Create a corrupted RPM DB
	tmpDir, err := ioutil.TempDir("", "rpm-err-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "var/lib/rpm")
	os.MkdirAll(dbPath, 0755)

	// Create a file that is not a valid RPM DB
	ioutil.WriteFile(filepath.Join(dbPath, "Packages"), []byte("garbage"), 0644)

	pkgs, err := scanRPMs(tmpDir)
	// rpmdb.Open should fail or ListPackages should fail
	assert.Error(t, err)
	assert.Nil(t, pkgs)
}

func TestGetELFArch_Error(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "elf-err-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 1. Non-existent file
	_, err = getELFArch(filepath.Join(tmpDir, "missing"))
	assert.Error(t, err)

	// 2. Invalid ELF file (text file)
	path := filepath.Join(tmpDir, "text.txt")
	ioutil.WriteFile(path, []byte("not elf"), 0644)
	_, err = getELFArch(path)
	assert.Error(t, err)
}

func TestAnalyzeFilesystem_MoreFileTypes(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fs-more-types")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	files := map[string][]byte{
		"archive.gz": {0x1f, 0x8b, 0x08, 0x00},
		"app.jar":    {0x50, 0x4b, 0x03, 0x04},
		"lib.whl":    {0x50, 0x4b, 0x03, 0x04},
		"data.zip":   {0x50, 0x4b, 0x03, 0x04},
		"main.c":     []byte("#include <stdio.h>"),
		"main.cpp":   []byte("#include <iostream>"),
		"script.js":  []byte("console.log('hello')"),
		"mod.py":     []byte("import os"),
		"binary.bin": {0x00, 0x01, 0x02, 0x03, 0xff}, // Binary (null byte or non-text)
		"text.txt":   []byte("Just some plain text"),
	}

	for name, content := range files {
		err := ioutil.WriteFile(filepath.Join(tmpDir, name), content, 0644)
		assert.NoError(t, err)
	}

	analyzer := &Analyzer{}
	fsInfo, _, err := analyzer.AnalyzeFilesystem(context.Background(), tmpDir)
	assert.NoError(t, err)

	typeMap := make(map[string]int)
	for _, ft := range fsInfo.FileTypes {
		typeMap[ft.Type] = ft.Count
	}

	assert.Contains(t, typeMap, "Gzip Archive")
	assert.Contains(t, typeMap, "Java Jar")
	assert.Contains(t, typeMap, "Python Wheel")
	assert.Contains(t, typeMap, "Zip Archive")
	assert.Contains(t, typeMap, "C/C++ Source")
	assert.Contains(t, typeMap, "JavaScript")
	assert.Contains(t, typeMap, "Python Source/Bytecode")
	assert.Contains(t, typeMap, "Binary Data")
	assert.Contains(t, typeMap, "Text")
}

func TestProcessLayerTar_ReadError(t *testing.T) {
	// A reader that returns error
	r := &errorReader{err: fmt.Errorf("read error")}
	_, _, _, _, _, err := processLayerTar(r, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading tar stream")
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestParsePythonMetadata_WithDependencies(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "py-dep-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	distDir := filepath.Join(tmpDir, "usr/lib/python3.9/site-packages/pkg1-1.0.0.dist-info")
	assert.NoError(t, os.MkdirAll(distDir, 0755))

	metaContent := `Name: pkg1
Version: 1.0.0
License: MIT
Requires-Dist: requests (>= 2.25.0)
Requires-Dist: numpy; python_version < "3.8"
Requires-Dist: pandas[all]
Requires-Dist: simplejson>=3.0
`
	assert.NoError(t, ioutil.WriteFile(filepath.Join(distDir, "METADATA"), []byte(metaContent), 0644))

	pkgs, err := scanPython(tmpDir)
	assert.NoError(t, err)
	assert.Len(t, pkgs, 1)

	p := pkgs[0]
	assert.Equal(t, "pkg1", p.Name)
	assert.Contains(t, p.Provides, "pkg1")

	expectedDeps := []string{"requests", "numpy", "pandas", "simplejson"}
	for _, dep := range expectedDeps {
		assert.Contains(t, p.Requires, dep)
	}
}
