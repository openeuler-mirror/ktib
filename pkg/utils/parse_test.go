#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

package utils

import (
	"bytes"
	"encoding/json"
	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/storage"
	container "github.com/containers/storage"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
	"time"
)

func TestHumanSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500.00B"},
		{1023, "1023.00B"},
		{1024, "1.00KB"},
		{2048, "2.00KB"},
		{1048575, "1024.00KB"},
		{1048576, "1.00MB"},
		{2097152, "2.00MB"},
		{1073741823, "1024.00MB"},
		{1073741824, "1.00GB"},
		{2147483648, "2.00GB"},
		{1099511627775, "1024.00GB"},
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
func TestSortImages(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		imgs     []imagemanager.Image
		expected []imageReport
	}{
		{
			name: "single image with names",
			imgs: []imagemanager.Image{
				{
					Size: 1024,
					OriImage: storage.Image{
						ID:       "1234567890abcdef",
						Digest:   digest.FromString("sha256:abcdef1234567890"),
						Created:  now.Add(-time.Hour),
						Names:    []string{"image1", "image2"},
						TopLayer: "layer1",
					},
				},
			},
			expected: []imageReport{
				{
					Name:     "image1",
					ID:       "1234567890",
					Digest:   digest.FromString("sha256:abcdef1234567890"),
					TopLayer: "layer1",
					Created:  "About an hour ago",
					Size:     "1.00KB",
				},
				{
					Name:     "image2",
					ID:       "1234567890",
					Digest:   digest.FromString("sha256:abcdef1234567890"),
					TopLayer: "layer1",
					Created:  "About an hour ago",
					Size:     "1.00KB",
				},
			},
		},
		{
			name: "single image with no names",
			imgs: []imagemanager.Image{
				{
					Size: 2048,
					OriImage: storage.Image{
						ID:       "abcdef1234567890",
						Digest:   digest.FromString("sha256:1234567890abcdef"),
						Created:  now.Add(-time.Hour * 2),
						Names:    []string{},
						TopLayer: "layer2",
					},
				},
			},
			expected: []imageReport{
				{
					Name:     unknownState,
					ID:       "abcdef1234",
					Digest:   digest.FromString("sha256:1234567890abcdef"),
					TopLayer: "layer2",
					Created:  "2 hours ago",
					Size:     "2.00KB",
				},
			},
		},
		// 可以添加更多测试用例
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sortImages(tt.imgs)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
func TestFormatImages(t *testing.T) {
	now := time.Now()

	// Simulate the images and options
	images := []imagemanager.Image{
		{
			Size: 2048,
			OriImage: storage.Image{
				ID:       "1234567890abcdef",
				Digest:   digest.FromString("sha256:abcdef1234567890"),
				Created:  now.Add(-time.Hour),
				Names:    []string{"image1"},
				TopLayer: "layer1",
			},
		},
		{
			Size: 4096,
			OriImage: storage.Image{
				ID:       "abcdef1234567890",
				Digest:   digest.FromString("sha256:1234567890abcdef"),
				Created:  now.Add(-time.Hour * 2),
				Names:    []string{"image2"},
				TopLayer: "layer2",
			},
		},
	}

	tests := []struct {
		name      string
		ops       options.ImagesOption
		expected  string
		expectErr bool
	}{
		{
			name: "default format",
			ops:  options.ImagesOption{Quiet: false, Digests: false},
			expected: `NAME        ID                      SIZE        TOP LAYER                           CREATED
image1      1234567890              2.00KB      layer1                              About an hour ago
image2      abcdef1234              4.00KB      layer2                              2 hours ago
`,
			expectErr: false,
		},
		{
			name: "quiet format",
			ops:  options.ImagesOption{Quiet: true},
			expected: `1234567890
abcdef1234
`,
			expectErr: false,
		},
		{
			name: "format with digest",
			ops:  options.ImagesOption{Digests: true},
			expected: `NAME        ID          DIGEST                                                                   SIZE        TOP LAYER   CREATED
image1      1234567890  sha256:c388bcbb21ce73d23ce515dcebced3b7a13d6116c5536d529e3ea8e1b1c87984  2.00KB      layer1      About an hour ago
image2      abcdef1234  sha256:d9be5dfca5c3189b0d8b6aea35bd30d91264606a1bf286b2fe8849aad91613b4  4.00KB      layer2      2 hours ago
`,
			expectErr: false,
		},
		{
			name: "custom format",
			ops:  options.ImagesOption{Format: "{{.ID}} {{.Name}}"},
			expected: `ID          NAME
1234567890  image1
abcdef1234  image2
`,
			expectErr: false,
		},
		{
			name:      "invalid format",
			ops:       options.ImagesOption{Format: "{{.InvalidField}}"},
			expected:  ``,
			expectErr: true,
		},
	}

	// Capture the output
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			defer func() { os.Stdout = oldStdout }()
			// Redirect stdout to buf
			r, w, _ := os.Pipe()
			os.Stdout = w
			err := FormatImages(images, tt.ops)
			// Close the writer and read from the pipe
			w.Close()
			var outBuf bytes.Buffer
			io.Copy(&outBuf, r)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, outBuf.String())
			}
		})
	}
}
func TestSortContainers(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		containers []container.Container
		expected   []containerReport
	}{
		{
			name: "single container with names",
			containers: []container.Container{
				{
					ID:      "1234567890abcdef",
					Created: now.Add(-time.Hour),
					ImageID: "image1",
					LayerID: "layer1",
					Names:   []string{"container1", "container2"},
				},
			},
			expected: []containerReport{
				{
					Created: "About an hour ago",
					Names:   "container1",
					ImageID: "image1",
					LayerID: "layer1",
					ID:      "1234567890",
				},
			},
		},
		{
			name: "single container with no names",
			containers: []container.Container{
				{
					ID:      "abcdef1234567890",
					Created: now.Add(-time.Hour * 2),
					ImageID: "image2",
					LayerID: "layer2",
					Names:   []string{},
				},
			},
			expected: []containerReport{
				{
					Created: "2 hours ago",
					ID:      "abcdef1234",
					ImageID: "image2",
					LayerID: "layer2",
				},
			},
		},
		// 可以添加更多测试用例
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sortContainers(tt.containers)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
func TestJsonFormatImages(t *testing.T) {
	now := time.Now()
	images := []imagemanager.Image{
		{
			OriImage: storage.Image{
				ID:      "1234567890abcdef",
				Digest:  digest.FromString("sha256:abcdef1234567890"),
				Created: now.Add(-time.Hour),
				Names:   []string{"image1"},
			},
			Size: 2048,
		},
		{
			OriImage: storage.Image{
				ID:      "abcdef1234567890",
				Digest:  digest.FromString("sha256:1234567890abcdef"),
				Created: now.Add(-time.Hour * 2),
				Names:   []string{"image2"},
			},
			Size: 4096,
		},
	}

	tests := []struct {
		name      string
		ops       options.ImagesOption
		expected  string
		expectErr bool
	}{
		{
			name: "valid images",
			ops:  options.ImagesOption{}, // 这里可以填入需要的选项
			expected: `[
    {
        "name": [
            "image1"
        ],
        "digest": "sha256:c388bcbb21ce73d23ce515dcebced3b7a13d6116c5536d529e3ea8e1b1c87984",
        "images ID": "1234567890abcdef",
        "created": "` + now.Add(-time.Hour).Format(time.RFC3339Nano) + `",
        "size": 2048
    },
    {
        "name": [
            "image2"
        ],
        "digest": "sha256:d9be5dfca5c3189b0d8b6aea35bd30d91264606a1bf286b2fe8849aad91613b4",
        "images ID": "abcdef1234567890",
        "created": "` + now.Add(-time.Hour*2).Format(time.RFC3339Nano) + `",
        "size": 4096
    }
]`,
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original stdout
			oldStdout := os.Stdout
			defer func() { os.Stdout = oldStdout }()

			r, w, _ := os.Pipe()
			os.Stdout = w

			err := JsonFormatImages(images, tt.ops)

			w.Close()
			var outBuf bytes.Buffer
			io.Copy(&outBuf, r)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Clean up the output for comparison
				var actualOutput interface{}
				var expectedOutput interface{}

				json.Unmarshal(outBuf.Bytes(), &actualOutput)
				json.Unmarshal([]byte(tt.expected), &expectedOutput)

				assert.Equal(t, expectedOutput, actualOutput)
			}
		})
	}
}
func TestFormatBuilders(t *testing.T) {
	now := time.Now()
	containers := []container.Container{
		{
			ID:      "1234567890abcdef",
			Created: now.Add(-time.Hour),
			ImageID: "image1",
			LayerID: "layer1",
			Names:   []string{"container1"},
		},
		{
			ID:      "abcdef1234567890",
			Created: now.Add(-time.Hour * 2),
			ImageID: "image2",
			LayerID: "layer2",
			Names:   []string{"container2"},
		},
	}
	tests := []struct {
		name      string
		ops       options.BuildersOption
		expected  string
		expectErr bool
	}{
		{
			name: "valid containers",
			ops:  options.BuildersOption{},
			expected: `[
    {
        "name": [
            "container1"
        ],
        "digest": "sha256:c388bcbb21ce73d23ce515dcebced3b7a13d6116c5536d529e3ea8e1b1c87984",
        "images ID": "image1",
		"layer ID": "layer1",
        "created": "` + now.Add(-time.Hour).Format(time.RFC3339Nano) + `"
    },
    {
        "name": [
            "container2"
        ],
        "digest": "sha256:d9be5dfca5c3189b0d8b6aea35bd30d91264606a1bf286b2fe8849aad91613b4",
        "images ID": "image2",
		"layer ID": "layer2",
        "created": "` + now.Add(-time.Hour*2).Format(time.RFC3339Nano) + `",
    }
]
`,
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			defer func() { os.Stdout = oldStdout }()
			r, w, _ := os.Pipe()
			os.Stdout = w
			err := FormatBuilders(containers, tt.ops)
			w.Close()
			var outBuf bytes.Buffer
			io.Copy(&outBuf, r)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Clean up the output for comparison
				var actualOutput interface{}
				var expectedOutput interface{}

				json.Unmarshal(outBuf.Bytes(), &actualOutput)
				json.Unmarshal([]byte(tt.expected), &expectedOutput)

				assert.Equal(t, expectedOutput, actualOutput)
			}
		})
	}
}
func TestJsonFormatBuilders(t *testing.T) {
	now := time.Now()
	containers := []container.Container{
		{
			ID:      "1234567890abcdef",
			Created: now.Add(-time.Hour),
			ImageID: "image1",
			Names:   []string{"container1"},
		},
		{
			ID:      "abcdef1234567890",
			Created: now.Add(-time.Hour * 2),
			ImageID: "image2",
			Names:   []string{"container2"},
		},
	}
	tests := []struct {
		name      string
		ops       options.BuildersOption
		expected  string
		expectErr bool
	}{
		{
			name: "valid containers",
			ops:  options.BuildersOption{}, // 这里可以填入需要的选项
			expected: `[
    {
        "names": [
            "container1"
        ],
		"id": "1234567890abcdef",
        "imageID": "image1",
        "created": "` + now.Add(-time.Hour).Format(time.RFC3339Nano) + `"
    },
    {
        "names": [
            "container2"
        ],
        "id": "abcdef1234567890",
		"imageID": "image2",
        "created": "` + now.Add(-time.Hour*2).Format(time.RFC3339Nano) + `"
    }
]`,
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			defer func() { os.Stdout = oldStdout }()
			r, w, _ := os.Pipe()
			os.Stdout = w
			err := JsonFormatBuilders(containers, tt.ops)
			w.Close()
			var outBuf bytes.Buffer
			io.Copy(&outBuf, r)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Clean up the output for comparison
				var actualOutput interface{}
				var expectedOutput interface{}
				json.Unmarshal(outBuf.Bytes(), &actualOutput)
				json.Unmarshal([]byte(tt.expected), &expectedOutput)
				assert.Equal(t, expectedOutput, actualOutput)
			}
		})
	}
}
func TestJsonFormatMountInfo(t *testing.T) {
	builders := []*builder.Builder{
		{
			ID:          "1234567890abcdef",
			MountPoint:  "/mnt/point1",
			FromImageID: "image1",
		},
		{
			ID:          "abcdef1234567890",
			MountPoint:  "/mnt/point2",
			FromImageID: "image2",
		},
	}
	// 预期的 JSON 输出
	expectedJSON := `[
    {
        "id": "1234567890abcdef",
        "mount": "/mnt/point1",
        "imageID": "image1",
		"created": "0001-01-01T00:00:00Z"
    },
    {
        "id": "abcdef1234567890",
        "mount": "/mnt/point2",
        "imageID": "image2",
		"created": "0001-01-01T00:00:00Z"
    }
]`
	// 捕获标准输出
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 调用被测函数
	err := JsonFormatMountInfo(builders)
	w.Close()

	// 捕获输出
	var outBuf bytes.Buffer
	io.Copy(&outBuf, r)

	// 检查错误
	assert.NoError(t, err)

	// 比较实际输出和预期输出
	actualJSON := outBuf.String()
	var actualOutput interface{}
	var expectedOutput interface{}
	errActual := json.Unmarshal([]byte(actualJSON), &actualOutput)
	errExpected := json.Unmarshal([]byte(expectedJSON), &expectedOutput)
	if errActual != nil {
		t.Fatalf("Failed to unmarshal actual output: %v", errActual)
	}
	if errExpected != nil {
		t.Fatalf("Failed to unmarshal expected output: %v", errExpected)
	}
	assert.Equal(t, expectedOutput, actualOutput)
}
