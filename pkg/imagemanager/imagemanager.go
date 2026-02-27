/*
Copyright (c) 2024 KylinSoft Co., Ltd.
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
	"time"

	"gitee.com/openeuler/ktib/pkg/options"
	types2 "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/common/pkg/config"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	auth_config "github.com/containers/image/v5/pkg/docker/config"
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

	// Parsed name information
	ParsedNames []ParsedImageName
}

// Parsed image name structure
type ParsedImageName struct {
	Repository string // Repository name
	Tag        string // Tag
	// Digest     string // Digest
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
	// Iterate over the fetched image data
	for _, img := range images {
		// Get the actual image data from the store
		storageImg := img.StorageImage()

		size, err := store.ImageSize(img.ID())
		if err != nil {
			return nil, err
		}

		// Create an Image instance and populate the data
		ktibImage := &Image{
			OriImage: *storageImg, // storage.Image direct assignment
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
		AuthFilePath:   loginOps.AuthFile,
		DockerCertPath: loginOps.CertDir,
		// Fix: TLS verification flag semantic reversal issue: when lops.TLSVerify is true (verification required), skip verification should be false
		DockerDaemonInsecureSkipTLSVerify: !lops.TLSVerify,
	}

	// Set insecure parameter
	if lops.Insecure {
		sctx.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}

	SetRegistriesConfPath(sctx)
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
	// enable progress bars on TTY and set retry defaults
	pullOptions.Writer = os.Stderr
	mr := uint(3)
	pullOptions.MaxRetries = &mr
	rd := 2 * time.Second
	pullOptions.RetryDelay = &rd

	// Add settings for SystemContext
	if pullOptions.SystemContext == nil {
		pullOptions.SystemContext = &types.SystemContext{}
	}
	SetRegistriesConfPath(pullOptions.SystemContext)

	// Get logged-in authentication information
	credentials, err := auth_config.GetAllCredentials(pullOptions.SystemContext)
	if err != nil || len(credentials) == 0 {
		// Use default TLS verification setting when no authentication information is present
		pullOptions.InsecureSkipTLSVerify = types.OptionalBoolFalse
	} else {
		// If authentication information is present, check if the image registry matches
		imageRegistry := extractRegistryFromImageName(imageName)
		matchFound := false
		if imageRegistry != "" {
			if _, exists := credentials[imageRegistry]; exists {
				matchFound = true
			}
		}

		if matchFound {
			// Image is from a logged-in registry
			pullOptions.InsecureSkipTLSVerify = types.OptionalBoolTrue
		} else {
			// Image is not from a logged-in registry
			pullOptions.InsecureSkipTLSVerify = types.OptionalBoolFalse
		}
	}

	// normalize short names to fully-qualified (docker.io/library/...) to avoid short-name resolution
	if named, nerr := reference.ParseNormalizedNamed(imageName); nerr == nil {
		imageName = named.String()
	}
	images, err := runtime.Pull(ctx, imageName, pullPolicy, pullOptions)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", images[0].ID())
	return nil
}

func extractRegistryFromImageName(imageName string) string {
	// Parse image name
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return ""
	}
	return reference.Domain(ref)
}

func (im *ImageManager) Push(ctx context.Context, source, destination string, op options.PushOption) (*options.ImagePushReport, error) {
	runtime := im.Manager
	pushOptions := &libimage.PushOptions{}
	pushOptions.Password = op.Password
	pushOptions.Username = op.Username
	pushOptions.SignBy = op.SignBy
	pushOptions.Writer = os.Stderr
	// enable progress bars on TTY and set retry defaults
	mr := uint(3)
	pushOptions.MaxRetries = &mr
	rd := 2 * time.Second
	pushOptions.RetryDelay = &rd

	// Add settings for SystemContext
	if pushOptions.SystemContext == nil {
		pushOptions.SystemContext = &types.SystemContext{}
	}
	SetRegistriesConfPath(pushOptions.SystemContext)

	// If the user explicitly set the insecure parameter, prioritize the user setting
	if op.Insecure {
		pushOptions.InsecureSkipTLSVerify = types.OptionalBoolTrue
	} else {
		// Get logged-in authentication information
		credentials, err := auth_config.GetAllCredentials(pushOptions.SystemContext)
		if err != nil || len(credentials) == 0 {
			// Use default TLS verification setting when no authentication information is present
			pushOptions.InsecureSkipTLSVerify = types.OptionalBoolFalse
		} else {
			// If authentication information is present, check if the destination registry matches
			destinationRegistry := extractRegistryFromImageName(destination)
			matchFound := false
			if destinationRegistry != "" {
				if _, exists := credentials[destinationRegistry]; exists {
					matchFound = true
				}
			}

			if matchFound {
				// Destination repository is a logged-in registry
				pushOptions.InsecureSkipTLSVerify = types.OptionalBoolTrue
			} else {
				// Destination repository is not a logged-in registry
				pushOptions.InsecureSkipTLSVerify = types.OptionalBoolFalse
			}
		}
	}

	// normalize short names to fully-qualified to avoid short-name resolution
	if sn, nerr := reference.ParseNormalizedNamed(source); nerr == nil {
		source = sn.String()
	}
	if dn, nerr := reference.ParseNormalizedNamed(destination); nerr == nil {
		destination = dn.String()
	}
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
	for _, arg := range images {
		targetID := arg
		removeName := ""

		// resolve name/tag to normalized form if it's not an ID
		if !store.Exists(targetID) {
			if named, err := reference.ParseNormalizedNamed(arg); err == nil {
				removeName = named.String()
				// lookup image by name via libimage runtime to get ID
				li, _, lerr := im.Manager.LookupImage(removeName, nil)
				if lerr != nil {
					allErrors = append(allErrors, lerr)
					logrus.Errorf("no such image by name: %s", arg)
					continue
				}
				targetID = li.ID()
			} else {
				// not an ID and cannot be parsed as name
				allErrors = append(allErrors, err)
				logrus.Errorf("invalid image reference: %s", arg)
				continue
			}
		}

		names, nerr := store.Names(targetID)
		if nerr != nil {
			allErrors = append(allErrors, nerr)
			logrus.Debugf("failed to get names for image %s: %v", targetID, nerr)
			continue
		}

		// if user passed a name and image has multiple names, untag just that name
		if removeName != "" && len(names) > 1 {
			// normalize to ensure the stored name matches
			if err := store.RemoveNames(targetID, []string{removeName}); err != nil {
				allErrors = append(allErrors, err)
				logrus.Errorf("untag %s failed: %v", removeName, err)
				continue
			}
			logrus.Infof("Untagged: %s", removeName)
			continue
		}

		// otherwise delete the whole image by ID
		si, ierr := store.Image(targetID)
		if ierr != nil {
			allErrors = append(allErrors, ierr)
			logrus.Errorf("no such image: %s", arg)
			continue
		}
		if _, derr := store.DeleteImage(si.ID, true); derr != nil {
			allErrors = append(allErrors, derr)
			logrus.Error(fmt.Sprintf("unable to remove image '%s': %s", arg, derr))
			continue
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
		return fmt.Errorf("image not exist: %s", name)
	}
	for i, arg := range args[1:] {
		if strings.HasSuffix(arg, ":") {
			return fmt.Errorf("Error parsing reference: %s is not a valid repository/tag: invalid reference format", arg)
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

func SetRegistriesConfPath(systemContext *types.SystemContext) {
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
	// In theory, the following tags for saveOptions should be assigned based on saveOptions, but currently save does not support these flags, so they are set to default values for now
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
	// In theory, it should be assigned from the load options, but currently these parameters are not supported, so they are temporarily written as defaults
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
	SetRegistriesConfPath(sysCtx)
	// Add credential matching logic
	allCreds, err := auth_config.GetAllCredentials(sysCtx)
	if err == nil {
		for _, image := range images {
			registryDomain := extractRegistryFromImageName(image)
			for registry := range allCreds {
				if registry == registryDomain {
					addOptions.InsecureSkipTLSVerify = types.OptionalBoolTrue
					break
				}
			}
		}
	}
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
	// todo：The following parameters are currently unsupported and set to default values. They will be supplemented later as needed
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

	// Supported parameter assignment
	pushOptions.Password = op.Password
	pushOptions.Username = op.Username
	pushOptions.SignBy = op.SignBy
	pushOptions.Writer = os.Stderr
	// enable progress bars on TTY and set retry defaults
	mr := uint(3)
	pushOptions.MaxRetries = &mr
	rd := 2 * time.Second
	pushOptions.RetryDelay = &rd
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

	// Add settings for SystemContext
	if pushOptions.SystemContext == nil {
		pushOptions.SystemContext = &types.SystemContext{}
	}
	SetRegistriesConfPath(pushOptions.SystemContext)

	// If the user explicitly set the insecure parameter, prioritize the user setting
	if op.Insecure {
		pushOptions.InsecureSkipTLSVerify = types.OptionalBoolTrue
	} else {
		// Get logged-in authentication information
		credentials, err := auth_config.GetAllCredentials(pushOptions.SystemContext)
		if err != nil || len(credentials) == 0 {
			// Use default TLS verification setting when no authentication information is present
			pushOptions.InsecureSkipTLSVerify = types.OptionalBoolFalse
		} else {
			// If authentication information is present, check if the destination registry matches
			destinationRegistry := extractRegistryFromImageName(destination)
			matchFound := false
			if destinationRegistry != "" {
				if _, exists := credentials[destinationRegistry]; exists {
					matchFound = true
				}
			}

			if matchFound {
				// Destination repository is a logged-in registry
				pushOptions.InsecureSkipTLSVerify = types.OptionalBoolTrue
			} else {
				// Destination repository is not a logged-in registry
				pushOptions.InsecureSkipTLSVerify = types.OptionalBoolFalse
			}
		}
	}

	pushOptions.ManifestMIMEType = manifestType

	// normalize destination short name
	if dn, nerr := reference.ParseNormalizedNamed(destination); nerr == nil {
		destination = dn.String()
	}
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

	// Add settings for SystemContext
	sysCtx := &types.SystemContext{}
	SetRegistriesConfPath(sysCtx)

	// Get logged-in authentication information
	credentials, err := auth_config.GetAllCredentials(sysCtx)
	insecureSkipTLS := types.OptionalBoolFalse
	if err == nil && len(credentials) > 0 {
		// If authentication information is present, check if the registry of all images matches
		matchFound := false
		for _, image := range images {
			imageRegistry := extractRegistryFromImageName(image)
			if imageRegistry != "" {
				if _, exists := credentials[imageRegistry]; exists {
					matchFound = true
					break
				}
			}
		}

		if matchFound {
			// At least one image is from a logged-in registry
			insecureSkipTLS = types.OptionalBoolTrue
		}
	}

	// If the user explicitly set the insecure parameter, prioritize the user setting
	if opts.Insecure {
		insecureSkipTLS = types.OptionalBoolTrue
	}

	addOptions := &libimage.ManifestListAddOptions{
		All:                   false,
		AuthFilePath:          "",
		CertDirPath:           "",
		InsecureSkipTLSVerify: insecureSkipTLS,
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
	// Look up the image, note that three return values are required here: image, resolvedName, err
	image, _, err := im.Manager.LookupImage(name, nil)
	if err != nil {
		return nil, err
	}

	// Set inspect options, including calculating image size and parent image
	inspectOptions := &libimage.InspectOptions{
		WithSize:   true,
		WithParent: true,
	}

	// Call libimage's Inspect method to get image data
	imageData, err := image.Inspect(ctx, inspectOptions)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}
