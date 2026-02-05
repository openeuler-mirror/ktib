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
	"path/filepath"
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
	var imageConfig coretypes.ImageConfig

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
		// Append Python packages to allPackages
		allPackages = append(allPackages, report.Analysis.Packages.Python...)
		imageConfig = report.ImageInfo.Config
	} else {
		// 1. Analyze Image to get package list
		// Use fast mode (true) to skip heavy checksums as we only need RPM metadata
		analyzer, err := analyze.NewAnalyzer(s.Store, imageRef, "", nil, true, "")
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
		// Append Python packages to allPackages
		allPackages = append(allPackages, report.Analysis.Packages.Python...)
		imageConfig = report.ImageInfo.Config

		if s.SaveData != "" {
			if err := saveAnalysisReport(s.SaveData, report); err != nil {
				return nil, err
			}
		}
	}

	// Validate data integrity (check if it's a pruned report)
	if len(allPackages) > 0 {
		// Check the first few packages to see if they have essential metadata
		hasMetadata := false
		checks := 5
		if len(allPackages) < checks {
			checks = len(allPackages)
		}
		for i := 0; i < checks; i++ {
			if len(allPackages[i].Files) > 0 || len(allPackages[i].Requires) > 0 {
				hasMetadata = true
				break
			}
		}
		if !hasMetadata {
			return nil, fmt.Errorf("loaded analysis data appears to be a pruned report (missing dependency/file info). Please use 'ktib analyze --save-data' to generate full data for fusion")
		}
	}

	logrus.Infof("Found %d packages (RPM+Python) in the image", len(allPackages))

	if len(allPackages) == 0 {
		logrus.Warn("No packages found. Fusion might result in empty image.")
	}

	// P0: App Anchor Analysis
	if len(imageConfig.Cmd) > 0 || len(imageConfig.Entrypoint) > 0 {
		anchors := s.resolveAppAnchors(imageConfig, allPackages)
		if len(anchors) > 0 {
			logrus.Infof("Automatically resolved application anchors: %v", anchors)
			cfg.Fusion.KeepPackages = append(cfg.Fusion.KeepPackages, anchors...)
		}
	}

	// P1: Ensure Vital Paths (Base System Integrity)
	vitalPackages := []string{"filesystem", "setup", "basesystem"}
	for _, v := range vitalPackages {
		// Only add if it exists in the image
		for _, p := range allPackages {
			if p.Name == v {
				cfg.Fusion.KeepPackages = append(cfg.Fusion.KeepPackages, v)
				break
			}
		}
	}

	// Deduplicate
	cfg.Fusion.KeepPackages = uniqueStrings(cfg.Fusion.KeepPackages)

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
	// Collect files from ALL kept packages (RPM + Python)
	// Now that Package.Files is populated even for Python (and RPM if analyzing), we can use it.
	// For from-data, Package.Files is loaded from JSON.
	var keptFiles []string
	keptFiles = s.collectFiles(allPackages, keptList)

	// Append explicitly kept files from config
	if len(cfg.Fusion.KeepFiles) > 0 {
		keptFiles = append(keptFiles, cfg.Fusion.KeepFiles...)
	}

	return &types.FusionPlan{
		KeptPackages: keptList,
		KeptFiles:    keptFiles,
		Config:       cfg,
	}, nil
}

func (s *DefaultSolver) resolveAppAnchors(cfg coretypes.ImageConfig, allPackages []coretypes.Package) []string {
	var anchors []string

	// Helper to find package owning a file
	findOwner := func(path string) string {
		// Normalize path
		// If not absolute, prepend WorkingDir or "/"
		absPath := path
		if !strings.HasPrefix(path, "/") {
			wd := cfg.WorkingDir
			if wd == "" {
				wd = "/"
			}
			absPath = filepath.Join(wd, path)
		}
		absPath = filepath.Clean(absPath)

		// Search in packages
		for _, p := range allPackages {
			for _, f := range p.Files {
				if f == absPath {
					return p.Name
				}
			}
		}

		// Heuristic: If it's a command name without path, check common bin dirs
		if !strings.Contains(path, "/") {
			candidates := []string{
				"/usr/bin/" + path,
				"/bin/" + path,
				"/usr/local/bin/" + path,
				"/usr/sbin/" + path,
				"/sbin/" + path,
			}
			for _, c := range candidates {
				for _, p := range allPackages {
					for _, f := range p.Files {
						if f == c {
							return p.Name
						}
					}
				}
			}
		}

		return ""
	}

	// Helper to analyze command arguments (handles shell wrapping)
	var analyzeArgs func([]string)
	analyzeArgs = func(args []string) {
		if len(args) == 0 {
			return
		}

		// 1. Keep the direct command
		if owner := findOwner(args[0]); owner != "" {
			anchors = append(anchors, owner)
		}

		// 2. Check for shell wrapping
		exe := filepath.Base(args[0])
		if exe == "sh" || exe == "bash" || exe == "zsh" || exe == "dash" || exe == "busybox" {
			for i, arg := range args {
				if arg == "-c" && i+1 < len(args) {
					// Found shell command string
					cmdStr := args[i+1]
					parts := strings.Fields(cmdStr)
					if len(parts) > 0 {
						// Recursively check the inner command
						analyzeArgs(parts)
					}
				}
			}
		}

		// 3. Special handling for Python
		if exe == "python" || exe == "python3" {
			// python -m <module>
			for i, arg := range args {
				if arg == "-m" && i+1 < len(args) {
					moduleName := args[i+1]
					// Find package owning this module
					// Module name to file path:
					// foo.bar -> foo/bar/__init__.py or foo/bar.py or foo/__init__.py (if bar is function)
					// We just search for simple mapping first:
					// foo -> foo/__init__.py or foo.py

					// Convert module to path segments
					modPath := strings.ReplaceAll(moduleName, ".", "/")

					// Candidates to search in Files
					candidates := []string{
						modPath + ".py",
						modPath + "/__init__.py",
					}

					// Scan all packages
					found := false
					for _, p := range allPackages {
						for _, f := range p.Files {
							for _, cand := range candidates {
								if strings.HasSuffix(f, cand) {
									anchors = append(anchors, p.Name)
									logrus.Infof("Resolved python module '%s' to package '%s'", moduleName, p.Name)
									found = true
									break
								}
							}
							if found {
								break
							}
						}
						if found {
							break
						}
					}
				}
				// python script.py
				// If argument ends with .py and is not a flag
				if strings.HasSuffix(arg, ".py") && !strings.HasPrefix(arg, "-") {
					if owner := findOwner(arg); owner != "" {
						anchors = append(anchors, owner)
						logrus.Infof("Resolved python script '%s' to package '%s'", arg, owner)
					}
				}
			}
			// Add python itself as an anchor
			if owner := findOwner(exe); owner != "" {
				// Only add if not already added
				alreadyAdded := false
				for _, a := range anchors {
					if a == owner {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					anchors = append(anchors, owner)
				}
			}
		}
	}

	// Check Entrypoint
	if len(cfg.Entrypoint) > 0 {
		analyzeArgs(cfg.Entrypoint)
	}

	// Check Cmd
	if len(cfg.Cmd) > 0 {
		analyzeArgs(cfg.Cmd)
	}

	return anchors
}

func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
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
