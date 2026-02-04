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

package commit

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type ImageBuildahCommitter struct {
	Store storage.Store
}

func NewImageBuildahCommitter(store storage.Store) *ImageBuildahCommitter {
	return &ImageBuildahCommitter{Store: store}
}

func (c *ImageBuildahCommitter) Commit(rootfs string, targetTag string) error {
	logrus.Infof("Committing rootfs %s to image %s", rootfs, targetTag)

	tmpDir, err := os.MkdirTemp("", "ktib-fusion-commit-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	tarPath := filepath.Join(tmpDir, "rootfs.tar")

	if err := os.WriteFile(dockerfilePath, []byte("FROM scratch\nADD rootfs.tar /\n"), 0o644); err != nil {
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
		Layers:                  true,
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
