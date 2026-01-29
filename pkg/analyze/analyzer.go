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
	"context"
	"fmt"
	"math/rand"
	"time"

	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

type Analyzer struct {
	Store    storage.Store
	ImageRef string
}

func NewAnalyzer(store storage.Store, imageRef string) *Analyzer {
	return &Analyzer{
		Store:    store,
		ImageRef: imageRef,
	}
}

func (a *Analyzer) Run(ctx context.Context) (*types.AnalysisReport, error) {
	logrus.Infof("Starting analysis for image: %s", a.ImageRef)

	// 1. Layer Analysis
	layers, waste, err := a.AnalyzeLayers(ctx)
	if err != nil {
		return nil, fmt.Errorf("layer analysis failed: %w", err)
	}

	// 2. Mount for Package & FS Analysis
	b, mountPoint, err := a.mount()
	if err != nil {
		return nil, fmt.Errorf("failed to mount image: %w", err)
	}
	defer a.cleanup(b)

	// 3. Package Analysis
	pkgs, err := a.AnalyzePackages(ctx, mountPoint)
	if err != nil {
		return nil, fmt.Errorf("package analysis failed: %w", err)
	}

	// 4. Filesystem Analysis
	fsInfo, arch, err := a.AnalyzeFilesystem(ctx, mountPoint)
	if err != nil {
		return nil, fmt.Errorf("filesystem analysis failed: %w", err)
	}

	// 5. Advisor
	recs := a.GenerateRecommendations(layers, pkgs, fsInfo, waste)

	// Calculate total size from layers
	totalSize := int64(0)
	for _, l := range layers {
		totalSize += l.Size
	}

	report := &types.AnalysisReport{
		ImageInfo: types.ImageInfo{
			Ref:          a.ImageRef,
			Size:         totalSize,
			Created:      time.Now(), // TODO: Get from image metadata
			OS:           "linux",    // TODO: Get from image metadata
			Architecture: arch,
		},
		Analysis: types.AnalysisData{
			Layers:         layers,
			Packages:       pkgs,
			Filesystem:     fsInfo,
			WasteDetection: waste,
		},
		Recommendations: recs,
	}

	return report, nil
}

func (a *Analyzer) mount() (*builder.Builder, string, error) {
	rand.Seed(time.Now().UnixNano())
	containerName := fmt.Sprintf("%s-analyze-%d", "ktib", rand.Int())

	opts := builder.BuilderOptions{
		FromImage: a.ImageRef,
		Container: containerName,
	}

	b, err := builder.NewBuilder(a.Store, opts)
	if err != nil {
		return nil, "", err
	}

	if err := b.Mount(""); err != nil {
		b.Remove(options.RemoveOption{Force: true})
		return nil, "", err
	}

	return b, b.MountPoint, nil
}

func (a *Analyzer) cleanup(b *builder.Builder) {
	if b == nil {
		return
	}
	if err := b.UMount(); err != nil {
		logrus.Warnf("Failed to unmount builder %s: %v", b.ID, err)
	}
	if err := b.Remove(options.RemoveOption{Force: true}); err != nil {
		logrus.Warnf("Failed to remove builder %s: %v", b.ID, err)
	}
}
