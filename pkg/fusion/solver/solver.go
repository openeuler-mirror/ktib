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
	"fmt"
	"strings"

	"gitee.com/openeuler/ktib/pkg/analyze"
	"gitee.com/openeuler/ktib/pkg/fusion/config"
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	coretypes "gitee.com/openeuler/ktib/pkg/types"
	"github.com/containers/storage"
	"github.com/sirupsen/logrus"
)

// DefaultSolver is a basic implementation of DependencySolver
type DefaultSolver struct {
	Store storage.Store
}

// NewDefaultSolver creates a new DefaultSolver
func NewDefaultSolver(store storage.Store) *DefaultSolver {
	return &DefaultSolver{
		Store: store,
	}
}

// Solve calculates the list of packages and files to keep
func (s *DefaultSolver) Solve(imageRef string, cfg *config.FusionConfig) (*types.FusionPlan, error) {
	logrus.Infof("Solving dependencies for %s", imageRef)

	// 1. Analyze Image to get package list
	// Use fast mode (true) to skip heavy checksums as we only need RPM metadata
	analyzer, err := analyze.NewAnalyzer(s.Store, imageRef, "", nil, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create analyzer: %w", err)
	}

	ctx := context.Background()

	// Perform analysis
	// We ignore mountPoint and entrypoints for now, as we focus on RPM DB
	report, _, _, cleanup, err := analyzer.Analyze(ctx, nil)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	allPackages := report.Analysis.Packages.RPM
	logrus.Infof("Found %d RPM packages in the image", len(allPackages))

	if len(allPackages) == 0 {
		logrus.Warn("No RPM packages found. Fusion might result in empty image.")
	}

	keptList := s.solveGraph(allPackages, cfg.Fusion.KeepPackages)

	logrus.Infof("Resolved %d packages to keep (from %d initial requests)", len(keptList), len(cfg.Fusion.KeepPackages))

	return &types.FusionPlan{
		KeptPackages: keptList,
		KeptFiles:    []string{}, // TODO: Implement file-level dependency solving if needed
	}, nil
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
