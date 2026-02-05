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

package fs

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/containers/storage"
	"github.com/stretchr/testify/assert"
)

// MockStore adapts storage.Store for testing
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
	if rc, ok := m.diffs[to]; ok {
		// In a real scenario we'd need to reset the reader, but for one-pass test it's okay
		// Or we can assume the test sets up a fresh reader for each call if needed.
		// Here we just return what's stored.
		return rc, nil
	}
	return nil, fmt.Errorf("diff not found")
}

// Helper to create tar content
func createTar(files map[string]string) (io.ReadCloser, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Sort keys to ensure deterministic order
	// This is critical for opaque whiteout tests where whiteout must appear before other files in the same dir
	var keys []string
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		content := files[name]
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
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
	return ioutil.NopCloser(&buf), nil
}

func TestExtractLayersWithFilter(t *testing.T) {
	// Setup Mock Store
	// Layer 1 (Base): /etc/os-release, /bin/bash
	// Layer 2 (Top): /bin/app, /bin/bash (overwrite)

	layer1ID := "layer1"
	layer2ID := "layer2"
	imageID := "test-image"

	mockStore := &MockStore{
		images: map[string]*storage.Image{
			imageID: {ID: imageID, TopLayer: layer2ID},
		},
		layers: map[string]*storage.Layer{
			layer2ID: {ID: layer2ID, Parent: layer1ID},
			layer1ID: {ID: layer1ID, Parent: ""},
		},
		diffs: make(map[string]io.ReadCloser),
	}

	tar1, _ := createTar(map[string]string{
		"etc/os-release": "ID=kylin",
		"bin/bash":       "bash-v1",
		"bin/ls":         "ls-v1",
	})
	mockStore.diffs[layer1ID] = tar1

	tar2, _ := createTar(map[string]string{
		"bin/app":  "my-app",
		"bin/bash": "bash-v2", // Overwrite
	})
	mockStore.diffs[layer2ID] = tar2

	// Output directory
	tmpDir, err := ioutil.TempDir("", "fs-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	synthesizer := NewDefaultSynthesizer(mockStore)

	// Case 1: Keep everything
	whitelistAll := func(p string) bool { return true }
	err = synthesizer.extractLayersWithFilter(imageID, tmpDir, whitelistAll)
	assert.NoError(t, err)

	// Verify files
	assertFileContent(t, filepath.Join(tmpDir, "etc/os-release"), "ID=kylin")
	assertFileContent(t, filepath.Join(tmpDir, "bin/app"), "my-app")
	assertFileContent(t, filepath.Join(tmpDir, "bin/bash"), "bash-v2") // Should be from Top layer
	assertFileContent(t, filepath.Join(tmpDir, "bin/ls"), "ls-v1")

	// Case 2: Keep only specific files
	tmpDir2, _ := ioutil.TempDir("", "fs-test-2-")
	defer os.RemoveAll(tmpDir2)

	whitelistSpecific := func(p string) bool {
		return p == "/bin/app" || p == "/bin/ls"
	}

	// Reset readers because they were consumed
	tar1, _ = createTar(map[string]string{
		"etc/os-release": "ID=kylin",
		"bin/bash":       "bash-v1",
		"bin/ls":         "ls-v1",
	})
	mockStore.diffs[layer1ID] = tar1

	tar2, _ = createTar(map[string]string{
		"bin/app":  "my-app",
		"bin/bash": "bash-v2",
	})
	mockStore.diffs[layer2ID] = tar2

	err = synthesizer.extractLayersWithFilter(imageID, tmpDir2, whitelistSpecific)
	assert.NoError(t, err)

	assertFileContent(t, filepath.Join(tmpDir2, "bin/app"), "my-app")
	assertFileContent(t, filepath.Join(tmpDir2, "bin/ls"), "ls-v1")
	assertFileMissing(t, filepath.Join(tmpDir2, "bin/bash"))
	assertFileMissing(t, filepath.Join(tmpDir2, "etc/os-release"))
}

func TestApplyTarWithFilter_Whiteout(t *testing.T) {
	// Test whiteout handling
	// Layer contains .wh.foo, meaning foo should be deleted

	tmpDir, err := ioutil.TempDir("", "wh-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file to be deleted
	os.MkdirAll(filepath.Join(tmpDir, "etc"), 0755)
	ioutil.WriteFile(filepath.Join(tmpDir, "etc/foo"), []byte("exists"), 0644)

	// Tar with whiteout
	tarData, _ := createTar(map[string]string{
		"etc/.wh.foo": "",
	})

	err = applyTarWithFilter(tarData, tmpDir, func(p string) bool { return true })
	assert.NoError(t, err)

	assertFileMissing(t, filepath.Join(tmpDir, "etc/foo"))
}

func TestApplyTarWithFilter_OpaqueWhiteout(t *testing.T) {
	// Test opaque whiteout handling (.wh.opq)
	// Layer contains dir/.wh.opq, meaning dir should be emptied

	tmpDir, err := ioutil.TempDir("", "opq-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory with existing files
	dirPath := filepath.Join(tmpDir, "etc")
	os.MkdirAll(dirPath, 0755)
	ioutil.WriteFile(filepath.Join(dirPath, "old1"), []byte("v1"), 0644)
	ioutil.WriteFile(filepath.Join(dirPath, "old2"), []byte("v1"), 0644)

	// Tar with opaque whiteout and a new file
	tarData, _ := createTar(map[string]string{
		"etc/.wh..wh.opq": "",
		"etc/new":         "v2",
	})

	err = applyTarWithFilter(tarData, tmpDir, func(p string) bool { return true })
	assert.NoError(t, err)

	// old files should be gone
	assertFileMissing(t, filepath.Join(dirPath, "old1"))
	assertFileMissing(t, filepath.Join(dirPath, "old2"))

	// new file should exist
	assertFileContent(t, filepath.Join(dirPath, "new"), "v2")
}

func assertFileContent(t *testing.T, path, expected string) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read file %s: %v", path, err)
		return
	}
	if string(content) != expected {
		t.Errorf("File %s content mismatch. Got %s, want %s", path, string(content), expected)
	}
}

func assertFileMissing(t *testing.T, path string) {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("File %s should not exist", path)
	}
}
