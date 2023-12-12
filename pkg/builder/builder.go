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
	"encoding/json"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/ioutils"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	DefaultRuntime = "runc"
	CRuntime       = "crun"
	RustRuntime    = "youki"
	stateFile      = "ktib.json"
	configFile     = "config.json"
	DefaultWorkdir = "/"
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
	builder := &Builder{
		Name:        container.Names[0],
		ID:          container.ID,
		Store:       store,
		FromImage:   image,
		FromImageID: "",
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
	b := &Builder{}
	err = json.Unmarshal(buildstate, &b)
	if err != nil {
		return nil, err
	}
	b.Store = store
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
		b := &Builder{}
		err = json.Unmarshal(buildstate, &b)
		if err != nil {
			return nil, err
		}
		b.Store = store
		bl = append(bl, b)
	}
	return bl, nil
}

//func (b *Builder) Run(args []string, option *options.RUNOption, data *libimage.ImageData, store storage.Store) error {
//	//cdir, err := b.store.ContainerDirectory(b.ContainerID)
//	//if err != nil {
//	//	return err
//	//}
//	//var cruntime string
//	//var runtimeCommond []string
//	//switch option.Runtime {
//	//case 1:
//	//	cruntime = DefaultRuntime
//	//case 2:
//	//	cruntime = CRuntime
//	//case 3:
//	//	cruntime = RustRuntime
//	//default:
//	//	cruntime = DefaultRuntime
//	//}
//	//// TODO ioutils.AtomicWriteFile config.json by github.com/opencontainers/runtime-tools
//	//specgen, err := generate.New(runtime.GOOS)
//	//
//	//if option.TTY {
//	//	specgen.SetProcessTerminal(option.TTY)
//	//} else {
//	//	specgen.SetProcessTerminal(false)
//	//}
//	//if option.Workdir != "" {
//	//	specgen.SetProcessCwd(option.Workdir)
//	//} else {
//	//	specgen.SetProcessCwd(DefaultWorkdir)
//	//}
//	//specstate, err := json.Marshal(specgen.Config)
//	//if err != nil {
//	//	return err
//	//}
//	//err = ioutils.AtomicWriteFile(filepath.Join(cdir, configFile), specstate, 0600)
//	//if err != nil {
//	//	return err
//	//}
//	//
//	//// TODO make bundle path
//	//u, err := user.Current()
//	//uid, err := strconv.Atoi(u.Uid)
//	//gid, err := strconv.Atoi(u.Gid)
//	//overlayDest := store.GraphRoot()
//	//rPath := "userdata/" + "rootfs"
//	//contentDir, err := overlay.GenerateStructure(overlayDest, b.ContainerID, rPath, uid, gid)
//	//if err != nil {
//	//	return err
//	//}
//	//if err != nil {
//	//	return err
//	//}
//	//var lastPath string
//	//for _, layer := range data.RootFS.Layers {
//	//	path := strings.LastIndex(string(layer), ":")
//	//	if path != -1 {
//	//		lastPath = string(layer)[path+1:]
//	//	}
//	//}
//	//rootfs := filepath.Join(store.GraphRoot(), "/overlay/", lastPath+"/diff")
//	//op := &overlay.Options{
//	//	UpperDirOptionFragment: rootfs,
//	//	WorkDirOptionFragment:  contentDir,
//	//	GraphOpts:              store.GraphOptions(),
//	//	ReadOnly:               false,
//	//	RootUID:                uid,
//	//	RootGID:                gid,
//	//}
//	//
//	//overlayMount, err := overlay.MountWithOptions(op.WorkDirOptionFragment, op.UpperDirOptionFragment, overlayDest, op)
//	////content: /var/lib/containers/storage/overlay-containers/5a4c5398332e0fe913c3407cd7a94c875f73790a0e8eac259bae9a19e50b9aef/userdata/rootfs
//	////rootfs: /var/lib/containers/storage/overlay/3d24ee258efc3bfe4066a1a9fb83febf6dc0b1548dfe896161533668281c9f4f/diff
//	////overlaydest /var/lib/containers/storage
//	//// TODO: lowerdir=镜像层；
//	//// TODO: workdir=merge=/var/lib/containers/storage/overlay-containers/5a4c5398332e0fe913c3407cd7a94c875f73790a0e8eac259bae9a19e50b9aef/userdata/rootfs
//	////[lowerdir=rootfs upperdir=rootfs workdir=content private]
//	//if err != nil {
//	//	return fmt.Errorf("rootfs-overlay: creating overlay failed %q: %w", rootfs, err)
//	//}
//	//b.MountPoint = overlayMount.Source
//	//rtargs := append(runtimeCommond, "run", "-b", cdir, b.ContainerID)
//	//cmd := exec.Command(cruntime, rtargs...)
//	//cmd.Dir = b.MountPoint
//	//cmd.Stdin = os.Stdin
//	//cmd.Stdout = os.Stdout
//	//cmd.Stderr = os.Stderr
//	//err = cmd.Run()
//	//if err != nil {
//	//	return err
//	//}
//	//return nil
//}

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

//func (b *Builder) Commit(args []string) error {
// TODO github.com/containers/image/v5/copy Image func
//Only one policy is now implemented: insecure
//policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
//policyContext, err := signature.NewPolicyContext(policy)
//defer func() {
//	_ = policyContext.Destroy()
//}()
//if err != nil {
//	return err
//}
//srcRef, err := alltransports.ParseImageName(b.ID)
//if err != nil {
//	return err
//}
//// TODO containers-storage: is default transport, orther support dir:// docker:// oci://
//transport := checkTransport(args[1])
//destRef, err := alltransports.ParseImageName(transport + args[1])
//if err != nil {
//	return err
//}
//cops := &cp.Options{
//	RemoveSignatures:      true,
//	SignBy:                "",
//	ReportWriter:          os.Stdout,
//	SourceCtx:             &types.SystemContext{},
//	DestinationCtx:        &types.SystemContext{},
//	ForceManifestMIMEType: "",
//	ImageListSelection:    1,
//	OciDecryptConfig:      nil,
//	OciEncryptLayers:      nil,
//	OciEncryptConfig:      nil,
//}
//_, err = cp.Image(context.Background(), policyContext, destRef, srcRef, cops)
//if err != nil {
//	return err
//}
//return nil
//}

func (b *Builder) Remove() error {
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

func checkTransport(imageref string) string {
	transportType := alltransports.TransportFromImageName(imageref)
	if transportType != nil {
		return transportType.Name() + "://"
	}
	return "containers-storage:"
}
