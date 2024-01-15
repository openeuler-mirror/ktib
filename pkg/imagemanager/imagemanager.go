package imagemanager

import (
	"context"
	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
)

type ImageManager struct {
	//TODO: 需要补充
	Manager *libimage.Runtime
}

type Image struct {
	KtibImage []*libimage.Image
}

func NewImageManager(store storage.Store) (*ImageManager, error) {
	var systemContext *types.SystemContext
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return nil, err
	}
	imageManager := &ImageManager{
		Manager: runtime,
	}
	return imageManager, nil
}

func (im *ImageManager) ListImage(args []string) (*Image, error) {
	ctx := context.Background()
	opts := &libimage.ListImagesOptions{}
	manager := im.Manager
	image, err := manager.ListImages(ctx, args, opts)
	if err != nil {
		return nil, err
	}
	images := &Image{
		KtibImage: image,
	}
	return images, nil
}

// TODO: 以下函数需要重构到这里
func (im *ImageManager) Login() error {
	return nil
}

func (im *ImageManager) Logout() error {
	return nil
}

func (im *ImageManager) Pull() error {
	return nil
}

func (im *ImageManager) Remove() error {
	return nil
}

func (im *ImageManager) Tag() error {
	return nil
}
