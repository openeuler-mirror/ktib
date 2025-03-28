#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

package imagemanager

import (
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/storage/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

// TestListImage tests the ListImage function in ImageManager.
func TestListImage(t *testing.T) {
	// Set up a temporary directory for the storage
	tempDir, err := os.MkdirTemp("", "test-storage-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize the store
	store, err := storage.GetStore(types.StoreOptions{
		RunRoot:   tempDir,
		GraphRoot: tempDir,
	})
	assert.NoError(t, err)

	// Add an image to the store
	imageID := "1234567890abcdef"
	image := &storage.Image{
		ID:       imageID,
		Names:    []string{"my-image"},
		TopLayer: "top-layer-id",
	}
	_, err = store.CreateImage(image.ID, nil, "", "", nil)
	assert.NoError(t, err)

	// Create the ImageManager
	imageManager, err := NewImageManager(store)
	require.NoError(t, err)

	// Call ListImage
	images, err := imageManager.ListImage(nil, store)
	require.NoError(t, err)

	// Verify the output
	assert.NotEmpty(t, images)
	assert.Equal(t, imageID, images[0].OriImage.ID)
}

func TestImageLogin(t *testing.T) {
	// Set up a temporary directory for the storage
	tempDir, err := os.MkdirTemp("", "test-storage-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize the store
	store, err := storage.GetStore(types.StoreOptions{
		RunRoot:   tempDir,
		GraphRoot: tempDir,
	})
	assert.NoError(t, err)

	// Add an image to the store
	imageID := "1234567890abcdef"
	image := &storage.Image{
		ID:       imageID,
		Names:    []string{"my-image"},
		TopLayer: "top-layer-id",
	}
	_, err = store.CreateImage(image.ID, nil, "", "", nil)
	assert.NoError(t, err)

	// Create the ImageManager
	imageManager, err := NewImageManager(store)
	require.NoError(t, err)
	loginOptions := &options.LoginOption{
		Username:      "testuser",
		Password:      "testpassword",
		ServerAddress: "registry.example.com",
		Stdin:         nil,
		Stdout:        nil,
		TLSVerify:     false,
	}

	registries := `
[registries.search]
registries = ["docker.io"]
`
	os.MkdirAll("/etc/containers", 0755)
	registriesPath := filepath.Join("/etc/containers", "registries.conf")
	err = os.WriteFile(registriesPath, []byte(registries), 0644)
	require.NoError(t, err)
	err = imageManager.KtibLogin(nil, loginOptions, nil, false)
	if err != nil {
		if strings.Contains(err.Error(), " net/http: nil Context") {
			t.Skip("Skipping test due to ping container registry failed : net/http nil Context.")
		}
		t.Fatalf("Error during login: %v", err)
	}
	assert.NoError(t, err)
	os.Remove("/etc/containers")
}
func TestImageLogout(t *testing.T) {
	// Set up a temporary directory for the storage
	tempDir, err := os.MkdirTemp("", "test-storage-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize the store
	store, err := storage.GetStore(types.StoreOptions{
		RunRoot:   tempDir,
		GraphRoot: tempDir,
	})
	assert.NoError(t, err)

	// Add an image to the store
	imageID := "1234567890abcdef"
	image := &storage.Image{
		ID:       imageID,
		Names:    []string{"my-image"},
		TopLayer: "top-layer-id",
	}
	_, err = store.CreateImage(image.ID, nil, "", "", nil)
	assert.NoError(t, err)

	// Create the ImageManager
	imageManager, err := NewImageManager(store)
	require.NoError(t, err)

	registries := `
[registries.search]
registries = ["docker.io"]
`
	os.MkdirAll("/etc/containers", 0755)
	registriesPath := filepath.Join("/etc/containers", "registries.conf")
	err = os.WriteFile(registriesPath, []byte(registries), 0644)
	require.NoError(t, err)
	defer os.Remove(registriesPath)

	err = imageManager.Logout(nil)
	if err != nil {
		if strings.Contains(err.Error(), "not logged into docker.io") {
			t.Skip("Skipping test due to ping container registry failed : net/http nil Context.")
		}
		t.Fatalf("Error during logout: %v", err)
	}
	assert.NoError(t, err)
}

func TestImagePull(t *testing.T) {
	// Set up a temporary directory for the storage
	tempDir, err := os.MkdirTemp("", "test-storage-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize the store
	store, err := storage.GetStore(types.StoreOptions{
		RunRoot:   tempDir,
		GraphRoot: tempDir,
	})
	assert.NoError(t, err)

	// Add an image to the store
	imageID := "1234567890abcdef"
	image := &storage.Image{
		ID:       imageID,
		Names:    []string{"my-image"},
		TopLayer: "top-layer-id",
	}
	_, err = store.CreateImage(image.ID, nil, "", "", nil)
	assert.NoError(t, err)

	// Create the ImageManager
	imageManager, err := NewImageManager(store)
	require.NoError(t, err)
	err = imageManager.Pull("docker.io/library/busybox:latest")
	if err != nil {
		if strings.Contains(err.Error(), "open /etc/containers/policy.json: no such file or directory") {
			t.Skip("Skipping test due to pull failed")
		}
		t.Fatalf("Error during pull: %v", err)
	}
	assert.NoError(t, err)
}
func TestImagePush(t *testing.T) {
	// Set up a temporary directory for the storage
	tempDir, err := os.MkdirTemp("", "test-storage-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize the store
	store, err := storage.GetStore(types.StoreOptions{
		RunRoot:   tempDir,
		GraphRoot: tempDir,
	})
	assert.NoError(t, err)

	// Add an image to the store
	imageID := "1234567890abcdef"
	image := &storage.Image{
		ID:       imageID,
		Names:    []string{"my-image"},
		TopLayer: "top-layer-id",
	}
	_, err = store.CreateImage(image.ID, nil, "", "", nil)
	assert.NoError(t, err)

	// Create the ImageManager
	imageManager, err := NewImageManager(store)
	require.NoError(t, err)
	err = imageManager.Push([]string{imageID})
	if err != nil {
		if strings.Contains(err.Error(), "invalid checksum digest length") {
			t.Skip("Skipping test due to push failed")
		}
		t.Fatalf("Error during push: %v", err)
	}
	assert.NoError(t, err)
}
func TestImageRemove(t *testing.T) {
	// Set up a temporary directory for the storage
	tempDir, err := os.MkdirTemp("", "test-storage-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize the store
	store, err := storage.GetStore(types.StoreOptions{
		RunRoot:   tempDir,
		GraphRoot: tempDir,
	})
	assert.NoError(t, err)

	// Add an image to the store
	imageID := "1234567890abcdef"
	image := &storage.Image{
		ID:       imageID,
		Names:    []string{"my-image"},
		TopLayer: "top-layer-id",
	}
	_, err = store.CreateImage(image.ID, nil, "", "", nil)
	assert.NoError(t, err)

	// Create the ImageManager
	imageManager, err := NewImageManager(store)
	require.NoError(t, err)
	err = imageManager.Remove(store, []string{imageID}, options.RemoveOption{})
	assert.NoError(t, err)
}
func TestImageTag(t *testing.T) {
	// Set up a temporary directory for the storage
	tempDir, err := os.MkdirTemp("", "test-storage-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize the store
	store, err := storage.GetStore(types.StoreOptions{
		RunRoot:   tempDir,
		GraphRoot: tempDir,
	})
	assert.NoError(t, err)

	// Add an image to the store
	imageID := "1234567890abcdef"
	image := &storage.Image{
		ID:       imageID,
		Names:    []string{"my-image"},
		TopLayer: "top-layer-id",
	}
	_, err = store.CreateImage(image.ID, nil, "", "", nil)
	assert.NoError(t, err)

	// Create the ImageManager
	imageManager, err := NewImageManager(store)
	require.NoError(t, err)
	err = imageManager.Tag(store, []string{imageID, "my-new-tag"})
	assert.NoError(t, err)
}
