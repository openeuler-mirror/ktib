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

	total, reqSize, saving := scanner.AssessFatSlim(required)

	// Total libs size = 100 + 200 + 300 = 600 (notalib.txt ignored)
	// Required size = 100 + 300 = 400
	// Saving = 200

	assert.Equal(t, int64(600), total)
	assert.Equal(t, int64(400), reqSize)
	assert.Equal(t, int64(200), saving)
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
