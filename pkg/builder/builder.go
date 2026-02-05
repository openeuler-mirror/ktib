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

package builder

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/ktib/pkg/options"
	cpier "github.com/containers/image/v5/copy"
	v5manifest "github.com/containers/image/v5/manifest"

	//"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sirupsen/logrus"
)

const (
	stateFile            = "ktib.json"
	specFile             = "config.json"
	defaultTransport     = "containers-storage:"
	defaultruntime       = "/usr/bin/runc"
	defaultNullImageName = "none"
)

type Builder struct {
	Name        string
	ID          string
	Store       storage.Store
	FromImage   string
	FromImageID string
	Container   string
	ContainerID string
	MountPoint  string
	Maintainer  string
	EntryPoint  string
	Cmd         string
	Env         []string
	Message     string
	Manifest    v1.Manifest
	OCIv1       v1.Image
	DockerV2    v5manifest.Schema2Image
	Workdir     string
	out         io.Writer
}

func stripComments(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	lines := strings.Split(string(input), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n")
}

type BuilderOptions struct {
	FromImage  string
	Container  string
	PullPolicy bool
}

func NewBuilder(store storage.Store, options BuilderOptions) (*Builder, error) {
	var err error
	var container *storage.Container
	var optionNames []string
	if options.FromImage == "scratch" {
		options.FromImage = ""
	}
	image := options.FromImage
	name := options.Container
	coptions := storage.ContainerOptions{}
	if name != "" {
		optionNames = []string{name}
	}

	imageID := ""
	if image != "" {
		img, err := store.Image(image)
		if err != nil {
			return nil, err
		}
		imageID = img.ID
	}

	container, err = store.CreateContainer("", optionNames, imageID, "", "", &coptions)

	if err != nil {
		return nil, err
	}
	builder := &Builder{
		Name:        name,
		ID:          container.ID,
		Store:       store,
		FromImage:   image,
		FromImageID: imageID,
		Container:   name,
		ContainerID: container.ID,
	}
	if err := builder.Save(); err != nil {
		return nil, err
	}
	return builder, nil
}

func FindBuilder(store storage.Store, name string) (*Builder, error) {
	container, err := store.Container(name)
	if err != nil {
		return nil, err
	}
	cdir, err := store.ContainerDirectory(container.ID)
	if err != nil {
		return nil, err
	}
	statefile := filepath.Join(cdir, stateFile)

	buildstate, err := os.ReadFile(statefile)
	if err != nil && os.IsNotExist(err) {
		return nil, err
	}
	b := &Builder{
		Store: store,
	}
	err = json.Unmarshal(buildstate, &b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func FindAllBuilders(store storage.Store) ([]*Builder, error) {
	var bl []*Builder
	containers, err := store.Containers()
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		cdir, err := store.ContainerDirectory(container.ID)
		if err != nil {
			return nil, err
		}
		buildstate, err := os.ReadFile(filepath.Join(cdir, stateFile))
		if err != nil && os.IsNotExist(err) {
			return nil, err
		}
		b := &Builder{
			Store: store,
		}
		err = json.Unmarshal(buildstate, &b)
		if err != nil {
			return nil, err
		}
		b.Store = store
		bl = append(bl, b)
	}
	return bl, nil
}

func (b *Builder) Mount(label string) error {
	mountpoint, err := b.Store.Mount(b.ContainerID, label)
	if err != nil {
		return err
	}
	b.MountPoint = mountpoint

	err = b.Save()
	if err != nil {
		return err
	}
	return nil

}

func (b *Builder) UMount() error {
	_, err := b.Store.Unmount(b.ContainerID, false)
	if err == nil {
		b.MountPoint = ""
		err = b.Save()
	}
	return err
}

func (b *Builder) Tag(args []string) error {
	return nil
}

func (b *Builder) SetMaintainer(args string) {
	b.Maintainer = args
}

func (b *Builder) SetEntryPoint(args string) {
	b.EntryPoint = args
}

func (b *Builder) SetCmd(args string) {
	b.Cmd = args
}

func (b *Builder) SetEnv(args []string) {
	b.Env = args
}

func (b *Builder) SetMessage(args string) {
	b.Message = args
}

func (b *Builder) Remove(op options.RemoveOption) error {
	// If the submitted image name exists, the container will be removed early
	if !b.Store.Exists(b.ContainerID) {
		return nil
	}
	//b.Store.Unmount(b.ContainerID, op.Force)
	if !op.Force {
		timesMounted, err := b.Store.Mounted(b.ContainerID)
		if err != nil {
			if errors.Is(err, storage.ErrContainerUnknown) {
				logrus.Infof("Storage for container %s already removed", b.ContainerID)
				return nil
			}
			logrus.Warnf("Checking if container %q is mounted, attempting to delete: %v", b.ContainerID, err)
		}
		if timesMounted > 0 {
			return fmt.Errorf("container %q is mounted and cannot be removed without using force: %w", b.ContainerID, err)
		}
	} else if _, err := b.Store.Unmount(b.ContainerID, true); err != nil {
		if errors.Is(err, storage.ErrContainerUnknown) {
			// Container again gone, no error
			logrus.Infof("Storage for container %s already removed", b.Container)
			return nil
		}
		logrus.Warnf("Unmounting container %q while attempting to delete storage: %v", b.ContainerID, err)
	}
	if err := b.Store.DeleteContainer(b.ContainerID); err != nil {
		logrus.Error(fmt.Sprintf("delete builder failed: %s", err))
		return err
	}
	return nil
}

func (b *Builder) name() string {
	return b.Name
}

func (b *Builder) Save() error {
	buildstate, err := json.Marshal(b)
	if err != nil {
		return err
	}
	cdir, err := b.Store.ContainerDirectory(b.ContainerID)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(filepath.Join(cdir, stateFile), buildstate, 0600)
}

func (b *Builder) Commit(exportTo string) error {
	ctx := context.Background()
	systemContext := types.SystemContext{}
	policy, err := signature.DefaultPolicy(&systemContext)
	if err != nil {
		return err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return err
	}
	var imageLayer string
	var containerLayer string
	importFrom := b.FromImage
	if !b.Store.Exists(importFrom) && b.FromImageID != "" {
		img, err := b.Store.Image(b.FromImageID)
		if err != nil {
			return err
		}
		importFrom = img.Names[0]
	} else {
		importFrom = "scratch"
	}
	// set transport to containers-storage:
	transportName := defaultTransport + exportTo
	exportRef, err := alltransports.ParseImageName(transportName)
	if err != nil {
		return err
	}

	ops := &cpier.Options{}

	// First need to determine whether there are changes in the builder's layers, if there are changes you need to
	// merge the layers, no changes only need to copy the image.
	if b.FromImageID != "" {
		iM, _ := b.Store.Image(b.FromImageID)
		imageLayer = iM.TopLayer
	} else {
		imageLayer = ""
	}
	ctr, _ := b.Store.Container(b.ContainerID)
	containerLayer = ctr.LayerID
	changes, err := b.Store.Changes(imageLayer, containerLayer)
	if err != nil {
		return err
	}
	for _, change := range changes {
		switch change.Kind {
		case archive.ChangeModify:
			logrus.Infof("modify %s", change.Path)
		case archive.ChangeAdd:
			logrus.Infof("add %s", change.Path)
		case archive.ChangeDelete:
			logrus.Infof("delete %s", change.Path)
		}
	}

	if len(changes) > 0 || importFrom == "scratch" {
		var layerOps storage.LayerOptions
		var diffOps storage.DiffOptions
		diffrdcloser, err := b.Store.Diff(imageLayer, containerLayer, &diffOps)
		if err != nil {
			return fmt.Errorf("failed to get layer diff: %w", err)
		}

		tar, err := os.CreateTemp("", "layer-diff-tar-")
		wt := bufio.NewWriter(tar)
		if err != nil {
			return err
		}
		defer os.Remove(tar.Name())
		defer tar.Close()

		_, err = io.Copy(wt, diffrdcloser)
		if err != nil {
			return fmt.Errorf("storing blob to file %v: %w", tar, err)
		}
		if err := wt.Flush(); err != nil {
			return fmt.Errorf("can not flush bufio: %v", err)
		}
		diffrdcloser.Close()

		f, err := os.Open(tar.Name())
		if err != nil {
			return fmt.Errorf("Can not open the file of: %q: %w", tar.Name(), err)
		}
		defer f.Close()

		destLayer, num, _ := b.Store.PutLayer("", imageLayer, []string{}, "", true, &layerOps, f)
		if num != -1 {
			logrus.Infof("apply diff %s successfully", containerLayer)
		}

		referceName := defaultNullImageName
		removeOldImage := false
		if exportTo != defaultNullImageName {
			referceName = exportRef.DockerReference().String()
		}
		logrus.Infof("export name is %s", referceName)
		if err, isRemove := b.verifyCommitTag(referceName); err != nil {
			return err
		} else {
			removeOldImage = isRemove
		}

		nname := []string{referceName}
		imageOptions := &storage.ImageOptions{
			Digest: digest.Digest(""),
		}
		nwImage, err := b.Store.CreateImage("", nname, destLayer.ID, "", imageOptions)
		if err != nil {
			logrus.Errorf("fail to create new image at store: %v", err)
			return err
		}

		// generate manifest info and setBigData to new images
		items, err := b.generateManifests(nwImage.ID)
		if err != nil {
			return err
		}

		// the manifest and instance.json information from builderBigData, write it to the new image
		for _, item := range items {
			var data []byte
			data, err = b.builderBigData(b.ContainerID, item)
			if err != nil {
				return fmt.Errorf("error copying data item %q: %w", item, err)
			}
			logrus.Infof("the id is %s , and the data is %s", item, data)
			err := b.Store.SetImageBigData(nwImage.ID, item, data, v5manifest.Digest)
			if err != nil {
				return fmt.Errorf("error copying data item %q: %w", item, err)
			}
			logrus.Debugf("copied data item %q to %q", item, nwImage.ID)
		}

		if removeOldImage {
			if err := b.Store.DeleteContainer(b.ContainerID); err != nil {
				logrus.Errorf("fail to remove builder %s: %v", b.ContainerID, err)
				return err
			}
			if _, err := b.Store.DeleteImage(b.FromImageID, true); err != nil {
				logrus.Errorf("fail to remove rename image of %s", b.FromImageID)
				return err
			}
		}
		logrus.Infof("create new image %s successful", nwImage.ID)
		return nil
	}

	// set transport to oci
	importFrom = defaultTransport + importFrom
	importRef, err := alltransports.ParseImageName(importFrom)
	if err != nil {
		return err
	}

	_, err = cpier.Image(ctx, policyContext, exportRef, importRef, ops)
	if err != nil {
		return err
	}
	return nil
}

// generate the manifest message and write it to BigData of builder
func (b *Builder) generateManifests(id string) ([]string, error) {
	// keys include sha256:imageID、 manifest-sha256:imageDigestID、manifest.
	// digest of manifest-sha256:imageDigestID and manifest is sha256:imageDigestID, and content is consistent with the image-spec
	// digest of sha256:imageID is self, and content is Schema2Image
	bigDatas := []storage.ContainerBigDataOption{}
	bigDataName := []string{}
	// about opencontainer image-spec manifest from builder
	imageSpecManifestData, err := json.Marshal(b.Manifest)
	if err != nil {
		logrus.Errorf("")
		return nil, err
	}
	// about docker image spec from builder
	schema2ImageData, err := json.Marshal(b.DockerV2)

	if err != nil {
		logrus.Errorf("")
		return nil, err
	}
	manifestDigest := digest.FromBytes(schema2ImageData)
	// generate bigDataOption
	bigDatas = append(bigDatas, storage.ContainerBigDataOption{
		Key:  storage.ImageDigestManifestBigDataNamePrefix,
		Data: imageSpecManifestData,
	})
	bigDatas = append(bigDatas, storage.ContainerBigDataOption{
		Key:  storage.ImageDigestManifestBigDataNamePrefix + "-" + manifestDigest.String(),
		Data: imageSpecManifestData,
	})
	bigDatas = append(bigDatas, storage.ContainerBigDataOption{
		Key:  digest.NewDigestFromHex(digest.Canonical.String(), id).String(),
		Data: schema2ImageData,
	})
	for _, data := range bigDatas {
		b.setBuilderBigData(b.ID, data.Key, data.Data)
		bigDataName = append(bigDataName, data.Key)
	}

	return bigDataName, nil
}

func (b *Builder) setBuilderBigData(id, key string, data []byte) error {
	err := b.Store.SetContainerBigData(id, key, data)
	if err != nil {
		logrus.Errorf("Failed to set BigData to builder: %v", err)
		return err
	}
	return nil
}

// builderBigData retrieves a (possibly large) chunk of named data
// associated with an container.
func (b *Builder) builderBigData(id, key string) ([]byte, error) {
	data, err := b.Store.ContainerBigData(id, key)
	if err != nil {
		logrus.Errorf("Failed to get BigData from builder: %v", err)
		return nil, err
	}
	return data, nil
}

func (b *Builder) verifyCommitTag(name string) (error, bool) {
	if !b.Store.Exists(name) {
		return nil, false
	}
	epImg, err := b.Store.Image(name)
	b.FromImageID = epImg.ID
	if err != nil {
		return err, false
	}
	logrus.Infof("begin to delete reuse image tag: %s", epImg.ID)
	if err := b.Store.RemoveNames(epImg.ID, []string{name}); err != nil {
		logrus.Errorf("fail to remove reuse image tag: %v", err)
		return err, false
	}
	return nil, true
}

func (b *Builder) SetWorkdir(args string) {
	b.Workdir = args
}

func (b *Builder) Add(dest string, source []string, extract bool) error {
	if err := b.Mount(""); err != nil {
		return err
	}
	mountPoint := b.MountPoint
	if filepath.IsAbs(dest) {
		dest = filepath.Join(mountPoint, dest)
	} else {
		dest = filepath.Join(mountPoint, b.Workdir, dest)
	}
	def, _ := os.Stat(dest)

	archiver := archive.NewDefaultArchiver()
	for _, src := range source {
		srf, err := os.Stat(src)
		if err != nil {
			return err
		}
		if srf.IsDir() {
			d := dest
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("error ensuring directory %q exists", d)
			}
			logrus.Debugf("copying %q to %q", src+string(os.PathSeparator)+"*", d+string(os.PathSeparator)+"*")
			// CopyWithTar creates a tar archive of filesystem path `src`, and unpacks it at filesystem path `dst`
			if err := archiver.CopyWithTar(src, d); err != nil {
				return fmt.Errorf("error copying %q to %q", src, d)
			}
			continue
		}
		// IsArchivePath checks if the (possibly compressed) file at the given path starts with a tar file header.
		if !extract || !archive.IsArchivePath(src) {
			d := dest
			if def != nil && def.IsDir() {
				d = filepath.Join(dest, filepath.Base(src))
			}
			logrus.Debugf("copying %q to %q", src, d)
			// CopyFileWithTar emulates the behavior of the 'cp' command-line for a single file. It copies a regular
			// file from path `src` to path `dst`, and preserves all its metadata.
			if err := archiver.CopyFileWithTar(src, d); err != nil {
				return fmt.Errorf("error copying %q to %q", src, d)
			}
			continue
		}
		logrus.Debugf("extracting contents of %q into %q", src, dest)
		// UntarPath untar a file from path to a destination, src is the source tar file path.
		if err := archiver.UntarPath(src, dest); err != nil {
			return fmt.Errorf("error extracting %q into %q", src, dest)
		}
	}
	return nil
}

func (b *Builder) Run(args []string, ops options.RUNOption) error {
	g, err := generate.New("linux")
	if err != nil {
		return err
	}
	// Currently, the cni component is not supported to create container networks, so the host network is still used.
	if err = g.RemoveLinuxNamespace("network"); err != nil {
		return fmt.Errorf("error removing network namespace for run: %v", err)
	}
	if err := b.Mount(""); err != nil {
		return err
	}
	mountPoint := b.MountPoint
	g.SetRootPath(mountPoint)
	if args != nil {
		g.SetProcessArgs([]string{"/bin/sh", "-c", strings.Join(args, " ")})
	} else {
		g.SetProcessArgs([]string{"bash"})
	}

	g.SetProcessCwd("/")
	if ops.Workdir != "" {
		g.SetProcessCwd(ops.Workdir)
	}
	cdir, err := b.Store.ContainerDirectory(b.ContainerID)
	if err != nil {
		return err
	}
	var exportOps generate.ExportOptions
	specPath := filepath.Join(cdir, specFile)
	if err := g.SaveToFile(specPath, exportOps); err != nil {
		return err
	}

	ctrid := "runtime" + "-" + b.ContainerID
	var allArgs []string
	allArgs = append(allArgs, "run", "-b", cdir, ctrid)

	cmd := exec.Command(defaultruntime, allArgs...)
	if ops.Runtime != "" {
		cmd = exec.Command(ops.Runtime, allArgs...)
	}
	cmd.Dir = mountPoint
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("runtime exec failed: %s", err)
	}
	return err
}

func (b *Builder) SetLabel(containerID string, labels map[string]string) error {
	// Find the container's configuration file path
	configDir, err := b.Store.ContainerDirectory(containerID)
	if err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "ktib.json")

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Parse the current configuration
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update labels
	if config["Labels"] == nil {
		config["Labels"] = make(map[string]string)
	}
	// Get existing labels
	existingLabels := make(map[string]string)

	if rawLabels, ok := config["Labels"].(map[string]interface{}); ok {
		// Type produced by JSON unmarshalling
		for k, v := range rawLabels {
			existingLabels[k] = fmt.Sprintf("%v", v)
		}
	} else if stringLabels, ok := config["Labels"].(map[string]string); ok {
		existingLabels = stringLabels
	}

	// Merge new labels
	for key, value := range labels {
		existingLabels[key] = value
	}

	// Update configuration
	config["Labels"] = existingLabels

	// Update the configuration file
	newData, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	fmt.Printf("Successfully set labels for container %s: %v\n", containerID, labels)
	return nil
}
