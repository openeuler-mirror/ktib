package imagemanager

import (
	"context"
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"os"
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
func (im *ImageManager) KtibLogin(ctx context.Context, lops *options.LoginOption, args []string, getLoginSet bool) error {
	var loginOps *auth.LoginOptions
	loginOps = &auth.LoginOptions{
		Password:                  lops.Password,
		Username:                  lops.Username,
		StdinPassword:             lops.PasswordStdin,
		GetLoginSet:               true,
		Stdin:                     os.Stdin,
		Stdout:                    os.Stdout,
		AcceptRepositories:        true,
		AcceptUnspecifiedRegistry: true,
	}
	sctx := &types.SystemContext{
		AuthFilePath:                      loginOps.AuthFile,
		DockerCertPath:                    loginOps.CertDir,
		DockerDaemonInsecureSkipTLSVerify: lops.TLSVerify,
	}
	setRegistriesConfPath(sctx)
	loginOps.GetLoginSet = getLoginSet
	return auth.Login(ctx, sctx, loginOps, args)
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

func setRegistriesConfPath(systemContext *types.SystemContext) {
	if systemContext.SystemRegistriesConfPath != "" {
		return
	}
	if envOverride, ok := os.LookupEnv("CONTAINERS_REGISTRIES_CONF"); ok {
		systemContext.SystemRegistriesConfPath = envOverride
		return
	}
	if envOverride, ok := os.LookupEnv("REGISTRIES_CONFIG_PATH"); ok {
		systemContext.SystemRegistriesConfPath = envOverride
		return
	}
}
