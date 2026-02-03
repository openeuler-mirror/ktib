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

package fs

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	// 1. Build whitelist of files to keep
	keptFiles, err := s.buildFileWhitelist(imageRef, plan)
	if err != nil {
		return fmt.Errorf("failed to build file whitelist: %w", err)
	}
	logrus.Infof("Total files to keep: %d", len(keptFiles))

	// 2. Extract files from layers
	if err := s.extractLayers(imageRef, keptFiles, outputDir); err != nil {
		return fmt.Errorf("failed to extract layers: %w", err)
	}

	return nil
}

func (s *DefaultSynthesizer) buildFileWhitelist(imageRef string, plan *types.FusionPlan) (map[string]bool, error) {
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
	// We need to find which layer has /var/lib/rpm.
	// For simplicity, we can reuse extractLayers logic but only for /var/lib/rpm
	// Or we use analyze.NewAnalyzer which has logic to mount/read?
	// analyze.Analyzer uses store.Mount if possible, or we can use it to walk layers.

	// Let's implement a targeted extraction for RPM DB
	if err := s.ExtractRPMDB(imageRef, tmpDir); err != nil {
		return nil, fmt.Errorf("failed to extract RPM DB: %w", err)
	}

	// Now read the DB
	dbPath := filepath.Join(tmpDir, "var/lib/rpm")
	// Try to find the DB file
	candidates := []string{"Packages", "rpmdb.sqlite", "Packages.db"}
	var dbFile string
	for _, f := range candidates {
		p := filepath.Join(dbPath, f)
		if _, err := os.Stat(p); err == nil {
			dbFile = p
			break
		}
	}

	if dbFile == "" {
		return nil, fmt.Errorf("RPM database not found in image")
	}

	db, err := rpmdb.Open(dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open rpmdb: %w", err)
	}
	defer db.Close()

	pkgList, err := db.ListPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	whitelist := make(map[string]bool)
	keptSet := make(map[string]bool)
	for _, p := range plan.KeptPackages {
		keptSet[p] = true
	}

	for _, p := range pkgList {
		if keptSet[p.Name] {
			// Add all files of this package to whitelist
			files, err := p.InstalledFiles()
			if err != nil {
				logrus.Warnf("Failed to get files for package %s: %v", p.Name, err)
				continue
			}
			for _, f := range files {
				whitelist[f.Path] = true
			}
		}
	}

	// Add explicit kept files from plan
	for _, f := range plan.KeptFiles {
		whitelist[f] = true
	}

	return whitelist, nil
}

func (s *DefaultSynthesizer) ExtractRPMDB(imageRef string, dest string) error {
	// Similar to extractLayers but only matching /var/lib/rpm prefix
	// and we want the *latest* version of it (Top layer wins)
	// But RPM DB usually is modified in place.
	return s.extractLayersWithFilter(imageRef, dest, func(path string) bool {
		return strings.HasPrefix(path, "/var/lib/rpm")
	})
}

func (s *DefaultSynthesizer) extractLayers(imageRef string, whitelist map[string]bool, outputDir string) error {
	return s.extractLayersWithFilter(imageRef, outputDir, func(path string) bool {
		return whitelist[path]
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
