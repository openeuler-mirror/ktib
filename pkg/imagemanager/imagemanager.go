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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"gitee.com/openeuler/ktib/pkg/options"
	types2 "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/common/pkg/config"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

type ImageManager struct {
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

func (im *ImageManager) ListImage(ops options.ImagesOption, store storage.Store, background context.Context) ([]*Image, error) {
	listImagesOptions := &libimage.ListImagesOptions{
		Filters:     ops.Filter,
		SetListData: true,
	}
	listImagesOptions.Filters = append(listImagesOptions.Filters, "intermediate=false")
	images, err := im.Manager.ListImages(background, nil, listImagesOptions)
	if err != nil {
		return nil, err
	}
	var ktibImages []*Image
	// 遍历获取到的镜像数据
	for _, img := range images {
		// 从存储中获取镜像的实际数据
		storageImg := img.StorageImage()

		size, err := store.ImageSize(img.ID())
		if err != nil {
			return nil, err
		}

		// 创建一个 Image 实例并填充数据
		ktibImage := &Image{
			OriImage: *storageImg, //storage.Image 直接赋值
			Size:     size,
		}

		ktibImages = append(ktibImages, ktibImage)
	}

	return ktibImages, nil
}

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

	// 设置 insecure 参数
	if lops.Insecure {
		sctx.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
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

	// 添加对 SystemContext 的设置
	if pullOptions.SystemContext == nil {
		pullOptions.SystemContext = &types.SystemContext{}
	}
	setRegistriesConfPath(pullOptions.SystemContext)

	images, err := runtime.Pull(ctx, imageName, pullPolicy, pullOptions)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", images[0].ID())
	return nil
}

func (im *ImageManager) Push(ctx context.Context, source, destination string, op options.PushOption) (*options.ImagePushReport, error) {
	runtime := im.Manager
	pushOptions := &libimage.PushOptions{}
	pushOptions.Password = op.Password
	pushOptions.Username = op.Username
	pushOptions.SignBy = op.SignBy
	pushOptions.Writer = os.Stderr

	if op.Insecure {
		pushOptions.InsecureSkipTLSVerify = types.OptionalBoolTrue
	} else {
		pushOptions.InsecureSkipTLSVerify = types.OptionalBoolFalse
	}

	// 添加对 SystemContext 的设置
	if pushOptions.SystemContext == nil {
		pushOptions.SystemContext = &types.SystemContext{}
	}
	setRegistriesConfPath(pushOptions.SystemContext)

	pushedManifestBytes, pushErr := runtime.Push(context.Background(), source, destination, pushOptions)
	if pushErr == nil {
		manifestDigest, err := manifest.Digest(pushedManifestBytes)
		if err != nil {
			return nil, err
		}
		return &options.ImagePushReport{ManifestDigest: manifestDigest.String()}, nil
	}
	if _, err := im.Manager.LookupManifestList(source); err == nil {
		pushedManifestString, err := im.ManifestPush(ctx, source, destination, op)
		if err != nil {
			return nil, err
		}
		return &options.ImagePushReport{ManifestDigest: pushedManifestString}, nil
	}
	return nil, pushErr
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

func (im *ImageManager) SaveImage(ctx context.Context, op options.SaveOption, tags []string, name string) error {
	saveOptions := &libimage.SaveOptions{}
	// 理论上saveOptions的如下标签应该基于saveOptions赋值，但是目前save不支持这些flags，所以先置为默认值
	saveOptions.RemoveSignatures = true
	saveOptions.DirForceCompress = false
	saveOptions.OciAcceptUncompressedLayers = false
	saveOptions.SignaturePolicyPath = ""

	names := []string{name}
	if op.MultiImageArchive {
		names = append(names, tags...)
	} else {
		saveOptions.AdditionalTags = tags
	}

	return im.Manager.Save(ctx, names, op.Format, op.Output, saveOptions)
}

func (im *ImageManager) LoadImage(background context.Context, op options.LoadOption) (*options.ImageLoadReport, error) {
	loadOptions := &libimage.LoadOptions{}
	// 理论上应该从load的options里面赋值，但是目前不支持这些参数，暂时写成默认的
	loadOptions.SignaturePolicyPath = ""
	loadOptions.Writer = os.Stderr

	loadedImages, err := im.Manager.Load(background, op.Input, loadOptions)
	if err != nil {
		return nil, err
	}
	return &options.ImageLoadReport{Names: loadedImages}, nil
}

func (im *ImageManager) ManifestCreate(ctx context.Context, name string, images []string, op options.ManifestCreateOptions) (string, error) {
	if len(name) == 0 {
		return "", errors.New("no name specified for creating a manifest list")
	}

	manifestList, err := im.Manager.CreateManifestList(name)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateName) && op.Amend {
			amendList, amendErr := im.Manager.LookupManifestList(name)
			if amendErr != nil {
				return "", err
			}
			manifestList = amendList
		} else {
			return "", err
		}
	}

	annotateOptions := &libimage.ManifestListAnnotateOptions{}
	if len(op.Annotations) != 0 {
		annotateOptions.Annotations = op.Annotations
		if err := manifestList.AnnotateInstance("", annotateOptions); err != nil {
			return "", err
		}
	}
	addOptions := &libimage.ManifestListAddOptions{
		All:                   op.All,
		InsecureSkipTLSVerify: op.SkipTLSVerify,
	}

	sysCtx := &types.SystemContext{}
	setRegistriesConfPath(sysCtx)

	for _, image := range images {
		if _, err := manifestList.Add(ctx, image, addOptions); err != nil {
			return "", err
		}
	}

	return manifestList.ID(), nil
}

func (im *ImageManager) ManifestAnnotate(name string, image string, opts options.ManifestAnnotateOptions) (string, error) {
	manifestList, err := im.Manager.LookupManifestList(name)
	if err != nil {
		return "", err
	}
	annotateOptions := &libimage.ManifestListAnnotateOptions{
		Architecture: opts.Arch,
		Features:     opts.Features,
		OS:           opts.OS,
		OSVersion:    opts.OSVersion,
		Variant:      "",
	}
	if annotateOptions.Annotations, err = types2.MergeAnnotations(opts.Annotations, opts.Annotation); err != nil {
		return "", err
	}
	instanceDigest, err := digest.Parse(image)
	if err != nil {
		return "", fmt.Errorf(`invalid image digest "%s": %v`, image, err)
	}
	if err := manifestList.AnnotateInstance(instanceDigest, annotateOptions); err != nil {
		return "", err
	}
	return manifestList.ID(), nil
}

func (im *ImageManager) ManifestPush(background context.Context, name string, destination string, op options.PushOption) (string, error) {
	manifestList, err := im.Manager.LookupManifestList(name)
	if err != nil {
		return "", err
	}
	pushOptions := &libimage.ManifestListPushOptions{}
	compressionLevel := 0
	// todo：以下参数暂不支持，赋为默认值，后续按实际情况补充参数
	pushOptions.AuthFilePath = auth.GetDefaultAuthFile()
	pushOptions.CertDirPath = ""
	pushOptions.ImageListSelection = cp.CopyAllImages
	pushOptions.RemoveSignatures = false
	pushOptions.Signers = nil
	pushOptions.SignPassphrase = ""
	pushOptions.SignBySigstorePrivateKeyFile = ""
	pushOptions.SignSigstorePrivateKeyPassphrase = nil
	pushOptions.CompressionLevel = &compressionLevel
	pushOptions.AddCompression = []string{}
	pushOptions.ForceCompressionFormat = false

	// 已支持参数赋值
	pushOptions.Password = op.Password
	pushOptions.Username = op.Username
	pushOptions.SignBy = op.SignBy
	pushOptions.Writer = os.Stderr
	var manifestType string
	if op.Format != "" {
		switch op.Format {
		case "oci":
			manifestType = v1.MediaTypeImageManifest
		case "v2s2", "docker":
			manifestType = manifest.DockerV2Schema2MediaType
		default:
			return "", fmt.Errorf("unknown format %q. Choose one of the supported formats: 'oci' or 'v2s2'", op.Format)
		}
	}
	pushOptions.ManifestMIMEType = manifestType

	// 确保 insecure 参数正确设置到 CopyOptions 中
	if op.Insecure {
		pushOptions.InsecureSkipTLSVerify = types.OptionalBoolTrue
	} else {
		pushOptions.InsecureSkipTLSVerify = types.OptionalBoolFalse
	}
	// 添加对 SystemContext 的设置
	if pushOptions.SystemContext == nil {
		pushOptions.SystemContext = &types.SystemContext{}
	}
	setRegistriesConfPath(pushOptions.SystemContext)
	manDigest, err := manifestList.Push(background, destination, pushOptions)
	if err != nil {
		return "", err
	}
	return manDigest.String(), err
}

func (im *ImageManager) ManifestAdd(background context.Context, manifestName string, images []string, opts options.ManifestAddOptions) (string, error) {
	if len(images) < 1 {
		return "", errors.New("manifest add requires at least one image")
	}

	manifestList, err := im.Manager.LookupManifestList(manifestName)
	if err != nil {
		return "", err
	}

	addOptions := &libimage.ManifestListAddOptions{
		All:                   false,
		AuthFilePath:          "",
		CertDirPath:           "",
		InsecureSkipTLSVerify: types.NewOptionalBool(opts.Insecure),
		Username:              opts.Username,
		Password:              opts.Password,
	}

	for _, image := range images {
		instanceDigest, err := manifestList.Add(background, image, addOptions)
		if err != nil {
			return "", err
		}

		annotateOptions := &libimage.ManifestListAnnotateOptions{
			Architecture: opts.Arch,
			Features:     opts.Features,
			OS:           opts.OS,
			OSVersion:    opts.OSVersion,
			Variant:      "",
		}
		if len(opts.Annotation) != 0 {
			annotations := make(map[string]string)
			for _, annotationSpec := range opts.Annotation {
				spec := strings.SplitN(annotationSpec, "=", 2)
				if len(spec) != 2 {
					return "", fmt.Errorf("no value given for annotation %q", spec[0])
				}
				annotations[spec[0]] = spec[1]
			}
			opts.Annotations = Join(opts.Annotations, annotations)
		}
		annotateOptions.Annotations = opts.Annotations

		if err := manifestList.AnnotateInstance(instanceDigest, annotateOptions); err != nil {
			return "", err
		}
	}

	return manifestList.ID(), nil
}

func (im *ImageManager) ManifestInspect(name string) ([]byte, error) {
	manifestList, err := im.Manager.LookupManifestList(name)
	if err != nil {
		return nil, err
	}
	schema2List, err := manifestList.Inspect()
	if err != nil {
		return nil, err
	}

	rawSchema2List, err := json.Marshal(schema2List)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := json.Indent(&b, rawSchema2List, "", "    "); err != nil {
		return nil, fmt.Errorf("rendering manifest %s for display: %w", name, err)
	}
	return b.Bytes(), nil
}

func Join(base map[string]string, override map[string]string) map[string]string {
	if len(base) == 0 {
		return override
	}
	for k, v := range override {
		base[k] = v
	}
	return base
}

func (im *ImageManager) Inspect(ctx context.Context, name string) (*libimage.ImageData, error) {
	// 查找镜像，注意这里需要接收三个返回值：image, resolvedName, err
	image, _, err := im.Manager.LookupImage(name, nil)
	if err != nil {
		return nil, err
	}

	// 设置检查选项，包括计算镜像大小和父镜像
	inspectOptions := &libimage.InspectOptions{
		WithSize:   true,
		WithParent: true,
	}

	// 调用 libimage 的 Inspect 方法获取镜像数据
	imageData, err := image.Inspect(ctx, inspectOptions)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}
