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

package fs

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	fusionrpm "gitee.com/openeuler/ktib/pkg/fusion/rpm"
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	"github.com/containers/storage"
	csarchive "github.com/containers/storage/pkg/archive"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	"github.com/sirupsen/logrus"
)

// DefaultSynthesizer implements FSSynthesizer
type DefaultSynthesizer struct {
	Store storage.Store
}

// NewDefaultSynthesizer creates a new DefaultSynthesizer
func NewDefaultSynthesizer(store storage.Store) *DefaultSynthesizer {
	return &DefaultSynthesizer{
		Store: store,
	}
}

// Synthesize creates the final rootfs
func (s *DefaultSynthesizer) Synthesize(imageRef string, plan *types.FusionPlan, outputDir string) error {
	logrus.Infof("Synthesizing filesystem for %s to %s", imageRef, outputDir)

	// 1. Build filter for files to keep
	filter, err := s.buildFileFilter(imageRef, plan)
	if err != nil {
		return fmt.Errorf("failed to build file filter: %w", err)
	}

	// 2. Extract files from layers
	if err := s.extractLayersWithFilter(imageRef, outputDir, filter); err != nil {
		return fmt.Errorf("failed to extract layers: %w", err)
	}

	ensureCompatShellLinks(outputDir)

	return nil
}

func ensureCompatShellLinks(rootfs string) {
	links := [][2]string{
		{"/bin/bash", "/usr/bin/bash"},
		{"/bin/sh", "/usr/bin/sh"},
	}

	for _, l := range links {
		dst := filepath.Join(rootfs, filepath.FromSlash(l[0]))
		if _, err := os.Lstat(dst); err == nil {
			continue
		}

		srcOnDisk := filepath.Join(rootfs, filepath.FromSlash(l[1]))
		if _, err := os.Stat(srcOnDisk); err != nil {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			continue
		}
		_ = os.Remove(dst)
		if err := os.Symlink(l[1], dst); err != nil {
			logrus.Debugf("Failed to create compat symlink %s -> %s: %v", l[0], l[1], err)
		}
	}
}

func (s *DefaultSynthesizer) buildFileFilter(imageRef string, plan *types.FusionPlan) (func(string) bool, error) {
	// We reuse analyze.Analyzer to get the RPM DB
	// Note: We need a way to access the RPM DB *content*.
	// analyze.scanRPMs works on a rootfs string.
	// Since we can't easily mount, we might need to extract the RPM DB first to a temp dir.

	tmpDir, err := os.MkdirTemp("", "ktib-fusion-rpmdb-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Extract RPM DB to tmpDir
	if err := s.ExtractRPMDB(imageRef, tmpDir); err != nil {
		return nil, fmt.Errorf("failed to extract RPM DB: %w", err)
	}

	loc, err := fusionrpm.FindRPMDB(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("RPM database not found in image: %w", err)
	}

	db, err := rpmdb.Open(loc.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rpmdb: %w", err)
	}
	defer db.Close()

	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	keptFiles := make(map[string]bool)
	droppedFiles := make(map[string]bool)

	keptSet := make(map[string]bool)
	for _, p := range plan.KeptPackages {
		keptSet[p] = true
	}

	for _, p := range pkgList {
		files, err := p.InstalledFiles()
		if err != nil {
			logrus.Warnf("Failed to get files for package %s: %v", p.Name, err)
			continue
		}

		if keptSet[p.Name] {
			// Add all files of this package to keptFiles
			for _, f := range files {
				keptFiles[f.Path] = true
			}
		} else {
			// Add all files of this package to droppedFiles
			for _, f := range files {
				droppedFiles[f.Path] = true
			}
		}
	}

	// Add explicit kept files from plan
	for _, f := range plan.KeptFiles {
		keptFiles[f] = true
	}

	retainUnowned := true
	if plan.Config != nil {
		retainUnowned = plan.Config.Fusion.Behavior.RetainUnowned
	}

	return func(path string) bool {
		if keptFiles[path] {
			return true
		}
		if droppedFiles[path] {
			return false
		}
		return retainUnowned
	}, nil
}

func (s *DefaultSynthesizer) ExtractRPMDB(imageRef string, dest string) error {
	// Similar to extractLayers but only matching /var/lib/rpm prefix
	// and we want the *latest* version of it (Top layer wins)
	// But RPM DB usually is modified in place.
	return s.extractLayersWithFilter(imageRef, dest, func(path string) bool {
		return strings.HasPrefix(path, "/var/lib/rpm") || strings.HasPrefix(path, "/usr/lib/sysimage/rpm")
	})
}

// ExtractFiles extracts specific files from the image to a destination directory
func (s *DefaultSynthesizer) ExtractFiles(imageRef string, files []string, outputDir string) error {
	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	return s.extractLayersWithFilter(imageRef, outputDir, func(path string) bool {
		return fileSet[path]
	})
}

func (s *DefaultSynthesizer) extractLayersWithFilter(imageRef string, outputDir string, filter func(string) bool) error {
	img, err := s.Store.Image(imageRef)
	if err != nil {
		return err
	}

	// We need to iterate layers from Bottom to Top to correctly simulate overlay
	// But wait, if we extract, we overwrite. So Bottom -> Top is correct.
	// Top layer overwrites lower layer files.
	// We need to find the layer chain.

	// Get layer chain
	layerID := img.TopLayer
	var layers []string
	for layerID != "" {
		layers = append([]string{layerID}, layers...) // Prepend to reverse order later? No, append is Top-down.
		// We want Bottom-up.
		// layers = [Top, Parent, Grandparent...]
		// We want [Grandparent, Parent, Top]

		l, err := s.Store.Layer(layerID)
		if err != nil {
			return err
		}
		layerID = l.Parent
	}
	// Now layers is [Base, ..., Top] because we prepended parents.
	// So we can just iterate them in order.

	for _, lid := range layers {
		// Get layer content stream
		// Diff("", lid) gives the diff of this layer against its parent.
		// This is exactly what we want (the content of this layer).
		rc, err := s.Store.Diff("", lid, nil)
		if err != nil {
			return fmt.Errorf("failed to get diff stream for layer %s: %w", lid, err)
		}
		decompressed, err := csarchive.DecompressStream(rc)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to decompress layer %s: %w", lid, err)
		}

		if err := applyTarWithFilter(decompressed, outputDir, filter); err != nil {
			decompressed.Close()
			rc.Close()
			return fmt.Errorf("failed to apply layer %s: %w", lid, err)
		}
		decompressed.Close()
		rc.Close()
	}
	return nil
}

func applyTarWithFilter(r io.Reader, dest string, filter func(string) bool) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Clean and normalize name
		name := filepath.Clean(hdr.Name)
		// Security check
		if strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
			continue
		}

		// Determine absolute path for whitelist check
		// RPM files are absolute. Tar names are relative.
		// We prepend / to match whitelist format.
		checkName := name
		if !strings.HasPrefix(checkName, "/") {
			checkName = "/" + checkName
		}

		// 1. Handle Whiteouts (OverlayFS)
		base := filepath.Base(name)
		dir := filepath.Dir(name)
		if strings.HasPrefix(base, ".wh.") {
			realName := strings.TrimPrefix(base, ".wh.")
			if realName == ".wh.opq" {
				// Opaque whiteout: clear the directory in dest
				targetDir := filepath.Join(dest, dir)
				os.RemoveAll(targetDir)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return err
				}
				continue
			}
			// Explicit whiteout
			pathToDelete := filepath.Join(dest, dir, realName)
			os.RemoveAll(pathToDelete)
			continue
		}

		// 2. Filter check
		if !filter(checkName) {
			continue
		}

		target := filepath.Join(dest, name)

		// 3. Extract Entry
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			if err := setFileMetadata(target, hdr); err != nil {
				logrus.Debugf("Failed to set metadata for dir %s: %v", target, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
			if err := setFileMetadata(target, hdr); err != nil {
				logrus.Debugf("Failed to set metadata for file %s: %v", target, err)
			}

		case tar.TypeLink:
			linkTarget := filepath.Join(dest, hdr.Linkname)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			os.Remove(target)
			if err := os.Link(linkTarget, target); err != nil {
				return err
			}
			// Hard links share metadata with target, but we can try setting it if needed.
			// Usually redundant.

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
			if err := setFileMetadata(target, hdr); err != nil {
				logrus.Debugf("Failed to set metadata for symlink %s: %v", target, err)
			}
		}
	}
	return nil
}
