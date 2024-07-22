package imagemanager

import (
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
