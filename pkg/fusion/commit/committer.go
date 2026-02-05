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
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// Committer defines interface for committing rootfs to image
type Committer interface {
	Commit(rootfs string, targetTag string) error
}

// BuildahCommitter uses buildah CLI to commit image
// We use CLI because importing buildah as library requires CGO and complicated setup.
// Since ktib runs in a container with buildah installed (hopefully), or on a host with buildah.
type BuildahCommitter struct{}

func NewBuildahCommitter() *BuildahCommitter {
	return &BuildahCommitter{}
}

func (c *BuildahCommitter) Commit(rootfs string, targetTag string) error {
	logrus.Infof("Committing rootfs %s to image %s", rootfs, targetTag)

	// Strategy:
	// 1. buildah from scratch
	// 2. mount
	// 3. copy rootfs content (or just use the directory as rootfs?)
	// Actually, buildah supports creating from a directory?
	// `buildah from scratch` -> container
	// `buildah mount container` -> path
	// `cp -r rootfs/* path/`
	// `buildah commit container targetTag`

	// Since we are running as a Go program, we can try to use `os/exec` to call `buildah`.
	
	// Check if buildah is available
	if _, err := exec.LookPath("buildah"); err != nil {
		return fmt.Errorf("buildah not found in PATH: %w", err)
	}

	// 1. Create scratch container
	cmd := exec.Command("buildah", "from", "scratch")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("buildah from scratch failed: %w", err)
	}
	containerID := string(out)
	// trim newline
	if len(containerID) > 0 && containerID[len(containerID)-1] == '\n' {
		containerID = containerID[:len(containerID)-1]
	}
	logrus.Debugf("Created scratch container: %s", containerID)
	defer func() {
		exec.Command("buildah", "rm", containerID).Run()
	}()

	// 2. Mount
	cmd = exec.Command("buildah", "mount", containerID)
	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("buildah mount failed: %w", err)
	}
	mountPoint := string(out)
	if len(mountPoint) > 0 && mountPoint[len(mountPoint)-1] == '\n' {
		mountPoint = mountPoint[:len(mountPoint)-1]
	}
	logrus.Debugf("Mounted at: %s", mountPoint)

	// 3. Copy rootfs content
	// We use `cp -a` to preserve attributes
	// cp -a rootfs/. mountPoint/
	cmd = exec.Command("cp", "-a", rootfs+"/.", mountPoint+"/")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy rootfs: %v, output: %s", err, string(out))
	}

	// 4. Commit
	cmd = exec.Command("buildah", "commit", containerID, targetTag)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("buildah commit failed: %v, output: %s", err, string(out))
	}

	logrus.Infof("Successfully committed image: %s", targetTag)
	return nil
}
