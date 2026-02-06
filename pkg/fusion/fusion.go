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
	"errors"
	"fmt"
	"os"
	"time"

	"gitee.com/openeuler/ktib/pkg/fusion/commit"
	"gitee.com/openeuler/ktib/pkg/fusion/config"
	"gitee.com/openeuler/ktib/pkg/fusion/fs"
	"gitee.com/openeuler/ktib/pkg/fusion/rpm"
	"gitee.com/openeuler/ktib/pkg/fusion/solver"
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	"gitee.com/openeuler/ktib/pkg/fusion/verify"
	"gitee.com/openeuler/ktib/pkg/i18n"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

// FusionManager orchestrates the image fusion process
type FusionManager struct {
	Config *config.FusionConfig
	Solver types.DependencySolver
	RPM    types.DBReconstructor
	FS     types.FSSynthesizer
	Verify types.Verifier
	Commit commit.Committer
	Lang   string

	OnProgress func(step string, done bool, duration time.Duration)
}

// NewFusionManager creates a new FusionManager
func NewFusionManager(cfg *config.FusionConfig, store storage.Store) *FusionManager {
	return &FusionManager{
		Config: cfg,
		Solver: solver.NewDefaultSolver(store),
		RPM:    rpm.NewDefaultReconstructor(""),
		FS:     fs.NewDefaultSynthesizer(store),
		Verify: verify.NewDefaultVerifier(),
		Commit: commit.NewImageBuildahCommitter(store),
	}
}

// Run executes the fusion pipeline
func (m *FusionManager) Run(imageRef string, outputDir string, targetTag string) error {
	logrus.Infof("Starting fusion for image: %s", imageRef)

	notifyProgress := func(step string, done bool, start time.Time) {
		if m.OnProgress == nil {
			return
		}
		var d time.Duration
		if done {
			d = time.Since(start)
		}
		m.OnProgress(step, done, d)
	}

	updateStep := func(step string) {
		if m.OnProgress == nil {
			return
		}
		m.OnProgress(step, false, 0)
	}

	// Phase 1: Solve Dependencies
	phaseName := i18n.T("Phase 1: Solving dependencies")
	startTime := time.Now()
	notifyProgress(phaseName, false, startTime)
	logrus.Info(phaseName + "...")
	if s, ok := m.Solver.(interface{ SetStepUpdater(func(string)) }); ok {
		s.SetStepUpdater(updateStep)
	}
	plan, err := m.Solver.Solve(imageRef, m.Config)
	if err != nil {
		return fmt.Errorf("dependency solving failed: %w", err)
	}
	notifyProgress(phaseName, true, startTime)
	logrus.Infof("Identified %d packages and %d files to keep", len(plan.KeptPackages), len(plan.KeptFiles))

	// Phase 2: Synthesize Filesystem
	phaseName = i18n.T("Phase 2: Synthesizing Filesystem")
	startTime = time.Now()
	notifyProgress(phaseName, false, startTime)
	logrus.Info(phaseName + "...")
	if err := m.FS.Synthesize(imageRef, plan, outputDir); err != nil {
		return fmt.Errorf("filesystem synthesis failed: %w", err)
	}
	notifyProgress(phaseName, true, startTime)

	// Phase 3: Reconstruct RPM DB (Best Effort)
	phaseName = i18n.T("Phase 3: Reconstructing RPM Database")
	startTime = time.Now()
	notifyProgress(phaseName, false, startTime)
	logrus.Info(phaseName + "...")

	tempRPMDB, err := os.MkdirTemp("", "ktib-fusion-rpmdb-src-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for RPM DB: %w", err)
	}
	defer os.RemoveAll(tempRPMDB)

	logrus.Debugf("Extracting original RPM DB to %s", tempRPMDB)
	if err := m.FS.ExtractRPMDB(imageRef, tempRPMDB); err != nil {
		return fmt.Errorf("failed to extract RPM DB: %w", err)
	}

	if err := m.RPM.Reconstruct(tempRPMDB, plan, outputDir); err != nil {
		logrus.Warnf("RPM DB reconstruction failed (fallback to extracted rpmdb): %v", err)
	}
	notifyProgress(phaseName, true, startTime)

	// Phase 4: Verify
	phaseName = i18n.T("Phase 4: Verifying result")
	startTime = time.Now()
	notifyProgress(phaseName, false, startTime)
	logrus.Info(phaseName + "...")
	if err := m.Verify.Verify(outputDir); err != nil {
		// Check for MissingLibsError and AutoHeal
		var missingLibsErr *verify.MissingLibsError
		if errors.As(err, &missingLibsErr) && m.Config.Fusion.Behavior.AutoHealLibs {
			logrus.Warnf("Auto-heal enabled: Attempting to recover %d missing libraries from source image...", len(missingLibsErr.Libs))

			// Extract missing libs
			if extractErr := m.FS.ExtractFiles(imageRef, missingLibsErr.Libs, outputDir); extractErr != nil {
				return fmt.Errorf("auto-heal failed to extract files: %w (original error: %v)", extractErr, err)
			}

			logrus.Info("Libraries recovered. Re-running verification...")
			if retryErr := m.Verify.Verify(outputDir); retryErr != nil {
				return fmt.Errorf("verification failed after auto-heal: %w", retryErr)
			}
			logrus.Info("Auto-heal successful!")
		} else {
			return fmt.Errorf("verification failed: %w", err)
		}
	}
	notifyProgress(phaseName, true, startTime)

	// Phase 5: Commit (Optional)
	if targetTag != "" {
		phaseName = i18n.T("Phase 5: Committing to new image")
		startTime = time.Now()
		notifyProgress(phaseName, false, startTime)
		logrus.Info(phaseName + "...")
		if err := m.Commit.Commit(outputDir, targetTag, imageRef); err != nil {
			return fmt.Errorf("commit failed: %w", err)
		}
		notifyProgress(phaseName, true, startTime)
	}

	logrus.Info("Fusion completed successfully!")
	return nil
}
