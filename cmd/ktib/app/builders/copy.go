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

package builders

import (
	"errors"
	"fmt"
	"gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"github.com/containers/buildah/copier"
	"github.com/containers/podman/v4/cmd/podman/registry"
	"github.com/containers/podman/v4/pkg/copy"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/containers/podman/v4/pkg/errorhandling"
	"github.com/containers/storage/pkg/idtools"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	chown     bool
	cpOptions options.CopyOption
)

func COPYCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy files from the local filesystem to container",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return Cp(cmd, args)
		},
	}

	return cmd
}

// sourceContainerStr: 如果从容器拷贝到本地，sourceContainerStr是容器id；反之为空。
// sourcePath: 要拷贝的文件路径。
// destContainerStr： 如果从本地拷贝到容器，destContainerStr是容器id；反之为空。
// destPath: 文件要被拷贝到的本地目录或容器目录。
func Cp(cmd *cobra.Command, args []string) error {
	sourceContainerStr, sourcePath, destContainerStr, destPath, err := copy.ParseSourceAndDestination(args[0], args[1])
	if err != nil {
		return err
	}
	containerEngine, err := registry.NewContainerEngine(cmd, args)
	if err != nil {
		return err
	}
	// TODO: 缺少从容器拷贝到容器的实现
	if len(destContainerStr) > 0 && len(sourceContainerStr) > 0 {
		return copyFromContainerToContainer(sourceContainerStr, sourcePath, destContainerStr, destPath, containerEngine)
	} else if len(destContainerStr) > 0 && len(sourceContainerStr) == 0 {
		return copyToContainer(destContainerStr, destPath, sourcePath, containerEngine)
	}
	return copyToHost(sourceContainerStr, sourcePath, destPath, containerEngine)
}

func copyToContainer(containerID string, containerPath string, copyPath string, engine entities.ContainerEngine) error {
	if err := containerShouldExist(containerID, engine); err != nil {
		return err
	}
	// hostInfo.LinkTarget && containerInfo.LinkTarget: 绝对路径
	hostInfo, err := copy.ResolveHostPath(copyPath)
	if err != nil {
		return fmt.Errorf("unable to find the file path %v to copy: %v", copyPath, err)
	}
	containerInfo, err := engine.ContainerStat(registry.GetContext(), containerID, containerPath)
	if err != nil {
		return err
	}
	if strings.HasSuffix(containerPath, "/") {
		return fmt.Errorf("could not found %v on container %v: %v", containerPath, containerID, err)
	}
	var containerBaseName string
	if containerInfo != nil {
		containerBaseName = filepath.Base(containerInfo.LinkTarget)
	} else {
		containerBaseName = filepath.Base(containerPath)
	}
	hostTarget := hostInfo.LinkTarget
	if hostInfo.IsDir && filepath.Base(hostTarget) == "." {
		hostTarget = filepath.Dir(hostTarget)
	}
	if hostInfo.IsDir && !containerInfo.IsDir {
		return errors.New("destination must be a directory when copying a directory")
	}
	reader, writer := io.Pipe()
	hostCopy := func() error {
		defer writer.Close()
		getOptions := copier.GetOptions{
			KeepDirectoryNames: hostInfo.IsDir && filepath.Base(hostTarget) != ".",
		}
		if !hostInfo.IsDir && !containerInfo.IsDir {
			// If we're having a file-to-file copy, make sure to
			// rename accordingly.
			getOptions.Rename = map[string]string{filepath.Base(hostTarget): containerBaseName}
		}
		if err := copier.Get("/", "", getOptions, []string{hostTarget}, writer); err != nil {
			return fmt.Errorf("copying from host: %w", err)
		}
		return nil
	}
	containerCopy := func() error {
		defer reader.Close()
		target := containerInfo.FileInfo.LinkTarget
		if !containerInfo.IsDir {
			target = filepath.Dir(target)
		}

		copyFunc, err := engine.ContainerCopyFromArchive(registry.GetContext(), containerID, target, reader, entities.CopyOptions{Chown: chown, NoOverwriteDirNonDir: !cpOptions.OverwriteDirNonDir})
		if err != nil {
			return err
		}
		if err := copyFunc(); err != nil {
			return fmt.Errorf("copying to container: %w", err)
		}
		return nil
	}
	return doCopy(hostCopy, containerCopy)
}

func copyToHost(containerID string, copyPath string, hostPath string, engine entities.ContainerEngine) error {
	if err := containerShouldExist(containerID, engine); err != nil {
		return err
	}
	containerInfo, err := engine.ContainerStat(registry.GetContext(), containerID, copyPath)
	if err != nil {
		return err
	}
	var hostBaseName string
	var resolvedToHostParentDir bool
	hostInfo, err := copy.ResolveHostPath(hostPath)
	if err != nil {
		if strings.HasSuffix(hostPath, "/") {
			return fmt.Errorf("%q could not be found on the host: %w", hostPath, err)
		}
		parentDir := filepath.Dir(hostPath)
		hostInfo, err = copy.ResolveHostPath(parentDir)
		if err != nil {
			return fmt.Errorf("%q could not be found on the host: %w", hostPath, err)
		}
		hostBaseName = filepath.Base(hostPath)
		resolvedToHostParentDir = true
	} else {
		hostBaseName = filepath.Base(hostInfo.LinkTarget)
	}
	containerTarget := containerInfo.LinkTarget
	if resolvedToHostParentDir && containerInfo.IsDir && filepath.Base(containerTarget) == "." {
		containerTarget = filepath.Dir(containerTarget)
	}

	if containerInfo.IsDir && !hostInfo.IsDir {
		return errors.New("destination must be a directory when copying a directory")
	}
	reader, writer := io.Pipe()
	hostCopy := func() error {
		defer reader.Close()
		groot, err := user.Current()
		if err != nil {
			return err
		}
		idPair := idtools.IDPair{}
		if i, err := strconv.Atoi(groot.Uid); err == nil {
			idPair.UID = i
		} else {
			logrus.Debugf("Error converting UID %q to int: %v", groot.Uid, err)
		}
		if i, err := strconv.Atoi(groot.Gid); err == nil {
			idPair.GID = i
		} else {
			logrus.Debugf("Error converting GID %q to int: %v", groot.Gid, err)
		}

		putOptions := copier.PutOptions{
			ChownDirs:            &idPair,
			ChownFiles:           &idPair,
			IgnoreDevices:        true,
			NoOverwriteDirNonDir: !cpOptions.OverwriteDirNonDir,
			NoOverwriteNonDirDir: !cpOptions.OverwriteDirNonDir,
		}
		if (!containerInfo.IsDir && !hostInfo.IsDir) || resolvedToHostParentDir {
			// If we're having a file-to-file copy, make sure to
			// rename accordingly.
			putOptions.Rename = map[string]string{filepath.Base(containerTarget): hostBaseName}
		}
		dir := hostInfo.LinkTarget
		if !hostInfo.IsDir {
			dir = filepath.Dir(dir)
		}
		if err := copier.Put(dir, "", putOptions, reader); err != nil {
			return fmt.Errorf("copying to host: %w", err)
		}
		return nil
	}
	containerCopy := func() error {
		defer writer.Close()
		copyFunc, err := engine.ContainerCopyToArchive(registry.GetContext(), containerID, containerTarget, writer)
		if err != nil {
			return err
		}
		if err := copyFunc(); err != nil {
			return fmt.Errorf("copying from container: %w", err)
		}
		return nil
	}
	return doCopy(containerCopy, hostCopy)
}

func copyFromContainerToContainer(sourceContainerID string, sourcePath string, destContainerID string, destPath string, engine entities.ContainerEngine) error {
	if err := containerShouldExist(sourceContainerID, engine); err != nil {
		return err
	}
	if err := containerShouldExist(destContainerID, engine); err != nil {
		return err
	}
	sourceContainerInfo, err := engine.ContainerStat(registry.GetContext(), sourceContainerID, sourcePath)
	if err != nil {
		return err
	}
	destContainerInfo, err := engine.ContainerStat(registry.GetContext(), destContainerID, destPath)
	if err != nil {
		return err
	}
	if strings.HasSuffix(destPath, "/") {
		return fmt.Errorf("%v can't be found on container %v: %v", destPath, destContainerID, err)
	}
	var baseName string
	if destContainerInfo != nil {
		baseName = filepath.Base(destContainerInfo.LinkTarget)
	} else {
		baseName = filepath.Base(destPath)
	}
	if sourceContainerInfo.IsDir && !destContainerInfo.IsDir {
		return errors.New("destination must be a directory when copying a directory")
	}
	sourceContainerTarget := sourceContainerInfo.LinkTarget
	destContainerTarget := destContainerInfo.LinkTarget
	if !destContainerInfo.IsDir {
		destContainerTarget = filepath.Dir(destPath)
	}
	if sourceContainerInfo.IsDir && filepath.Base(sourcePath) == "." {
		sourceContainerTarget = filepath.Dir(sourceContainerTarget)
	}
	reader, writer := io.Pipe()
	sourceContainerCopy := func() error {
		defer writer.Close()
		copyFunc, err := engine.ContainerCopyToArchive(registry.GetContext(), sourceContainerID, sourceContainerTarget, writer)
		if err != nil {
			return err
		}
		if err := copyFunc(); err != nil {
			return fmt.Errorf("copying from container: %w", err)
		}
		return nil
	}
	destContainerCopy := func() error {
		defer reader.Close()

		copyOptions := entities.CopyOptions{Chown: chown, NoOverwriteDirNonDir: !cpOptions.OverwriteDirNonDir}
		if !sourceContainerInfo.IsDir && !destContainerInfo.IsDir {
			// If we're having a file-to-file copy, make sure to
			// rename accordingly.
			copyOptions.Rename = map[string]string{filepath.Base(sourceContainerTarget): baseName}
		}

		copyFunc, err := engine.ContainerCopyFromArchive(registry.GetContext(), destContainerID, destContainerTarget, reader, copyOptions)
		if err != nil {
			return err
		}
		if err := copyFunc(); err != nil {
			return fmt.Errorf("copying to container: %w", err)
		}
		return nil
	}
	return doCopy(sourceContainerCopy, destContainerCopy)
}

func containerShouldExist(containerID string, engine entities.ContainerEngine) error {
	ex, err := engine.ContainerExists(registry.GetContext(), containerID, options.ExistOption{}.ContainerExistsOptions)
	if err != nil {
		return err
	}
	if !ex.Value {
		return fmt.Errorf("container %v does not exits", containerID)
	}
	return nil
}
func doCopy(funcA func() error, funcB func() error) error {
	errChan := make(chan error)
	go func() {
		errChan <- funcA()
	}()
	var copyErrors []error
	copyErrors = append(copyErrors, funcB())
	copyErrors = append(copyErrors, <-errChan)
	return errorhandling.JoinErrors(copyErrors)
}
