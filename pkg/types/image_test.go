#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

package types

import (
	"github.com/opencontainers/go-digest"
	"reflect"
	"testing"
	"time"
)

func TestJsonImage(t *testing.T) {
	tests := []struct {
		name            string
		input           JsonImage
		expectedName    []string
		expectedID      string
		expectedSize    int64
		expectedCreated time.Time
	}{
		{
			name: "Test with valid data",
			input: JsonImage{
				Name:    []string{"image1", "image2"},
				Digest:  digest.FromString("test-digest"),
				ImageID: "1234567890abcdef",
				Created: time.Date(0001, time.January, 1, 0, 0, 0, 0, time.UTC),
				Size:    1024,
			},
			expectedName:    []string{"image1", "image2"},
			expectedID:      "1234567890abcdef",
			expectedSize:    1024,
			expectedCreated: time.Date(0001, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		// 可以添加更多的测试案例
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotName := tt.input.Name; !reflect.DeepEqual(gotName, tt.expectedName) {
				t.Errorf("JsonImage.Name = %v, want %v", gotName, tt.expectedName)
			}
			if gotID := tt.input.ImageID; gotID != tt.expectedID {
				t.Errorf("JsonImage.ImageID = %v, want %v", gotID, tt.expectedID)
			}
			if gotSize := tt.input.Size; gotSize != tt.expectedSize {
				t.Errorf("JsonImage.Size = %v, want %v", gotSize, tt.expectedSize)
			}
			if gotCreated := tt.input.Created; !reflect.DeepEqual(gotCreated, tt.expectedCreated) {
				t.Errorf("JsonImage.Created = %v, want %v", gotCreated, tt.expectedCreated)
			}
		})
	}
}
func TestTableImage(t *testing.T) {
	tests := []struct {
		name               string
		input              TableImage
		expectedDigest     string
		expectedImageID    string
		expectedCreated    string
		expectedSize       string
		expectedRepository string
		expectedTag        string
	}{
		{
			name: "Test with valid data",
			input: TableImage{
				Repository: "ubuntu",
				Tag:        "latest",
				ImageID:    "1234567890abcdef",
				Created:    "2023-01-01 00:00:00",
				Size:       "1024",
				Digest:     "sha256:test-digest",
			},
			expectedDigest:     "sha256:test-digest",
			expectedImageID:    "1234567890abcdef",
			expectedCreated:    "2023-01-01 00:00:00",
			expectedSize:       "1024",
			expectedRepository: "ubuntu",
			expectedTag:        "latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotDigest := tt.input.Digest; gotDigest != tt.expectedDigest {
				t.Errorf("TableImage.Digest = %v, want %v", gotDigest, tt.expectedDigest)
			}
			if gotImageID := tt.input.ImageID; gotImageID != tt.expectedImageID {
				t.Errorf("TableImage.ImageID = %v, want %v", gotImageID, tt.expectedImageID)
			}
			if gotCreated := tt.input.Created; gotCreated != tt.expectedCreated {
				t.Errorf("TableImage.Created = %v, want %v", gotCreated, tt.expectedCreated)
			}
			if gotSize := tt.input.Size; gotSize != tt.expectedSize {
				t.Errorf("TableImage.Size = %v, want %v", gotSize, tt.expectedSize)
			}
			if gotRepository := tt.input.Repository; gotRepository != tt.expectedRepository {
				t.Errorf("TableImage.Repository = %v, want %v", gotRepository, tt.expectedRepository)
			}
		})
	}
}
func TestJsonBuilder(t *testing.T) {
	tests := []struct {
		name            string
		input           JsonBuilder
		expectedID      string
		expectedImageID string
		expectedNames   []string
		expectedCreated time.Time
		expectedMount   string
	}{
		{
			name: "Test with valid data",
			input: JsonBuilder{
				ID:      "1234567890abcdef",
				Names:   []string{"image1", "image2"},
				ImageID: "1234567890abcdef",
				Created: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				Mount:   "test-mount",
			},
			expectedID:      "1234567890abcdef",
			expectedImageID: "1234567890abcdef",
			expectedNames:   []string{"image1", "image2"},
			expectedCreated: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
			expectedMount:   "test-mount",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotID := tt.input.ID; gotID != tt.expectedID {
				t.Errorf("JsonBuilder.ID = %v, want %v", gotID, tt.expectedID)
			}
			if gotImageID := tt.input.ImageID; gotImageID != tt.expectedImageID {
				t.Errorf("JsonBuilder.ImageID = %v, want %v", gotImageID, tt.expectedImageID)
			}
			if gotNames := tt.input.Names; !reflect.DeepEqual(gotNames, tt.expectedNames) {
				t.Errorf("JsonBuilder.Names = %v, want %v", gotNames, tt.expectedNames)
			}
			if gotCreated := tt.input.Created; !reflect.DeepEqual(gotCreated, tt.expectedCreated) {
				t.Errorf("JsonBuilder.Created = %v, want %v", gotCreated, tt.expectedCreated)
			}
			if gotMount := tt.input.Mount; gotMount != tt.expectedMount {
				t.Errorf("JsonBuilder.Mount = %v, want %v", gotMount, tt.expectedMount)
			}
		})
	}

}
