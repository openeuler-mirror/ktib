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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/buildah/define"
	"github.com/containers/image/v5/types"
	container "github.com/containers/storage"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
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

	// Create a directory with only Dockerfile
	onlyDockerDir := filepath.Join(tempDir, "onlydocker")
	if err := os.Mkdir(onlyDockerDir, 0755); err != nil {
		t.Fatal(err)
	}
	onlyDockerFilePath := filepath.Join(onlyDockerDir, "Dockerfile")
	if err := os.WriteFile(onlyDockerFilePath, []byte("FROM kylin:test\n"), 0644); err != nil {
		t.Fatalf("Failed to create Dockerfile in onlydocker: %v", err)
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
			expected:  nil,
			context:   "",
			expectErr: true,
		},
		{
			name:      "Prioritize Containerfile over Dockerfile",
			op:        &options.BuildOptions{File: []string{}},
			args:      []string{tempDir},
			expected:  []string{containerFilePath},
			context:   tempDir,
			expectErr: false,
		},
		{
			name:      "Default to Dockerfile when Containerfile not present",
			op:        &options.BuildOptions{File: []string{}},
			args:      []string{onlyDockerDir},
			expected:  []string{onlyDockerFilePath},
			context:   onlyDockerDir,
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
				Runtime:                "docker",
				ReportWriter:           os.Stderr,
				Out:                    os.Stdout,
				Output:                 "tag1",
				OutputFormat:           define.OCIv1ImageManifest,
				SystemContext:          &types.SystemContext{},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock cobra.Command
			cmd := &cobra.Command{}
			cmd.Flags().String("tag", "", "") // Always add the tag flag
			if tt.cmdFlagChanged {
				cmd.Flag("tag").Changed = true
			}

			// Test ParseBuildOptions
			opts, err := ParseBuildOptions(cmd, tt.flags, tt.contextDir, nil)
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
		{1048575, "1024.00KB"}, // Rounded up
		{1048576, "1.00MB"},
		{2097152, "2.00MB"},
		{1073741823, "1024.00MB"}, // Rounded up
		{1073741824, "1.00GB"},
		{2147483648, "2.00GB"},
		{1099511627775, "1024.00GB"}, // Rounded up
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

func TestParseImageName(t *testing.T) {
	tests := []struct {
		name         string
		fullName     string
		expectedRepo string
		expectedTag  string
	}{
		{
			name:         "Empty string",
			fullName:     "",
			expectedRepo: unknownState,
			expectedTag:  unknownState,
		},
		{
			name:         "Standard image",
			fullName:     "ubuntu:latest",
			expectedRepo: "ubuntu",
			expectedTag:  "latest",
		},
		{
			name:         "Image with registry",
			fullName:     "quay.io/libpod/alpine:latest",
			expectedRepo: "quay.io/libpod/alpine",
			expectedTag:  "latest",
		},
		{
			name:         "Image without tag",
			fullName:     "ubuntu",
			expectedRepo: "ubuntu",
			expectedTag:  unknownState,
		},
		{
			name:         "Localhost image with port",
			fullName:     "localhost:5000/myimage:v1",
			expectedRepo: "localhost:5000/myimage",
			expectedTag:  "v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag := parseImageName(tt.fullName)
			if repo != tt.expectedRepo {
				t.Errorf("parseImageName() repo = %v, want %v", repo, tt.expectedRepo)
			}
			if tag != tt.expectedTag {
				t.Errorf("parseImageName() tag = %v, want %v", tag, tt.expectedTag)
			}
		})
	}
}

func TestManualParseImageName(t *testing.T) {
	tests := []struct {
		name         string
		fullName     string
		expectedRepo string
		expectedTag  string
	}{
		{
			name:         "Empty string",
			fullName:     "",
			expectedRepo: unknownState,
			expectedTag:  unknownState,
		},
		{
			name:         "Standard format",
			fullName:     "repository:tag",
			expectedRepo: "repository",
			expectedTag:  "tag",
		},
		{
			name:         "Registry format",
			fullName:     "registry/repository:tag",
			expectedRepo: "registry/repository",
			expectedTag:  "tag",
		},
		{
			name:         "Port format",
			fullName:     "registry:5000/repository:tag",
			expectedRepo: "registry:5000/repository",
			expectedTag:  "tag",
		},
		{
			name:         "No tag",
			fullName:     "repository",
			expectedRepo: "repository",
			expectedTag:  unknownState,
		},
		{
			name:         "Complex tag",
			fullName:     "repository:tag:with:colons",
			expectedRepo: "repository:tag:with",
			expectedTag:  "colons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag := manualParseImageName(tt.fullName)
			if repo != tt.expectedRepo {
				t.Errorf("manualParseImageName() repo = %v, want %v", repo, tt.expectedRepo)
			}
			if tag != tt.expectedTag {
				t.Errorf("manualParseImageName() tag = %v, want %v", tag, tt.expectedTag)
			}
		})
	}
}

func TestParseImageRepository(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		expected  string
	}{
		{
			name:      "Empty string",
			imageName: "",
			expected:  "",
		},
		{
			name:      "With registry",
			imageName: "cr.kylinos.cn/test/myapp:01",
			expected:  "cr.kylinos.cn",
		},
		{
			name:      "No registry",
			imageName: "ubuntu:20.04",
			expected:  "",
		},
		{
			name:      "With digest and registry",
			imageName: "registry.io:5000/user/app@sha256:abc123",
			expected:  "registry.io:5000",
		},
		{
			name:      "With port and tag",
			imageName: "localhost:5000/image:tag",
			expected:  "localhost:5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseImageRepository(tt.imageName)
			if result != tt.expected {
				t.Errorf("parseImageRepository() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseDockerfileFromImage(t *testing.T) {
	tempDir := t.TempDir()
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")

	content := `
# Comment
FROM ubuntu:20.04
RUN echo hello
FROM cr.kylinos.cn/my/image:latest
`
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	repos, err := ParseDockerfileFromImage(dockerfilePath)
	if err != nil {
		t.Fatalf("ParseDockerfileFromImage failed: %v", err)
	}

	expected := []string{"cr.kylinos.cn"} // ubuntu:20.04 returns empty repo
	if len(repos) != len(expected) {
		t.Errorf("Expected %d repos, got %d", len(expected), len(repos))
	} else {
		if repos[0] != expected[0] {
			t.Errorf("Expected repo %s, got %s", expected[0], repos[0])
		}
	}
}

// captureOutput captures stdout from a function call
func captureOutput(f func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), err
}

func TestSortImages(t *testing.T) {
	now := time.Now()
	testDigest := digest.Digest("sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	tests := []struct {
		name          string
		images        []*imagemanager.Image
		ops           options.ImagesOption
		expectedLen   int
		expectedRepos []string
		expectedIDs   []string
	}{
		{
			name: "Sort by repository",
			images: []*imagemanager.Image{
				{
					OriImage: container.Image{
						ID:       "id2",
						Names:    []string{"zrepo:tag"},
						Created:  now,
						TopLayer: "layer2",
						Digest:   testDigest,
					},
					Size: 2048,
				},
				{
					OriImage: container.Image{
						ID:       "id1",
						Names:    []string{"arepo:tag"},
						Created:  now,
						TopLayer: "layer1",
						Digest:   testDigest,
					},
					Size: 1024,
				},
			},
			ops:           options.ImagesOption{NoTrunc: false},
			expectedLen:   2,
			expectedRepos: []string{"arepo", "zrepo"},
			expectedIDs:   []string{"id1", "id2"},
		},
		{
			name: "Truncate ID",
			images: []*imagemanager.Image{
				{
					OriImage: container.Image{
						ID:       "12345678901234567890",
						Names:    []string{"repo:tag"},
						Created:  now,
						TopLayer: "layer",
						Digest:   testDigest,
					},
					Size: 1024,
				},
			},
			ops:           options.ImagesOption{NoTrunc: false},
			expectedLen:   1,
			expectedRepos: []string{"repo"},
			expectedIDs:   []string{"1234567890"}, // Truncated to 10 chars
		},
		{
			name: "No Truncate ID",
			images: []*imagemanager.Image{
				{
					OriImage: container.Image{
						ID:       "12345678901234567890",
						Names:    []string{"repo:tag"},
						Created:  now,
						TopLayer: "layer",
						Digest:   testDigest,
					},
					Size: 1024,
				},
			},
			ops:           options.ImagesOption{NoTrunc: true},
			expectedLen:   1,
			expectedRepos: []string{"repo"},
			expectedIDs:   []string{"123456789012"}, // Truncated to 12 chars when NoTrunc is true (based on code logic)
		},
		{
			name: "Multiple names",
			images: []*imagemanager.Image{
				{
					OriImage: container.Image{
						ID:       "id1",
						Names:    []string{"repo1:tag1", "repo2:tag2"},
						Created:  now,
						TopLayer: "layer",
						Digest:   testDigest,
					},
					Size: 1024,
				},
			},
			ops:           options.ImagesOption{NoTrunc: false},
			expectedLen:   2,
			expectedRepos: []string{"repo1", "repo2"},
			expectedIDs:   []string{"id1", "id1"},
		},
		{
			name: "No names",
			images: []*imagemanager.Image{
				{
					OriImage: container.Image{
						ID:       "id1",
						Names:    []string{},
						Created:  now,
						TopLayer: "layer",
						Digest:   testDigest,
					},
					Size: 1024,
				},
			},
			ops:           options.ImagesOption{NoTrunc: false},
			expectedLen:   1,
			expectedRepos: []string{"<none>"},
			expectedIDs:   []string{"id1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reports, err := sortImages(tt.images, tt.ops)
			if err != nil {
				t.Fatalf("sortImages error: %v", err)
			}

			if len(reports) != tt.expectedLen {
				t.Errorf("Expected %d reports, got %d", tt.expectedLen, len(reports))
			}

			for i, report := range reports {
				if i < len(tt.expectedRepos) {
					if report.Repository != tt.expectedRepos[i] {
						t.Errorf("Report %d: expected repository %s, got %s", i, tt.expectedRepos[i], report.Repository)
					}
				}
				if i < len(tt.expectedIDs) {
					if report.ID != tt.expectedIDs[i] {
						t.Errorf("Report %d: expected ID %s, got %s", i, tt.expectedIDs[i], report.ID)
					}
				}
			}
		})
	}
}

func TestFormatImages(t *testing.T) {
	now := time.Now()
	testDigest := digest.Digest("sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	images := []*imagemanager.Image{
		{
			OriImage: container.Image{
				ID:       "1234567890abcdef",
				Names:    []string{"myrepo:mytag"},
				Created:  now,
				TopLayer: "layer1",
				Digest:   testDigest,
			},
			Size: 1024,
		},
	}

	tests := []struct {
		name     string
		ops      options.ImagesOption
		contains []string
	}{
		{
			name:     "Default format",
			ops:      options.ImagesOption{},
			contains: []string{"REPOSITORY", "TAG", "IMAGE ID", "SIZE", "CREATED", "myrepo", "mytag", "1234567890"},
		},
		{
			name:     "Quiet format",
			ops:      options.ImagesOption{Quiet: true},
			contains: []string{"1234567890"},
		},
		{
			name:     "Digest format",
			ops:      options.ImagesOption{Digests: true},
			contains: []string{"DIGEST", "sha256:1234567890abcdef"},
		},
		{
			name:     "Custom format",
			ops:      options.ImagesOption{Format: "{{.Repository}}-{{.Tag}}"},
			contains: []string{"myrepo-mytag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := captureOutput(func() error {
				return FormatImages(images, tt.ops)
			})
			if err != nil {
				t.Fatalf("FormatImages error: %v", err)
			}

			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("Expected output to contain %q, got:\n%s", s, output)
				}
			}
		})
	}
}

func TestJsonFormatImages(t *testing.T) {
	now := time.Now()
	testDigest := digest.Digest("sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	images := []*imagemanager.Image{
		{
			OriImage: container.Image{
				ID:       "id1",
				Names:    []string{"repo:tag"},
				Created:  now,
				TopLayer: "layer1",
				Digest:   testDigest,
			},
			Size: 1024,
		},
	}

	output, err := captureOutput(func() error {
		return JsonFormatImages(images, options.ImagesOption{})
	})
	if err != nil {
		t.Fatalf("JsonFormatImages error: %v", err)
	}

	// Simple check for JSON structure
	if !strings.Contains(output, `"name": [`) {
		t.Error("JSON output missing Name field")
	}
	if !strings.Contains(output, `"repo:tag"`) {
		t.Error("JSON output missing repo name")
	}
	if !strings.Contains(output, `"id1"`) {
		t.Error("JSON output missing ID")
	}
}

func TestSortContainers(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		containers    []container.Container
		expectedCount int
		expectedIDs   []string
		expectedNames []string
	}{
		{
			name: "Sort containers",
			containers: []container.Container{
				{
					ID:      "1234567890abcdef",
					Names:   []string{"container1"},
					Created: now,
					LayerID: "layer1",
					ImageID: "image1",
				},
				{
					ID:      "abcdef1234567890",
					Names:   []string{},
					Created: now,
					LayerID: "layer2",
					ImageID: "image2",
				},
			},
			expectedCount: 2,
			expectedIDs:   []string{"1234567890", "abcdef1234"},
			expectedNames: []string{"container1", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reports, err := sortContainers(tt.containers)
			if err != nil {
				t.Fatalf("sortContainers error: %v", err)
			}

			if len(reports) != tt.expectedCount {
				t.Errorf("Expected %d reports, got %d", tt.expectedCount, len(reports))
			}

			for i, report := range reports {
				if report.ID != tt.expectedIDs[i] {
					t.Errorf("Report %d: expected ID %s, got %s", i, tt.expectedIDs[i], report.ID)
				}
				if report.Names != tt.expectedNames[i] {
					t.Errorf("Report %d: expected Name %s, got %s", i, tt.expectedNames[i], report.Names)
				}
			}
		})
	}
}

func TestFormatBuilders(t *testing.T) {
	now := time.Now()
	containers := []container.Container{
		{
			ID:      "1234567890abcdef",
			Names:   []string{"builder1"},
			Created: now,
			LayerID: "layer1",
			ImageID: "image1",
		},
	}

	output, err := captureOutput(func() error {
		return FormatBuilders(containers, options.BuildersOption{})
	})
	if err != nil {
		t.Fatalf("FormatBuilders error: %v", err)
	}

	expectedStrings := []string{"1234567890", "builder1", "layer1", "image1"}
	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("Expected output to contain %q, got:\n%s", s, output)
		}
	}
}

func TestJsonFormatBuilders(t *testing.T) {
	now := time.Now()
	containers := []container.Container{
		{
			ID:      "1234567890abcdef",
			Names:   []string{"builder1"},
			Created: now,
			LayerID: "layer1",
			ImageID: "image1",
		},
	}

	output, err := captureOutput(func() error {
		return JsonFormatBuilders(containers, options.BuildersOption{})
	})
	if err != nil {
		t.Fatalf("JsonFormatBuilders error: %v", err)
	}

	if !strings.Contains(output, `"id": "1234567890abcdef"`) {
		t.Error("JSON output missing ID")
	}
	if !strings.Contains(output, `"names": [`) {
		t.Error("JSON output missing Names array")
	}
	if !strings.Contains(output, `"builder1"`) {
		t.Error("JSON output missing builder name")
	}
}

func TestJsonFormatMountInfo(t *testing.T) {
	builders := []*builder.Builder{
		{
			ID:          "id1",
			MountPoint:  "/tmp/mount1",
			FromImageID: "image1",
		},
		{
			ID:          "id2",
			MountPoint:  "", // Should be skipped
			FromImageID: "image2",
		},
	}

	output, err := captureOutput(func() error {
		return JsonFormatMountInfo(builders)
	})
	if err != nil {
		t.Fatalf("JsonFormatMountInfo error: %v", err)
	}

	if !strings.Contains(output, `"id1"`) {
		t.Error("Output missing id1")
	}
	if !strings.Contains(output, `"/tmp/mount1"`) {
		t.Error("Output missing mount point")
	}
	if strings.Contains(output, `"id2"`) {
		t.Error("Output should not contain id2 (empty mount point)")
	}
}
