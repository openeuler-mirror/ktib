package imagemanager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

type ImageManager struct {
	//TODO: 需要补充
	Manager *libimage.Runtime
}

type Image struct {
	OriImage storage.Image
	Size     int64
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

func (im *ImageManager) ListImage(args []string, store storage.Store) ([]Image, error) {
	var imageList []Image
	image, err := store.Images()
	if err != nil {
		return nil, err
	}
	for _, img := range image {
		size, err := store.ImageSize(img.ID)
		if err != nil {
			return nil, err
		}
		imageList = append(imageList, Image{
			OriImage: img,
			Size:     size,
		})
	}
	return imageList, nil
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

func (im *ImageManager) Remove(store storage.Store, images []string, op options.RemoveOption) error {
	var allErrors []error
	for i, img := range images {
		// If more than one tag exists for the image, the Untag operation is performed
		names, err := store.Names(img)
		im, err := store.Image(img)
		if err != nil {
			logrus.Errorf("No such image: %s", img)
			continue
		}
		if len(names) > 1 {
			if err := store.RemoveNames(img, images[i:i+1]); err != nil {
				logrus.Errorf("Untaged %s failed.", img)
			}
			logrus.Infof("Untagged: %s", img)
			continue
		}
		_, err = store.DeleteImage(im.ID, true)
		if err != nil {
			allErrors = append(allErrors, err)
			logrus.Error(fmt.Sprintf("unable to remove repository reference '%s': %s", img, err))
		}
	}
	if len(allErrors) > 0 {
		return errors.New("The remove operation failed.")
	}
	return nil
}

func (im *ImageManager) Tag(store storage.Store, args []string) error {
	name := args[0]
	if !store.Exists(name) {
		err := errors.New("image not exist")
		return err
	}
	for i, arg := range args[1:] {
		if strings.HasSuffix(arg, ":") {
			return errors.New(fmt.Sprintf("Error parsing reference: %s is not a valid repository/tag: invalid reference format", arg))
		}
		if !strings.Contains(arg, ":") {
			args[1:][i] += ":latest"
		}
	}
	for i, s := range args[1:] {
		noralName, err := reference.ParseNormalizedNamed(s)
		if err != nil {
			return err
		}
		args[i+1] = noralName.String()
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
