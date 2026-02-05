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
	"github.com/containers/common/libimage"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

type Analyzer struct {
	Store    storage.Store
	ImageRef string
	Rules    types.Config
	Fast     bool
}

func NewAnalyzer(store storage.Store, imageRef string, rulesPath string, levels []string, fast bool) (*Analyzer, error) {
	rules, err := LoadRules(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}

	// Override levels if provided
	if len(levels) > 0 {
		rules.Strategy.EnableLevels = levels
	}

	return &Analyzer{
		Store:    store,
		ImageRef: imageRef,
		Rules:    *rules,
		Fast:     fast,
	}, nil
}

func (a *Analyzer) Run(ctx context.Context, onProgress func(step string, done bool, duration time.Duration)) (*types.AnalysisReport, error) {
	// 1. Run Analysis (Data Collection)
	report, mountPoint, entrypoints, cleanup, err := a.Analyze(ctx, onProgress)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// 2. Run Advisor (Recommendation Generation)
	stepName := "Advisor Generation"
	notifyProgress := func(step string, done bool, start time.Time) {
		if onProgress != nil {
			var d time.Duration
			if done {
				d = time.Since(start)
			}
			onProgress(step, done, d)
		}
	}

	startTime := time.Now()
	notifyProgress(stepName, false, startTime)
	
	recs := a.GenerateRecommendations(
		report.Analysis.Layers,
		report.Analysis.Packages,
		report.Analysis.Filesystem,
		report.Analysis.WasteDetection,
		mountPoint,
		entrypoints,
	)
	report.Recommendations = recs
	
	notifyProgress(stepName, true, startTime)

	return report, nil
}

// Analyze performs the data collection phase. 
// It returns the report, mount point, entrypoints, a cleanup function, and any error.
// The caller IS RESPONSIBLE for calling the cleanup function to unmount the image.
func (a *Analyzer) Analyze(ctx context.Context, onProgress func(step string, done bool, duration time.Duration)) (*types.AnalysisReport, string, []string, func(), error) {
	logrus.Infof("Starting analysis for image: %s", a.ImageRef)

	// Helper for progress reporting
	notifyProgress := func(step string, done bool, start time.Time) {
		if onProgress != nil {
			var d time.Duration
			if done {
				d = time.Since(start)
			}
			onProgress(step, done, d)
		}
	}

	noopCleanup := func() {}

	// 1. Layer Analysis
	stepName := "Layer Analysis"
	startTime := time.Now()
	notifyProgress(stepName, false, startTime)
	layers, waste, err := a.AnalyzeLayers(ctx)
	if err != nil {
		return nil, "", nil, noopCleanup, fmt.Errorf("layer analysis failed: %w", err)
	}
	notifyProgress(stepName, true, startTime)

	// 2. Mount for Package & FS Analysis
	stepName = "Image Mount"
	startTime = time.Now()
	notifyProgress(stepName, false, startTime)
	b, mountPoint, err := a.mount()
	if err != nil {
		return nil, "", nil, noopCleanup, fmt.Errorf("failed to mount image: %w", err)
	}
	// We do NOT defer cleanup here; we pass it back.
	cleanup := func() {
		a.cleanup(b)
	}
	notifyProgress(stepName, true, startTime)

	// 3. Package Analysis
	stepName = "Package Analysis"
	startTime = time.Now()
	notifyProgress(stepName, false, startTime)
	pkgs, err := a.AnalyzePackages(ctx, mountPoint)
	if err != nil {
		cleanup() // cleanup if we fail here
		return nil, "", nil, noopCleanup, fmt.Errorf("package analysis failed: %w", err)
	}
	notifyProgress(stepName, true, startTime)

	// 4. Filesystem Analysis
	stepName = "Filesystem Analysis"
	startTime = time.Now()
	notifyProgress(stepName, false, startTime)
	fsInfo, arch, err := a.AnalyzeFilesystem(ctx, mountPoint)
	if err != nil {
		cleanup() // cleanup if we fail here
		return nil, "", nil, noopCleanup, fmt.Errorf("filesystem analysis failed: %w", err)
	}
	notifyProgress(stepName, true, startTime)

	// Get Image Config
	imgConfig, err := a.getImageConfig(ctx)
	if err != nil {
		logrus.Debugf("Failed to get image config: %v", err)
	}

	// Calculate entrypoints for dependency analysis
	var entrypoints []string
	entrypoints = append(entrypoints, imgConfig.Entrypoint...)
	entrypoints = append(entrypoints, imgConfig.Cmd...)

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
			Config:       imgConfig,
		},
		Analysis: types.AnalysisData{
			Layers:         layers,
			Packages:       pkgs,
			Filesystem:     fsInfo,
			WasteDetection: waste,
		},
	}

	return report, mountPoint, entrypoints, cleanup, nil
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

func (a *Analyzer) getImageConfig(ctx context.Context) (types.ImageConfig, error) {
	runtime, err := libimage.RuntimeFromStore(a.Store, &libimage.RuntimeOptions{})
	if err != nil {
		return types.ImageConfig{}, err
	}

	img, _, err := runtime.LookupImage(a.ImageRef, nil)
	if err != nil {
		return types.ImageConfig{}, err
	}

	data, err := img.Inspect(ctx, nil)
	if err != nil {
		return types.ImageConfig{}, err
	}

	if data.Config == nil {
		return types.ImageConfig{}, nil
	}

	return types.ImageConfig{
		Cmd:        data.Config.Cmd,
		Entrypoint: data.Config.Entrypoint,
		Env:        data.Config.Env,
		WorkingDir: data.Config.WorkingDir,
	}, nil
}
