package imagemanager

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/common/pkg/config"
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

func (im *ImageManager) Logout(args []string) error {
	var logoutOps *auth.LogoutOptions
	logoutOps = &auth.LogoutOptions{
		Stdout:                    os.Stdout,
		AcceptUnspecifiedRegistry: true,
		AcceptRepositories:        true,
	}
	sctx := &types.SystemContext{
		AuthFilePath: logoutOps.AuthFile,
	}
	return auth.Logout(sctx, logoutOps, args)
}

func (im *ImageManager) Pull(imageName string) error {
	runtime := im.Manager
	ctx := context.Background()
	pullPolicy, err := config.ParsePullPolicy("always")
	if err != nil {
		return err
	}
	pullOptions := &libimage.PullOptions{}
	images, err := runtime.Pull(ctx, imageName, pullPolicy, pullOptions)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", images[0].ID())
	return nil
}

func (im *ImageManager) Push(args []string) error {
	runtime := im.Manager
	pushOptions := &libimage.PushOptions{}
	image := args[0]
	destination := args[len(args)-1]
	_, err := runtime.Push(context.Background(), image, destination, pushOptions)
	if err != nil {
		return err
	}
	fmt.Println("Successfully")
	return nil
}

func (im *ImageManager) Remove(image []string, op options.RemoveOption) error {
	runtime := im.Manager
	rmOptions := &libimage.RemoveImagesOptions{}
	rmOptions.Force = op.Force
	_, errs := runtime.RemoveImages(context.Background(), image, rmOptions)
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (im *ImageManager) Tag(store storage.Store, args []string) error {
	name := args[0]
	if !store.Exists(name) {
		err := errors.New("image not exist")
		return err
	}
	err := store.AddNames(name, args[1:])
	if err != nil {
		return err
	}
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
