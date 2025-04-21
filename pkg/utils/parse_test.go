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
package utils

import (
	"errors"
	"fmt"
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/buildah/define"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDockerfiles(t *testing.T) {
	tempDir := t.TempDir()

	// temp目录下创建测试文件
	containerFilePath := filepath.Join(tempDir, "Containerfile")
	dockerFilePath := filepath.Join(tempDir, "Dockerfile")
	if err := os.WriteFile(containerFilePath, []byte("FROM kylin:test\n"), 0644); err != nil {
		t.Fatalf("Failed to create Containerfile: %v", err)
	}
	if err := os.WriteFile(dockerFilePath, []byte("FROM kylin:test\n"), 0644); err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	tests := []struct {
		name      string
		op        *options.BuildOptions
		args      []string
		expected  []string
		context   string
		expectErr bool
	}{
		{
			name:      "Single Dockerfile path",
			op:        &options.BuildOptions{File: []string{dockerFilePath}},
			args:      []string{},
			expected:  []string{dockerFilePath},
			context:   tempDir,
			expectErr: false,
		},
		{
			name:      "Single Containerfile path",
			op:        &options.BuildOptions{File: []string{containerFilePath}},
			args:      []string{},
			expected:  []string{containerFilePath},
			context:   tempDir,
			expectErr: false,
		},
		{
			name:      "Default to Containerfile when no args and no files specified",
			op:        &options.BuildOptions{File: []string{}},
			args:      []string{},
			expected:  []string{containerFilePath},
			context:   tempDir,
			expectErr: false,
		},
		{
			name:      "Default to Dockerfile when Containerfile not present",
			op:        &options.BuildOptions{File: []string{}},
			args:      []string{tempDir},
			expected:  []string{dockerFilePath},
			context:   tempDir,
			expectErr: false,
		},
		{
			name:      "Context directory does not exist",
			op:        &options.BuildOptions{File: []string{}},
			args:      []string{"nonexistent/directory"},
			expected:  nil,
			context:   "",
			expectErr: true,
		},
		{
			name:      "Invalid file paths",
			op:        &options.BuildOptions{File: []string{"invalid/path"}},
			args:      []string{},
			expected:  nil,
			context:   "",
			expectErr: true,
		},
		{
			name:      "Invalid directory path",
			op:        &options.BuildOptions{File: []string{containerFilePath}},
			args:      []string{"invalid/directory"},
			expected:  nil,
			context:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dockerfiles, contextDir, err := ResolveDockerfiles(tt.op, tt.args)
			if (err != nil) != tt.expectErr {
				t.Errorf("ResolveDockerfiles() error = %v, wantErr %v", err, tt.expectErr)
				return
			}
			if contextDir != tt.context {
				t.Errorf("ResolveDockerfiles() contextDir = %v, want %v", contextDir, tt.context)
			}
			if len(dockerfiles) != len(tt.expected) {
				t.Errorf("ResolveDockerfiles() dockerfiles length = %v, want %v", len(dockerfiles), len(tt.expected))
			}
			for i, file := range dockerfiles {
				if file != tt.expected[i] {
					t.Errorf("ResolveDockerfiles() dockerfiles[%d] = %v, want %v", i, file, tt.expected[i])
				}
			}
		})
	}
}

// 用于比较两个 string 类型的切片 (a 和 b)，并判断a,b是否相等。相等条件：长度相同，且所有对应的元素相同。
func compareSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// compareBuildOptions compares two BuildOptions and returns true if they are equal.
func compareBuildOptions(a, b *define.BuildOptions) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Compare all fields of the structs here
	return compareSlices(a.AdditionalTags, b.AdditionalTags) &&
		a.ContextDirectory == b.ContextDirectory &&
		a.Err == b.Err &&
		a.NoCache == b.NoCache &&
		a.RemoveIntermediateCtrs == b.RemoveIntermediateCtrs &&
		a.Runtime == b.Runtime &&
		a.ReportWriter == b.ReportWriter &&
		a.Out == b.Out &&
		a.Output == b.Output &&
		a.OutputFormat == b.OutputFormat
}

func TestParseBuildOptions(t *testing.T) {
	tests := []struct {
		name           string
		cmdFlagChanged bool
		flags          *options.BuildOptions
		contextDir     string
		expectedOpts   *define.BuildOptions
		expectedErr    error
	}{
		{
			name:           "Valid options with OCI format",
			cmdFlagChanged: true,
			flags: &options.BuildOptions{
				Tags:    []string{"tag1", "tag2"},
				Format:  "oci",
				NoCache: true,
				Rm:      true,
				Runtime: "docker",
			},
			contextDir: "/context",
			expectedOpts: &define.BuildOptions{
				AdditionalTags:         []string{"tag2"},
				ContextDirectory:       "/context",
				Err:                    os.Stderr,
				NoCache:                true,
				RemoveIntermediateCtrs: true,
				Runtime:                "/usr/bin/docker",
				ReportWriter:           os.Stderr,
				Out:                    os.Stdout,
				Output:                 "tag1",
				OutputFormat:           define.OCIv1ImageManifest,
			},
			expectedErr: nil,
		},
		{
			name:           "Invalid format",
			cmdFlagChanged: false,
			flags: &options.BuildOptions{
				Tags:    []string{},
				Format:  "invalid",
				NoCache: false,
				Rm:      false,
				Runtime: "docker",
			},
			contextDir:   "/context",
			expectedOpts: nil,
			expectedErr:  fmt.Errorf("unrecognized image type %q", "invalid"),
		},
		{
			name:           "Runtime path not found",
			cmdFlagChanged: false,
			flags: &options.BuildOptions{
				Tags:    []string{},
				Format:  "docker",
				NoCache: false,
				Rm:      false,
				Runtime: "nonexistent-runtime",
			},
			contextDir:   "/context",
			expectedOpts: nil,
			expectedErr:  errors.New("command not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock cobra.Command
			cmd := &cobra.Command{}
			if tt.cmdFlagChanged {
				cmd.Flags().String("tag", "", "")
				cmd.Flag("tag").Changed = true
			}

			// Test ParseBuildOptions
			opts, err := ParseBuildOptions(cmd, tt.flags, tt.contextDir)
			if (err != nil) != (tt.expectedErr != nil) {
				t.Errorf("ParseBuildOptions() error = %v, wantErr %v", err, tt.expectedErr)
				return
			}
			if err != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("ParseBuildOptions() error = %v, want %v", err, tt.expectedErr)
			}

			if opts == nil && tt.expectedOpts == nil {
				return // Both are nil, so they're considered equal
			}

			if opts == nil || tt.expectedOpts == nil {
				t.Errorf("ParseBuildOptions() opts = %v, want %v", opts, tt.expectedOpts)
				return
			}

			// Compare opts and expectedOpts field by field
			if !compareBuildOptions(opts, tt.expectedOpts) {
				t.Errorf("ParseBuildOptions() opts = %+v, want %+v", opts, tt.expectedOpts)
			}
		})
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500.00B"},
		{1023, "1023.00B"},
		{1024, "1.00KB"},
		{2048, "2.00KB"},
		{1048575, "1023.99KB"},
		{1048576, "1.00MB"},
		{2097152, "2.00MB"},
		{1073741823, "1023.99MB"},
		{1073741824, "1.00GB"},
		{2147483648, "2.00GB"},
		{1099511627775, "1023.99GB"},
		{1099511627776, "1.00TB"},
		{2199023255552, "2.00TB"},
	}

	for _, test := range tests {
		result := humanSize(test.input)
		if result != test.expected {
			t.Errorf("humanSize(%d) = %s; want %s", test.input, result, test.expected)
		}
	}
}
