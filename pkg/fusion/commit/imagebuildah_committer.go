/*
   Copyright (c) 2026 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package commit

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// Committer defines interface for committing rootfs to image
type Committer interface {
	Commit(rootfs string, targetTag string, sourceImage string) error
}

type ImageBuildahCommitter struct {
	Store storage.Store
}

func NewImageBuildahCommitter(store storage.Store) *ImageBuildahCommitter {
	return &ImageBuildahCommitter{Store: store}
}

func (c *ImageBuildahCommitter) Commit(rootfs string, targetTag string, sourceImage string) error {
	logrus.Infof("Committing rootfs %s to image %s", rootfs, targetTag)

	tmpDir, err := os.MkdirTemp("", "ktib-fusion-commit-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	tarPath := filepath.Join(tmpDir, "rootfs.tar")

	// Base Dockerfile
	dockerfileContent := "FROM scratch\nADD rootfs.tar /\n"

	// Inherit config from source image
	if sourceImage != "" {
		configCmds, err := c.generateConfigCommands(sourceImage)
		if err != nil {
			logrus.Warnf("Failed to inherit config from %s: %v", sourceImage, err)
		} else {
			dockerfileContent += configCmds
		}
	}

	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0o644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	if err := tarDirectory(rootfs, tarPath); err != nil {
		return fmt.Errorf("failed to tar rootfs: %w", err)
	}

	ctx := context.Background()
	opts := define.BuildOptions{
		ContextDirectory:        tmpDir,
		Output:                  targetTag,
		OutputFormat:            define.Dockerv2ImageManifest,
		Layers:                  false,
		Squash:                  true,
		RemoveIntermediateCtrs:  true,
		ForceRmIntermediateCtrs: true,
		Runtime:                 "runc",
		Out:                     os.Stdout,
		Err:                     os.Stderr,
		ReportWriter:            os.Stderr,
		SystemContext:           &types.SystemContext{},
	}

	_, _, err = imagebuildah.BuildDockerfiles(ctx, c.Store, opts, dockerfilePath)
	if err != nil {
		return fmt.Errorf("imagebuildah build failed: %w", err)
	}

	logrus.Infof("Successfully committed image: %s", targetTag)
	return nil
}

func (c *ImageBuildahCommitter) generateConfigCommands(imageRef string) (string, error) {
	ctx := context.Background()
	runtime, err := libimage.RuntimeFromStore(c.Store, &libimage.RuntimeOptions{})
	if err != nil {
		return "", err
	}

	img, _, err := runtime.LookupImage(imageRef, nil)
	if err != nil {
		return "", err
	}

	data, err := img.Inspect(ctx, nil)
	if err != nil {
		return "", err
	}

	if data.Config == nil {
		return "", nil
	}

	var cmds strings.Builder

	// ENV
	for _, env := range data.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			val := parts[1]
			// Use JSON marshaling to handle quoting/escaping safely
			jsonVal, _ := json.Marshal(val)
			cmds.WriteString(fmt.Sprintf("ENV %s=%s\n", key, string(jsonVal)))
		}
	}

	// WORKDIR
	if data.Config.WorkingDir != "" {
		cmds.WriteString(fmt.Sprintf("WORKDIR %s\n", data.Config.WorkingDir))
	}

	// USER
	if data.Config.User != "" {
		cmds.WriteString(fmt.Sprintf("USER %s\n", data.Config.User))
	}

	// CMD
	if len(data.Config.Cmd) > 0 {
		jsonCmd, _ := json.Marshal(data.Config.Cmd)
		cmds.WriteString(fmt.Sprintf("CMD %s\n", string(jsonCmd)))
	}

	// ENTRYPOINT
	if len(data.Config.Entrypoint) > 0 {
		jsonEp, _ := json.Marshal(data.Config.Entrypoint)
		cmds.WriteString(fmt.Sprintf("ENTRYPOINT %s\n", string(jsonEp)))
	}

	// LABEL
	if len(data.Config.Labels) > 0 {
		for k, v := range data.Config.Labels {
			jsonV, _ := json.Marshal(v)
			cmds.WriteString(fmt.Sprintf("LABEL %s=%s\n", k, string(jsonV)))
		}
	}

	// EXPOSE
	if len(data.Config.ExposedPorts) > 0 {
		for p := range data.Config.ExposedPorts {
			cmds.WriteString(fmt.Sprintf("EXPOSE %s\n", p))
		}
	}

	// VOLUME
	if len(data.Config.Volumes) > 0 {
		for v := range data.Config.Volumes {
			jsonV, _ := json.Marshal(v)
			cmds.WriteString(fmt.Sprintf("VOLUME %s\n", string(jsonV)))
		}
	}

	return cmds.String(), nil
}

func tarDirectory(srcDir string, tarPath string) error {
	f, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	srcDir = filepath.Clean(srcDir)
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		linkname := ""
		if info.Mode()&os.ModeSymlink != 0 {
			l, err := os.Readlink(path)
			if err != nil {
				return err
			}
			linkname = l
		}

		hdr, err := tar.FileInfoHeader(info, linkname)
		if err != nil {
			return err
		}
		hdr.Name = strings.TrimPrefix(rel, "/")
		if info.IsDir() && !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			r, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(tw, r)
			_ = r.Close()
			if copyErr != nil {
				return copyErr
			}
		}
		return nil
	})
}
