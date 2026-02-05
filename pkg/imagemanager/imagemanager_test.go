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
package imagemanager

import (
	"testing"

	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageManager(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		options, err := storage.DefaultStoreOptions(unshare.GetRootlessUID() > 0, unshare.GetRootlessUID())
		store, err := storage.GetStore(options)
		im, err := NewImageManager(store)
		require.NoError(t, err)
		assert.NotNil(t, im)
		assert.NotNil(t, im.Manager)
	})
}

func TestImage(t *testing.T) {
	t.Run("create new image", func(t *testing.T) {
		oriImage := storage.Image{
			// Set some sample data for the original image
		}
		image := Image{
			OriImage: oriImage,
			Size:     123456,
		}

		assert.Equal(t, oriImage, image.OriImage)
		assert.Equal(t, int64(123456), image.Size)
	})
}

func TestJoin(t *testing.T) {
	type args struct {
		base     map[string]string
		override map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "base is nil",
			args: args{
				base:     nil,
				override: map[string]string{"k1": "v1"},
			},
			want: map[string]string{"k1": "v1"},
		},
		{
			name: "base is empty",
			args: args{
				base:     map[string]string{},
				override: map[string]string{"k1": "v1"},
			},
			want: map[string]string{"k1": "v1"},
		},
		{
			name: "override is nil",
			args: args{
				base:     map[string]string{"k1": "v1"},
				override: nil,
			},
			want: map[string]string{"k1": "v1"},
		},
		{
			name: "merge disjoint",
			args: args{
				base:     map[string]string{"k1": "v1"},
				override: map[string]string{"k2": "v2"},
			},
			want: map[string]string{"k1": "v1", "k2": "v2"},
		},
		{
			name: "override existing key",
			args: args{
				base:     map[string]string{"k1": "v1", "k2": "v2"},
				override: map[string]string{"k2": "new_v2", "k3": "v3"},
			},
			want: map[string]string{"k1": "v1", "k2": "new_v2", "k3": "v3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Join(tt.args.base, tt.args.override)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractRegistryFromImageName(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		want      string
	}{
		{
			name:      "docker.io full reference",
			imageName: "docker.io/library/alpine:latest",
			want:      "docker.io",
		},
		{
			name:      "short name implies docker.io",
			imageName: "alpine",
			want:      "docker.io",
		},
		{
			name:      "private registry",
			imageName: "myregistry.local:5000/image:tag",
			want:      "myregistry.local:5000",
		},
		{
			name:      "invalid image name",
			imageName: "!!!",
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRegistryFromImageName(tt.imageName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetRegistriesConfPath(t *testing.T) {
	tests := []struct {
		name        string
		initialPath string
		envVars     map[string]string
		wantPath    string
	}{
		{
			name:        "already set",
			initialPath: "/etc/containers/registries.conf",
			envVars:     map[string]string{},
			wantPath:    "/etc/containers/registries.conf",
		},
		{
			name:        "set from CONTAINERS_REGISTRIES_CONF",
			initialPath: "",
			envVars:     map[string]string{"CONTAINERS_REGISTRIES_CONF": "/tmp/reg1.conf"},
			wantPath:    "/tmp/reg1.conf",
		},
		{
			name:        "set from REGISTRIES_CONFIG_PATH",
			initialPath: "",
			envVars:     map[string]string{"REGISTRIES_CONFIG_PATH": "/tmp/reg2.conf"},
			wantPath:    "/tmp/reg2.conf",
		},
		{
			name:        "CONTAINERS_REGISTRIES_CONF priority",
			initialPath: "",
			envVars: map[string]string{
				"CONTAINERS_REGISTRIES_CONF": "/tmp/prio.conf",
				"REGISTRIES_CONFIG_PATH":     "/tmp/ignored.conf",
			},
			wantPath: "/tmp/prio.conf",
		},
		{
			name:        "no env vars",
			initialPath: "",
			envVars:     map[string]string{},
			wantPath:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			sysCtx := &types.SystemContext{
				SystemRegistriesConfPath: tt.initialPath,
			}
			SetRegistriesConfPath(sysCtx)
			assert.Equal(t, tt.wantPath, sysCtx.SystemRegistriesConfPath)
		})
	}
}
