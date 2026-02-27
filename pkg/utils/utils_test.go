/*
   Copyright (c) 2025 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package utils

import (
	"github.com/containers/buildah/define"
	"github.com/spf13/cobra"
	"os"
	"testing"
)

func TestFileExists(t *testing.T) {
	// Create a temporary file
	file, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer file.Close()

	// Create a temporary directory
	dir := t.TempDir()

	tests := []struct {
		path     string
		expected bool
	}{
		{file.Name(), true},     // Valid file
		{dir, false},            // Valid directory
		{"invalid/path", false}, // Invalid path
	}

	for _, test := range tests {
		result := FileExists(test.path)
		if result != test.expected {
			t.Errorf("FileExists(%q) = %v; want %v", test.path, result, test.expected)
		}
	}
}

func TestIsDir(t *testing.T) {
	// Create a temporary file
	file, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(file.Name())

	// Create a temporary directory
	dir := t.TempDir()

	tests := []struct {
		path     string
		expected bool
	}{
		{file.Name(), false},    // File is not a directory
		{dir, true},             // Directory is a directory
		{"invalid/path", false}, // Invalid path
	}

	for _, test := range tests {
		result := IsDir(test.path)
		if result != test.expected {
			t.Errorf("IsDir(%q) = %v; want %v", test.path, result, test.expected)
		}
	}
}

func TestDefaultFormat(t *testing.T) {
	// Save the original environment variable
	origFormat := os.Getenv("BUILDAH_FORMAT")
	defer os.Setenv("BUILDAH_FORMAT", origFormat)

	tests := []struct {
		env      string
		expected string
	}{
		{"", define.OCI},                   // No environment variable set
		{"custom_format", "custom_format"}, // Environment variable set
	}

	for _, test := range tests {
		os.Setenv("BUILDAH_FORMAT", test.env)
		result := DefaultFormat()
		if result != test.expected {
			t.Errorf("DefaultFormat() = %q; want %q", result, test.expected)
		}
	}
}

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		filename string
		hasError bool
	}{
		{"valid_filename", false},
		{"invalid:filename", true},
	}

	for _, test := range tests {
		err := ValidateFileName(test.filename)
		if (err != nil) != test.hasError {
			t.Errorf("ValidateFileName(%q) = %v; want error? %v", test.filename, err, test.hasError)
		}
	}
}

func TestExists(t *testing.T) {
	// Create a temporary file
	file, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(file.Name())

	// Create a temporary directory
	dir := t.TempDir()

	tests := []struct {
		path   string
		exists bool
	}{
		{file.Name(), true},     // File exists
		{dir, true},             // Directory exists
		{"invalid/path", false}, // Invalid path
	}

	for _, test := range tests {
		err := Exists(test.path)
		if (err == nil) != test.exists {
			t.Errorf("Exists(%q) = %v; want exists? %v", test.path, err == nil, test.exists)
		}
	}
}

func TestNoArgs(t *testing.T) {
	cmd := &cobra.Command{
		Use: "testcmd",
	}

	tests := []struct {
		args     []string
		hasError bool
	}{
		{[]string{}, false},              // No arguments
		{[]string{"arg1"}, true},         // One argument
		{[]string{"arg1", "arg2"}, true}, // Multiple arguments
	}

	for _, test := range tests {
		err := NoArgs(cmd, test.args)
		if (err != nil) != test.hasError {
			t.Errorf("NoArgs(%v) = %v; want error? %v", test.args, err, test.hasError)
		}
	}
}
