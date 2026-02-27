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

package analyze

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/storage"
	csarchive "github.com/containers/storage/pkg/archive"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type layerResult struct {
	index           int
	layerID         string
	size            int64
	addedCount      int
	deletedCount    int
	topFiles        []types.File
	duplicates      []types.DuplicateFile // Only intra-layer duplicates if any? No, we check inter-layer later.
	localFileHashes map[string]fileMeta   // Path -> Meta
	err             error
}

type fileMeta struct {
	Hash string
	Size int64
}

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

	// Parallel processing
	numLayers := len(layerIDs)
	results := make([]*layerResult, numLayers)

	// Limit concurrency
	concurrency := runtime.NumCPU()
	if concurrency < 2 {
		concurrency = 2
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for i, layerID := range layerIDs {
		index := i
		id := layerID
		g.Go(func() error {
			res, err := a.processLayer(ctx, id, index)
			if err != nil {
				return err
			}
			results[index] = res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, types.WasteDetection{}, err
	}

	// Aggregate results sequentially to detect duplicates across layers
	var layers []types.LayerInfo
	
	type fileOccurrence struct {
		hash    string
		layerID string
	}
	globalFileHashes := make(map[string]fileOccurrence)
	duplicatesMap := make(map[string]*types.DuplicateFile)

	for i := 0; i < numLayers; i++ {
		res := results[i]

		// Check for duplicates against previous layers
		if !a.Fast {
			for path, meta := range res.localFileHashes {
				if meta.Size == 0 {
					continue
				}
				if occ, exists := globalFileHashes[path]; exists {
					if occ.hash == meta.Hash {
						if df, ok := duplicatesMap[path]; ok {
							df.LayerDigest = append(df.LayerDigest, res.layerID)
						} else {
							duplicatesMap[path] = &types.DuplicateFile{
								Path:        path,
								Size:        meta.Size,
								LayerDigest: []string{occ.layerID, res.layerID},
							}
						}
					}
				}
				globalFileHashes[path] = fileOccurrence{hash: meta.Hash, layerID: res.layerID}
			}
		}

		layers = append(layers, types.LayerInfo{
			Index:            res.index,
			Digest:           res.layerID,
			Size:             res.size,
			AddedFileCount:   res.addedCount,
			DeletedFileCount: res.deletedCount,
			TopFiles:         res.topFiles,
		})
	}
	
	var allDuplicates []types.DuplicateFile
	for _, df := range duplicatesMap {
		allDuplicates = append(allDuplicates, *df)
	}
	sort.Slice(allDuplicates, func(i, j int) bool {
		return allDuplicates[i].Path < allDuplicates[j].Path
	})

	waste := types.WasteDetection{
		Duplicates: allDuplicates,
	}

	return layers, waste, nil
}

func (a *Analyzer) processLayer(ctx context.Context, layerID string, index int) (*layerResult, error) {
	logrus.Debugf("Analyzing layer %d: %s", index+1, layerID)

	diffOptions := &storage.DiffOptions{}
	rc, err := a.Store.Diff("", layerID, diffOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff for layer %s: %w", layerID, err)
	}
	defer rc.Close()

	decompressed, err := csarchive.DecompressStream(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress layer %s: %w", layerID, err)
	}
	defer decompressed.Close()

	lSize, added, deleted, tFiles, localHashes, err := processLayerTar(decompressed, a.Fast)
	if err != nil {
		return nil, err
	}

	return &layerResult{
		index:           index,
		layerID:         layerID,
		size:            lSize,
		addedCount:      added,
		deletedCount:    deleted,
		topFiles:        tFiles,
		localFileHashes: localHashes,
	}, nil
}

func processLayerTar(r io.Reader, fast bool) (int64, int, int, []types.File, map[string]fileMeta, error) {
	tr := tar.NewReader(r)

	layerSize := int64(0)
	addedCount := 0
	deletedCount := 0
	var topFiles []types.File
	localHashes := make(map[string]fileMeta)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, 0, nil, nil, fmt.Errorf("error reading tar stream: %w", err)
		}

		baseName := filepath.Base(header.Name)
		if strings.HasPrefix(baseName, ".wh.") {
			deletedCount++
			continue
		}

		if header.Typeflag == tar.TypeReg {
			layerSize += header.Size
			addedCount++
			filePath := "/" + header.Name

			if !fast {
				hasher := sha256.New()
				if _, err := io.Copy(hasher, tr); err != nil {
					logrus.Warnf("Failed to hash file %s: %v", header.Name, err)
					continue
				}
				hash := hex.EncodeToString(hasher.Sum(nil))
				localHashes[filePath] = fileMeta{
					Hash: hash,
					Size: header.Size,
				}
			} else {
				// Skip reading content
				// tr.Next() will handle skipping
			}

			topFiles = append(topFiles, types.File{Path: filePath, Size: header.Size})
		}
	}

	sort.Slice(topFiles, func(i, j int) bool {
		return topFiles[i].Size > topFiles[j].Size
	})
	if len(topFiles) > 10 {
		topFiles = topFiles[:10]
	}

	return layerSize, addedCount, deletedCount, topFiles, localHashes, nil
}
