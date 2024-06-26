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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitee.com/openeuler/ktib/pkg/options"
	cpier "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/ioutils"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	stateFile        = "ktib.json"
	specFile         = "config.json"
	defaultTransport = "containers-storage:"
	defaultruntime   = "runc"
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
	Env         []string
	Message     string
	OCIv1       v1.Image
	Workdir     string
}

type BuilderOptions struct {
	FromImage  string
	Container  string
	PullPolicy bool
}

func newBuidler(store storage.Store, options BuilderOptions) (*Builder, error) {
	image := options.FromImage
	name := options.Container
	coptions := storage.ContainerOptions{}
	container, err := store.CreateContainer("", []string{name}, image, "", "", &coptions)
	if err != nil {
		return nil, err
	}

	iMage, err := store.Image(image)
	if err != nil {
		return nil, err
	}
	builder := &Builder{
		Name:        container.Names[0],
		ID:          container.ID,
		Store:       store,
		FromImage:   image,
		FromImageID: iMage.ID,
		Container:   name,
		ContainerID: container.ID,
	}
	if err := builder.Save(); err != nil {
		return nil, err
	}
	return builder, nil
}

func NewBuilder(store storage.Store, options BuilderOptions) (*Builder, error) {
	// TODO 构造builder对象
	return newBuidler(store, options)
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
	buildstate, err := ioutil.ReadFile(filepath.Join(cdir, stateFile))
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
		buildstate, err := ioutil.ReadFile(filepath.Join(cdir, stateFile))
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

func (b *Builder) Copy(args []string) error {
	return nil
}

func (b *Builder) Label(args []string) error {
	return nil
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

func (b *Builder) SetEnv(args []string) {
	b.Env = args
}

func (b *Builder) SetMessage(args string) {
	b.Message = args
}

func (b *Builder) Remove() error {
	if err := b.Store.DeleteContainer(b.ContainerID); err != nil {
		logrus.Error(fmt.Sprintf("delete builder failed: %s", err))
		return err
	}
	return nil
}

func (b Builder) name() string {
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

func (b Builder) Commit(exportTo string) error {
	ctx := context.Background()
	systemContext := types.SystemContext{}
	policy, err := signature.DefaultPolicy(&systemContext)
	if err != nil {
		return err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	importFrom := b.FromImage
	var imageLayer string
	var containerLayer string
	if !b.Store.Exists(importFrom) {
		iMage, err := b.Store.Image(b.FromImageID)
		if err != nil {
			return err
		}
		importFrom = iMage.Names[0]
	}
	// set transport to oci
	importFrom = defaultTransport + importFrom
	importRef, err := alltransports.ParseImageName(importFrom)
	if err != nil {
		return err
	}

	// set transport to containers-storage:
	exportTo = defaultTransport + exportTo
	exportRef, err := alltransports.ParseImageName(exportTo)
	if err != nil {
		return err
	}

	exportName := exportRef.DockerReference().String()
	if b.Store.Exists(exportName) {
		return errors.New(fmt.Sprintf("The image %s is exists.", exportName))
	}

	ops := &cpier.Options{}

	// First need to determine whether there are changes in the builder's layers, if there are changes you need to
	// merge the layers, no changes only need to copy the image.
	iM, _ := b.Store.Image(b.FromImageID)
	imageLayer = iM.TopLayer
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

	if len(changes) > 0 {
		var layerOps storage.LayerOptions
		var diffOps storage.DiffOptions
		diffrdcloser, err := b.Store.Diff(imageLayer, containerLayer, &diffOps)

		tar, err := os.CreateTemp("", "layer-diff-tar-")
		if err != nil {
			return err
		}
		defer os.Remove(tar.Name())
		defer tar.Close()

		_, err = io.Copy(tar, diffrdcloser)
		if err != nil {
			return fmt.Errorf("storing blob to file %q: %w", tar, err)
		}

		diffrdcloser.Close()

		destLayer, num, _ := b.Store.PutLayer("", imageLayer, []string{}, "", true, &layerOps, tar)
		if num != -1 {
			logrus.Infof("apply diff %s successfully", containerLayer)
		}
		nname := []string{exportName}
		nwImage, _ := b.Store.CreateImage("", nname, destLayer.ID, "", nil)
		if err != nil {
			return err
		}
		logrus.Infof("create new image %s successful", nwImage.ID)
		return nil
	}
	_, err = cpier.Image(ctx, policyContext, exportRef, importRef, ops)
	if err != nil {
		return err
	}
	return nil
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
	def, err := os.Stat(dest)
	if err != nil {
		return err
	}
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

	if err := b.Mount(""); err != nil {
		return err
	}
	mountPoint := b.MountPoint
	g.SetRootPath(mountPoint)

	g.SetProcessArgs([]string{"bash"})
	if args != nil {
		g.SetProcessArgs(args)
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
	cmd := exec.Command(defaultruntime, args...)
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
	// 找到容器的配置文件路径
	configDir, err := b.Store.ContainerDirectory(containerID)
	if err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "ktib.json")

	// 读取配置文件
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	// 解析当前配置
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 更新标签
	if config["Labels"] == nil {
		config["Labels"] = make(map[string]string)
	}
	labelsMap := config["Labels"].(map[string]string)
	for key, value := range labels {
		labelsMap[key] = value
	}
	config["Labels"] = labelsMap

	// 更新配置文件
	newData, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(configPath, newData, 0644); err != nil {
		return err
	}

	fmt.Printf("成功为容器 %s 设置标签: %v\n", containerID, labels)
	return nil
}
