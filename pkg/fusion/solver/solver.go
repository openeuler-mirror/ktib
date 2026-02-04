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

package solver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gitee.com/openeuler/ktib/pkg/analyze"
	"gitee.com/openeuler/ktib/pkg/fusion/config"
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	coretypes "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

type Options struct {
	FromData string
	SaveData string
}

// DefaultSolver is a basic implementation of DependencySolver
type DefaultSolver struct {
	Store    storage.Store
	FromData string
	SaveData string

	stepUpdater func(string)
}

// NewDefaultSolver creates a new DefaultSolver
func NewDefaultSolver(store storage.Store) *DefaultSolver {
	return &DefaultSolver{
		Store: store,
	}
}

func NewDefaultSolverWithOptions(store storage.Store, opts Options) *DefaultSolver {
	return &DefaultSolver{
		Store:    store,
		FromData: opts.FromData,
		SaveData: opts.SaveData,
	}
}

func (s *DefaultSolver) SetStepUpdater(fn func(string)) {
	s.stepUpdater = fn
}

// Solve calculates the list of packages and files to keep
func (s *DefaultSolver) Solve(imageRef string, cfg *config.FusionConfig) (*types.FusionPlan, error) {
	logrus.Infof("Solving dependencies for %s", imageRef)

	ctx := context.Background()

	var mountPoint string
	var allPackages []coretypes.Package

	if s.FromData != "" {
		if s.stepUpdater != nil {
			s.stepUpdater("Loading analysis data")
		}
		report, err := loadAnalysisReport(s.FromData)
		if err != nil {
			return nil, err
		}
		if report.ImageInfo.Ref != "" && report.ImageInfo.Ref != imageRef {
			logrus.Warnf("from-data image ref '%s' differs from argument '%s'", report.ImageInfo.Ref, imageRef)
		}
		allPackages = report.Analysis.Packages.RPM
	} else {
		// 1. Analyze Image to get package list
		// Use fast mode (true) to skip heavy checksums as we only need RPM metadata
		analyzer, err := analyze.NewAnalyzer(s.Store, imageRef, "", nil, true)
		if err != nil {
			return nil, fmt.Errorf("failed to create analyzer: %w", err)
		}

		// Perform analysis
		// We capture mountPoint for ELF analysis
		var onProgress func(step string, done bool, duration time.Duration)
		if s.stepUpdater != nil {
			onProgress = func(step string, done bool, duration time.Duration) {
				if done {
					return
				}
				s.stepUpdater(step)
			}
		}

		report, mp, _, cleanup, err := analyzer.Analyze(ctx, onProgress)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			return nil, fmt.Errorf("analysis failed: %w", err)
		}
		mountPoint = mp
		allPackages = report.Analysis.Packages.RPM

		if s.SaveData != "" {
			if err := saveAnalysisReport(s.SaveData, report); err != nil {
				return nil, err
			}
		}
	}

	logrus.Infof("Found %d RPM packages in the image", len(allPackages))

	if len(allPackages) == 0 {
		logrus.Warn("No RPM packages found. Fusion might result in empty image.")
	}

	// 2. Initial RPM Graph Solve
	keptList := s.solveGraph(allPackages, cfg.Fusion.KeepPackages)

	// 3. ELF Dynamic Library Analysis
	if cfg.Fusion.Behavior.AutoHealLibs {
		if mountPoint == "" {
			logrus.Warn("auto_heal_libs enabled but mount point is unavailable (from-data mode); skipping ELF analysis")
		} else {
			var err error
			keptList, err = s.solveELF(mountPoint, allPackages, keptList)
			if err != nil {
				logrus.Warnf("ELF analysis failed: %v", err)
			}
		}
	}

	logrus.Infof("Resolved %d packages to keep (from %d initial requests)", len(keptList), len(cfg.Fusion.KeepPackages))

	// 4. Collect Files
	// from-data mode does not contain per-package file lists in JSON (Files is not serialized),
	// so we skip collecting kept files here and rely on RPMDB-driven whitelist in the FS synthesizer.
	var keptFiles []string
	if mountPoint != "" {
		keptFiles = s.collectFiles(allPackages, keptList)
	}
	// Append explicitly kept files from config
	if len(cfg.Fusion.KeepFiles) > 0 {
		keptFiles = append(keptFiles, cfg.Fusion.KeepFiles...)
	}

	return &types.FusionPlan{
		KeptPackages: keptList,
		KeptFiles:    keptFiles,
	}, nil
}

func loadAnalysisReport(path string) (*coretypes.AnalysisReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read analysis data %s: %w", path, err)
	}
	report := &coretypes.AnalysisReport{}
	if err := json.Unmarshal(data, report); err != nil {
		return nil, fmt.Errorf("failed to parse analysis data %s: %w", path, err)
	}
	return report, nil
}

func saveAnalysisReport(path string, report *coretypes.AnalysisReport) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create analysis data file %s: %w", path, err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("failed to write analysis data file %s: %w", path, err)
	}
	logrus.Infof("Analysis data saved to %s", path)
	return nil
}

func (s *DefaultSolver) solveELF(mountPoint string, allPackages []coretypes.Package, keptList []string) ([]string, error) {
	scanner := analyze.NewDependencyScanner(mountPoint)

	// Map File -> Package Name
	fileToPkg := make(map[string]string)
	pkgMap := make(map[string]coretypes.Package)

	for _, p := range allPackages {
		pkgMap[p.Name] = p
		for _, f := range p.Files {
			fileToPkg[f] = p.Name
		}
	}

	currentList := keptList

	// Iterative resolution
	for i := 0; i < 5; i++ {
		logrus.Debugf("ELF Analysis Iteration %d, current packages: %d", i+1, len(currentList))

		// Build entrypoints from current kept packages
		var entrypoints []string
		keptSet := make(map[string]struct{})
		for _, name := range currentList {
			keptSet[name] = struct{}{}
			if p, ok := pkgMap[name]; ok {
				entrypoints = append(entrypoints, p.Files...)
			}
		}

		// Scan dependencies
		neededLibs, err := scanner.ScanDependencies(entrypoints)
		if err != nil {
			return currentList, err
		}

		// Resolve providers
		var added []string
		for _, libPath := range neededLibs {
			// libPath is absolute path in container
			if pkgName, ok := fileToPkg[libPath]; ok {
				if _, kept := keptSet[pkgName]; !kept {
					logrus.Infof("ELF dependency: %s provided by %s (added)", libPath, pkgName)
					added = append(added, pkgName)
					keptSet[pkgName] = struct{}{}
				}
			}
		}

		if len(added) == 0 {
			break
		}

		// Add new packages and re-solve RPM graph to satisfy their dependencies
		currentList = append(currentList, added...)
		currentList = s.solveGraph(allPackages, currentList)
	}

	return currentList, nil
}

func (s *DefaultSolver) collectFiles(allPackages []coretypes.Package, keptList []string) []string {
	var files []string
	pkgMap := make(map[string]coretypes.Package)
	for _, p := range allPackages {
		pkgMap[p.Name] = p
	}

	for _, name := range keptList {
		if p, ok := pkgMap[name]; ok {
			files = append(files, p.Files...)
		}
	}
	return files
}

// solveGraph performs the dependency resolution on a list of packages
func (s *DefaultSolver) solveGraph(allPackages []coretypes.Package, keepRequests []string) []string {
	// 2. Build Dependency Graph
	pkgMap := make(map[string]coretypes.Package)
	providers := make(map[string][]string) // Capability -> []PackageName

	for _, p := range allPackages {
		pkgMap[p.Name] = p

		// Package provides itself
		providers[p.Name] = append(providers[p.Name], p.Name)

		// Package provides specific capabilities
		for _, cap := range p.Provides {
			providers[cap] = append(providers[cap], p.Name)
		}
	}

	// 3. Solve from KeepPackages
	keptSet := make(map[string]struct{})
	queue := make([]string, 0)

	// Init queue with user requested packages
	for _, name := range keepRequests {
		if _, ok := keptSet[name]; !ok {
			// Check if the package exists directly
			if _, exists := pkgMap[name]; exists {
				keptSet[name] = struct{}{}
				queue = append(queue, name)
				continue
			}

			// If not found by name, check if it's a capability provided by someone
			if provs, ok := providers[name]; ok && len(provs) > 0 {
				// Pick the first one
				pName := provs[0]
				if _, ok := keptSet[pName]; !ok {
					keptSet[pName] = struct{}{}
					queue = append(queue, pName)
					logrus.Infof("Resolved requested '%s' to package '%s'", name, pName)
				}
				continue
			}

			logrus.Warnf("Requested keep_package '%s' not found in image", name)
		}
	}

	// Process queue (Transitive Closure)
	for len(queue) > 0 {
		currentName := queue[0]
		queue = queue[1:]

		pkg, exists := pkgMap[currentName]
		if !exists {
			continue
		}

		// Check requires
		for _, req := range pkg.Requires {
			// Skip self-requires or rpmlib(...)
			if strings.HasPrefix(req, "rpmlib(") {
				continue
			}

			// Check if already satisfied
			satisfied := false
			possibleProviders := providers[req]

			for _, pp := range possibleProviders {
				if _, ok := keptSet[pp]; ok {
					satisfied = true
					break
				}
			}
			if satisfied {
				continue
			}

			// If not satisfied, pick a provider
			if len(possibleProviders) == 0 {
				logrus.Debugf("Unmet dependency: %s requires %s", currentName, req)
				continue
			}

			// Strategy: Pick the best provider
			selectedProvider := possibleProviders[0]

			if _, ok := keptSet[selectedProvider]; !ok {
				keptSet[selectedProvider] = struct{}{}
				queue = append(queue, selectedProvider)
				logrus.Debugf("Adding %s to satisfy %s (required by %s)", selectedProvider, req, currentName)
			}
		}
	}

	// Convert keptSet to list
	keptList := make([]string, 0, len(keptSet))
	for k := range keptSet {
		keptList = append(keptList, k)
	}
	return keptList
}
