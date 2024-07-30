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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gitee.com/openeuler/ktib/pkg/options"
	cpier "github.com/containers/image/v5/copy"
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
	defaultruntime       = "runc"
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
	OCIv1       v1.Image
	Workdir     string
	out         io.Writer
}

type BuilderOptions struct {
	FromImage  string
	Container  string
	PullPolicy bool
}

type Executor struct {
	store      storage.Store
	contextDir string
	builders   *Builder
	out        io.Writer
	err        io.Writer
}

func newBuidler(store storage.Store, options BuilderOptions) (*Builder, error) {
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
		iMage, err := store.Image(image)
		if err != nil {
			return nil, err
		}
		imageID = iMage.ID
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
	statefile := filepath.Join(cdir, stateFile)

	buildstate, err := ioutil.ReadFile(statefile)
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

func (b *Builder) SetCmd(args string) {
	b.Cmd = args
}

func (b *Builder) SetEnv(args []string) {
	b.Env = args
}

func (b *Builder) SetMessage(args string) {
	b.Message = args
}

func (b *Builder) Remove() error {
	// If the submitted image name exists, the container will be removed early
	if !b.Store.Exists(b.ContainerID) {
		return nil
	}

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

func (b *Builder) Commit(exportTo string) error {
	ctx := context.Background()
	systemContext := types.SystemContext{}
	policy, err := signature.DefaultPolicy(&systemContext)
	if err != nil {
		return err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	var imageLayer string
	var containerLayer string
	importFrom := b.FromImage
	if !b.Store.Exists(importFrom) && b.FromImageID != "" {
		iMage, err := b.Store.Image(b.FromImageID)
		if err != nil {
			return err
		}
		importFrom = iMage.Names[0]
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
			return fmt.Errorf("Can not flush bufio: %w", err)
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
			logrus.Errorf("fail to create new image at store: %w", err)
			return err
		}
		if removeOldImage {
			if err := b.Store.DeleteContainer(b.ContainerID); err != nil {
				logrus.Errorf("fail to remove builder %s of %w", b.ContainerID, err)
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

func (b *Builder) verifyCommitTag(name string) (error, bool) {
	isRemove := false
	if b.Store.Exists(name) {
		epImg, err := b.Store.Image(name)
		b.FromImageID = epImg.ID
		if err != nil {
			return err, isRemove
		}
		logrus.Infof("begin to delete reuse image tag: %s", epImg.ID)
		if err := b.Store.RemoveNames(epImg.ID, []string{name}); err != nil {
			logrus.Errorf("fail to remove reuse image tag: %w", err)
			return err, isRemove
		}
		isRemove = true
	}
	return nil, isRemove
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
		return fmt.Errorf("error removing network namespace for run", err)
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

func BuildDockerfiles(store storage.Store, op *options.BuildOptions, dockerfile ...string) error {
	var lineContinuation = regexp.MustCompile(`\\\s*\n`)
	if len(dockerfile) == 0 {
		return errors.New("error building: no dockerfiles specified\n")
	}
	exec, err := NewExecutor(store, op)
	if err != nil {
		return fmt.Errorf("error creating build executor: %w", err)
	}

	for _, value := range dockerfile {
		fileBytes, err := ioutil.ReadFile(value)
		if err != nil {
			return err
		}
		if len(fileBytes) == 0 {
			return errors.New("Dockerfile cannot be empty")
		}
		var (
			dockerfileContent = lineContinuation.ReplaceAllString(stripComments(fileBytes), "")
			stepN             = 1
		)
		//Split into lines based on line breaks
		for _, line := range strings.Split(dockerfileContent, "\n") {
			//Remove spaces, tabs, and line breaks at the beginning and end of a line
			line = strings.Trim(strings.Replace(line, "\t", " ", -1), " \t\r\n")
			if len(line) == 0 {
				continue
			}
			//Execute each step of construction
			if err := exec.BuildStep(fmt.Sprintf("%d", stepN), line); err != nil {
				return err
			}
			stepN += 1
		}

		if err := exec.BuildCommit(op); err != nil {
			return err
		}
	}
	return nil
}

func NewExecutor(store storage.Store, options *options.BuildOptions) (*Executor, error) {
	exec := Executor{
		store:      store,
		contextDir: options.ContextDirectory,
		out:        options.Out,
		err:        options.Err,
	}
	if exec.err == nil {
		exec.err = os.Stderr
	}
	if exec.out == nil {
		exec.out = os.Stdout
	}
	return &exec, nil
}

// Remove all lines starting with '#' or empty spaces from the []byte
func stripComments(raw []byte) string {
	var (
		out   []string
		lines = strings.Split(string(raw), "\n")
	)
	for _, l := range lines {
		//Filter out lines that start with "#" or are empty
		if len(l) == 0 || l[0] == '#' {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func (b *Executor) BuildStep(name, expression string) error {
	fmt.Fprintf(b.out, "Step %s : %s\n", name, expression)
	tmp := strings.SplitN(expression, " ", 2)
	if len(tmp) != 2 {
		return fmt.Errorf("Invalid Dockerfile format")
	}
	instruction := strings.Trim(tmp[0], " ")
	arguments := strings.Trim(tmp[1], " ")
	switch instruction {
	case "FROM":
		option := BuilderOptions{
			FromImage: arguments,
		}
		builders, err := NewBuilder(b.store, option)
		if err != nil {
			return errors.New(fmt.Sprintf("error creating build container: %s\n", err))
		}
		fmt.Printf("%s\n", builders.ContainerID)
		if err := builders.Save(); err != nil {
			return err
		}
		b.builders = builders
	case "ADD", "COPY":
		tmp := strings.Split(arguments, " ")
		source := tmp[:len(tmp)-1]
		dest := tmp[len(tmp)-1]
		var isADD bool
		if instruction == "ADD" {
			isADD = true
		}
		err := b.builders.Add(dest, source, isADD)
		if err != nil {
			return errors.New(fmt.Sprintf("error adding or copying content to builder: %s", err))
		}
	case "RUN":
		args := strings.Split(arguments, " ")
		var ops options.RUNOption
		if err := b.builders.Run(args, ops); err != nil {
			return err
		}
	case "CMD":
		b.builders.SetCmd(arguments)
	default:
		if b.builders != nil {
			err := b.builders.Remove()
			b.builders = nil
			return fmt.Errorf("Unsupported Dockerfile directive: %s, %v\n", instruction, err)
		}
	}
	return nil
}

func (b *Executor) BuildCommit(op *options.BuildOptions) error {
	err := b.builders.Commit(op.Tags)
	if err != nil {
		return errors.New(fmt.Sprintf("error commit container to images: %s", err))
	}
	if b.builders != nil {
		err = b.builders.Remove()
		b.builders = nil
	}
	return nil
}
