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

package fusion

import (
	"fmt"

	"gitee.com/openeuler/ktib/pkg/fusion/config"
	"gitee.com/openeuler/ktib/pkg/fusion/fs"
	"gitee.com/openeuler/ktib/pkg/fusion/rpm"
	"gitee.com/openeuler/ktib/pkg/fusion/solver"
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	"gitee.com/openeuler/ktib/pkg/fusion/verify"
	"github.com/sirupsen/logrus"
)

// FusionManager orchestrates the image fusion process
type FusionManager struct {
	Config *config.FusionConfig
	Solver types.DependencySolver
	RPM    types.DBReconstructor
	FS     types.FSSynthesizer
	Verify types.Verifier
}

// NewFusionManager creates a new FusionManager
func NewFusionManager(cfg *config.FusionConfig) *FusionManager {
	return &FusionManager{
		Config: cfg,
		Solver: solver.NewDefaultSolver(),
		RPM:    rpm.NewDefaultReconstructor(),
		FS:     fs.NewDefaultSynthesizer(),
		Verify: verify.NewDefaultVerifier(),
	}
}

// Run executes the fusion pipeline
func (m *FusionManager) Run(imageRef string, outputDir string) error {
	logrus.Infof("Starting fusion for image: %s", imageRef)

	// Phase 1: Solve Dependencies
	logrus.Info("Phase 1: Solving dependencies...")
	plan, err := m.Solver.Solve(imageRef, m.Config)
	if err != nil {
		return fmt.Errorf("dependency solving failed: %w", err)
	}
	logrus.Infof("Identified %d packages and %d files to keep", len(plan.KeptPackages), len(plan.KeptFiles))

	// Phase 2: Reconstruct RPM DB
	logrus.Info("Phase 2: Reconstructing RPM Database...")
	if err := m.RPM.Reconstruct(plan, outputDir); err != nil {
		return fmt.Errorf("RPM DB reconstruction failed: %w", err)
	}

	// Phase 3: Synthesize Filesystem
	logrus.Info("Phase 3: Synthesizing Filesystem...")
	if err := m.FS.Synthesize(plan, outputDir); err != nil {
		return fmt.Errorf("filesystem synthesis failed: %w", err)
	}

	// Phase 4: Verify
	logrus.Info("Phase 4: Verifying result...")
	if err := m.Verify.Verify(outputDir); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	logrus.Info("Fusion completed successfully!")
	return nil
}
