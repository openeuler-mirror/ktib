/*
   Copyright (c) 2025 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package project

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/buildah/imagebuildah"
)

// BuildImage method is used to build a container image
func (b *Bootstrap) BuildImage(imageName, tag string) error {
	// Check if the rootfs directory exists
	rootfsDir := filepath.Join(b.DestinationDir, "rootfs")
	if _, err := os.Stat(rootfsDir); os.IsNotExist(err) {
		return fmt.Errorf("rootfs directory does not exist, please run 'ktib project build-rootfs' first")
	}

	// Create a temporary directory for the build
	buildDir, err := os.MkdirTemp("", "ktib-build-")
	if err != nil {
		return fmt.Errorf("failed to create temporary build directory: %v", err)
	}
	defer os.RemoveAll(buildDir)

	// Create the rootfs.tar file
	rootfsTarPath := filepath.Join(buildDir, "rootfs.tar")
	if err := createTarFromDirectory(rootfsDir, rootfsTarPath); err != nil {
		return fmt.Errorf("failed to create rootfs.tar file: %v", err)
	}

	// Create the Dockerfile
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	projectDockerfilePath := filepath.Join(b.DestinationDir, "dockerfile", "Dockerfile")
	if _, err := os.Stat(projectDockerfilePath); err == nil {
		srcFile, err := os.Open(projectDockerfilePath)
		if err != nil {
			return fmt.Errorf("failed to open project Dockerfile: %v", err)
		}
		defer srcFile.Close()
		dstFile, err := os.Create(dockerfilePath)
		if err != nil {
			return fmt.Errorf("failed to create build Dockerfile: %v", err)
		}
		defer dstFile.Close()
		if _, err = io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy project Dockerfile: %v", err)
		}
	} else {
		cmd := "/bin/bash"
		if b.BuildType == "init" {
			cmd = "/sbin/init"
		}
		dockerfileContent := "FROM scratch\nADD rootfs.tar /\nCMD [\"" + cmd + "\"]\n"
		if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
			return fmt.Errorf("failed to create Dockerfile: %v", err)
		}
	}

	// Copy files from the files directory to the build directory (if necessary)
	filesDir := filepath.Join(b.DestinationDir, "files")
	if _, err := os.Stat(filesDir); err == nil {
		entries, err := os.ReadDir(filesDir)
		if err != nil {
			return fmt.Errorf("failed to read files directory: %v", err)
		}

		for _, entry := range entries {
			srcPath := filepath.Join(filesDir, entry.Name())
			dstPath := filepath.Join(buildDir, entry.Name())

			if entry.IsDir() {
				// If it's a directory, create the corresponding directory
				if err := os.MkdirAll(dstPath, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %v", dstPath, err)
				}
			} else {
				// If it's a file, copy the file content
				srcFile, err := os.Open(srcPath)
				if err != nil {
					return fmt.Errorf("failed to open source file %s: %v", srcPath, err)
				}
				defer srcFile.Close()

				dstFile, err := os.Create(dstPath)
				if err != nil {
					return fmt.Errorf("failed to create destination file %s: %v", dstPath, err)
				}
				defer dstFile.Close()

				if _, err = io.Copy(dstFile, srcFile); err != nil {
					return fmt.Errorf("failed to copy file content: %v", err)
				}
			}
		}
	}

	// Use ktib's internal build interface to build the image
	imageTag := fmt.Sprintf("%s:%s", imageName, tag)

	// Directly use the underlying buildah interface to build the image, avoiding cobra.Command initialization issues
	// Create build options
	buildOptions := &options.BuildOptions{
		Tags:    []string{imageTag},
		Format:  utils.DefaultFormat(),
		Rm:      true,
		ForceRm: true,
	}

	// Set build arguments
	args := []string{buildDir}

	// Resolve Dockerfile path and context directory
	dockerfiles, contextDir, err := utils.ResolveDockerfiles(buildOptions, args)
	if err != nil {
		return fmt.Errorf("failed to resolve Dockerfile: %v", err)
	}

	// Get store
	store, err := utils.GetStore(nil)
	if err != nil {
		return fmt.Errorf("failed to get store: %v", err)
	}

	// Manually create buildah build options
	buildahOptions := &imagebuildah.BuildOptions{
		ContextDirectory:        contextDir,
		PullPolicy:              imagebuildah.PullIfMissing,
		Compression:             imagebuildah.Gzip,
		Output:                  imageTag,
		AdditionalTags:          []string{},
		Out:                     os.Stdout,
		Err:                     os.Stderr,
		ReportWriter:            os.Stderr,
		ForceRmIntermediateCtrs: true,
		RemoveIntermediateCtrs:  true,
	}

	// Build image
	ctx := context.Background()
	_, _, err = imagebuildah.BuildDockerfiles(ctx, store, *buildahOptions, dockerfiles...)
	if err != nil {
		return fmt.Errorf("failed to build image: %v", err)
	}

	fmt.Printf("Successfully built image: %s\n", imageTag)
	return nil
}

// createTarFromDirectory creates a tar file containing all content in the specified directory
func createTarFromDirectory(sourceDir, tarFilePath string) error {
	tarFile, err := os.Create(tarFilePath)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %v", err)
	}
	defer tarFile.Close()

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create file header: %v", err)
		}

		// Modify the name in the header to the relative path
		header.Name = relPath

		// Handle symbolic links
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink target: %v", err)
			}
			header.Linkname = linkTarget
		}

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write file header: %v", err)
		}

		// If it is a regular file, write the file content
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file: %v", err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file content: %v", err)
			}
		}

		return nil
	})
}

// createDefaultDockerfile creates the default Dockerfile
func createDefaultDockerfile(projectDir string) error {
	dockerfileDir := filepath.Join(projectDir, "dockerfile")
	if err := os.MkdirAll(dockerfileDir, 0755); err != nil {
		return err
	}

	dockerfilePath := filepath.Join(dockerfileDir, "Dockerfile")
	content := `FROM scratch
ADD rootfs.tar /
CMD ["/bin/bash"]
`
	return os.WriteFile(dockerfilePath, []byte(content), 0644)
}
