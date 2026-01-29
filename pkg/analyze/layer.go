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

package analyze

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

// AnalyzeLayers traverses the image layers to collect statistics and detect redundancy
func (a *Analyzer) AnalyzeLayers(ctx context.Context) ([]types.LayerInfo, types.WasteDetection, error) {
	img, err := a.Store.Image(a.ImageRef)
	if err != nil {
		return nil, types.WasteDetection{}, fmt.Errorf("failed to find image %s: %w", a.ImageRef, err)
	}

	// 1. Build layer chain (from Top to Base)
	var layerIDs []string
	currentLayerID := img.TopLayer
	for currentLayerID != "" {
		layerIDs = append(layerIDs, currentLayerID)
		layer, err := a.Store.Layer(currentLayerID)
		if err != nil {
			return nil, types.WasteDetection{}, fmt.Errorf("failed to get layer %s: %w", currentLayerID, err)
		}
		currentLayerID = layer.Parent
	}

	// Reverse to get Base -> Top
	for i, j := 0, len(layerIDs)-1; i < j; i, j = i+1, j-1 {
		layerIDs[i], layerIDs[j] = layerIDs[j], layerIDs[i]
	}

	var layers []types.LayerInfo
	fileHashes := make(map[string]string) // Path -> Hash (Track latest version of file)
	// globalFileHashes := make(map[string][]string) // Hash -> [Path, Path] (For global dedup? No, requirement says "redundancy in different layers")
	// "Identify same files (Hash identical) added in different layers"
	// This usually means if I add file A in Layer 1, and add exact same file A in Layer 2, it's waste.
	// Or if I add file A in Layer 1, and file B in Layer 2, and A == B? No, usually refers to overwriting with same content.

	var duplicates []types.DuplicateFile

	for i, layerID := range layerIDs {
		logrus.Infof("Analyzing layer %d/%d: %s", i+1, len(layerIDs), layerID)

		info := types.LayerInfo{
			Index:  i,
			Digest: layerID,
		}

		// Get Diff stream
		diffOptions := &storage.DiffOptions{}
		rc, err := a.Store.Diff("", layerID, diffOptions)
		if err != nil {
			return nil, types.WasteDetection{}, fmt.Errorf("failed to get diff for layer %s: %w", layerID, err)
		}

		lSize, added, deleted, tFiles, dups, err := processLayerTar(rc, layerID, fileHashes)
		rc.Close()
		if err != nil {
			return nil, types.WasteDetection{}, err
		}

		duplicates = append(duplicates, dups...)

		info.Size = lSize
		info.AddedFileCount = added
		info.DeletedFileCount = deleted
		info.TopFiles = tFiles

		layers = append(layers, info)
	}

	waste := types.WasteDetection{
		Duplicates: duplicates,
	}

	return layers, waste, nil
}

func processLayerTar(r io.Reader, layerID string, fileHashes map[string]string) (int64, int, int, []types.File, []types.DuplicateFile, error) {
	tr := tar.NewReader(r)

	layerSize := int64(0)
	addedCount := 0
	deletedCount := 0
	var topFiles []types.File
	var duplicates []types.DuplicateFile

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, 0, nil, nil, fmt.Errorf("error reading tar stream: %w", err)
		}

		// Handle Whiteout files
		// Standard Whiteout: .wh.<filename>
		// Opaque Whiteout: .wh..wh.opq
		baseName := filepath.Base(header.Name)
		if strings.HasPrefix(baseName, ".wh.") {
			deletedCount++
			// If it's an opaque whiteout (.wh..wh.opq), it hides all siblings.
			// If it's a file whiteout (.wh.foo), it deletes "foo".
			// We just count them as "deletions" for now.
			continue
		}

		if header.Typeflag == tar.TypeReg {
			layerSize += header.Size
			addedCount++

			// Calculate Hash
			hasher := sha256.New()
			if _, err := io.Copy(hasher, tr); err != nil {
				logrus.Warnf("Failed to hash file %s: %v", header.Name, err)
				continue
			}
			hash := hex.EncodeToString(hasher.Sum(nil))
			filePath := "/" + header.Name // tar names are relative

			// Check for duplicate (overwriting with same content)
			if existingHash, exists := fileHashes[filePath]; exists {
				if existingHash == hash {
					// Found duplicate!
					duplicates = append(duplicates, types.DuplicateFile{
						Path:        filePath,
						Size:        header.Size,
						LayerDigest: []string{layerID}, // TODO: Track original layer too?
					})
				}
			}
			fileHashes[filePath] = hash

			topFiles = append(topFiles, types.File{Path: filePath, Size: header.Size})
		} else {
			// Handle directories, symlinks, etc. if needed
			// For size, we mainly care about regular files
		}
	}

	// Sort top files
	sort.Slice(topFiles, func(i, j int) bool {
		return topFiles[i].Size > topFiles[j].Size
	})
	if len(topFiles) > 10 {
		topFiles = topFiles[:10]
	}

	return layerSize, addedCount, deletedCount, topFiles, duplicates, nil
}
