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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/ktib/pkg/analyze/rules"
	"gitee.com/openeuler/ktib/pkg/types"
	"gopkg.in/yaml.v3"
)

// DefaultRulesPath is the default path for external rules
var DefaultRulesPath = "/etc/ktib/default_rules.yaml"

// LoadRules loads default rules and optionally merges user provided rules
func LoadRules(path string, lang string) (*types.Config, error) {
	// 1. Load defaults
	var cfg types.Config
	defaultData := rules.DefaultRules
	if lang == "zh" || lang == "zh_cn" || lang == "zh_CN" {
		defaultData = rules.DefaultRulesZH
	}

	if err := yaml.Unmarshal(defaultData, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default rules: %w", err)
	}

	// 2. Determine external config path
	loadPath := ""
	if path != "" {
		loadPath = path
	} else {
		// Check if default system config exists
		if _, err := os.Stat(DefaultRulesPath); err == nil {
			loadPath = DefaultRulesPath
		}
	}

	// 3. Load user config if provided
	if loadPath != "" {
		data, err := os.ReadFile(loadPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read rule file %s: %w", loadPath, err)
		}

		var userCfg types.Config
		if err := yaml.Unmarshal(data, &userCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal user rules: %w", err)
		}

		// Merge strategy (user overrides)
		if len(userCfg.Strategy.EnableLevels) > 0 {
			cfg.Strategy = userCfg.Strategy
		}
		// Merge whitelist
		cfg.Whitelist = append(cfg.Whitelist, userCfg.Whitelist...)

		// Merge rules (update existing or append new)
		existingIDs := make(map[string]bool)
		for _, r := range cfg.Rules {
			existingIDs[r.ID] = true
		}

		for _, rule := range userCfg.Rules {
			if existingIDs[rule.ID] {
				// Update existing rule
				for i, r := range cfg.Rules {
					if r.ID == rule.ID {
						cfg.Rules[i] = rule
						break
					}
				}
			} else {
				cfg.Rules = append(cfg.Rules, rule)
			}
		}
	}

	return &cfg, nil
}

// DumpDefaultRules writes the embedded default rules to the specified path
func DumpDefaultRules(path string) error {
	if path == "" {
		path = DefaultRulesPath
	}
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(path, rules.DefaultRules, 0644)
}

func (a *Analyzer) GenerateRecommendations(
	layers []types.LayerInfo,
	pkgs types.PackageInfo,
	fs types.FilesystemInfo,
	waste types.WasteDetection,
	mountPoint string,
	entrypoints []string,
) []types.Recommendation {
	var recs []types.Recommendation

	// Helper map for enabled levels
	enabledLevels := make(map[string]bool)
	for _, l := range a.Rules.Strategy.EnableLevels {
		enabledLevels[l] = true
	}

	// Helper for whitelist check
	isWhitelisted := func(p string) bool {
		for _, w := range a.Rules.Whitelist {
			if p == w || strings.HasPrefix(p, w) {
				return true
			}
		}
		return false
	}

	for _, rule := range a.Rules.Rules {
		// Skip if level not enabled
		if !enabledLevels[rule.Level] {
			continue
		}

		// Match Logic
		matched := false
		saving := int64(0)
		var matchedItems []string

		// 1. Path Matching (Directories)
		if len(rule.Match.Paths) > 0 {
			for _, dir := range fs.TopDirectories {
				if isWhitelisted(dir.Path) {
					continue
				}
				for _, target := range rule.Match.Paths {
					if strings.HasPrefix(dir.Path, target) {
						matched = true
						saving += dir.Size
						matchedItems = append(matchedItems, dir.Path)
					}
				}
			}
		}

		// 2. Path Globs (File patterns)
		if len(rule.Match.PathGlobs) > 0 {
			for _, dir := range fs.TopDirectories {
				if isWhitelisted(dir.Path) {
					continue
				}
				for _, pattern := range rule.Match.PathGlobs {
					if match, _ := filepath.Match(pattern, filepath.Base(dir.Path)); match {
						matched = true
						saving += dir.Size
						matchedItems = append(matchedItems, dir.Path)
					}
				}
			}
		}

		// 3. Package Name Matching
		if len(rule.Match.PkgNames) > 0 {
			// RPM
			for _, p := range pkgs.RPM {
				// Packages don't usually have paths in this context, but if we had file lists, we would check whitelist.
				// Here we check package names against whitelist? No, whitelist is for paths.
				// But maybe we should check if package is critical?
				// The doc says: "Whitelist: global whitelist (absolute path list)". So it applies to paths.
				// For packages, we assume the rule itself (blacklisting) is correct.
				for _, pattern := range rule.Match.PkgNames {
					if match, _ := filepath.Match(pattern, p.Name); match {
						matched = true
						saving += p.Size
						matchedItems = append(matchedItems, fmt.Sprintf("rpm:%s", p.Name))
					}
				}
			}
			// Python
			for _, p := range pkgs.Python {
				for _, pattern := range rule.Match.PkgNames {
					if match, _ := filepath.Match(pattern, p.Name); match {
						matched = true
						saving += p.Size
						matchedItems = append(matchedItems, fmt.Sprintf("pip:%s", p.Name))
					}
				}
			}
		}

		// 4. Extensions
		if len(rule.Match.Extensions) > 0 {
			for _, dir := range fs.TopDirectories {
				if isWhitelisted(dir.Path) {
					continue
				}
				ext := filepath.Ext(dir.Path)
				for _, target := range rule.Match.Extensions {
					if ext == target {
						matched = true
						saving += dir.Size
						matchedItems = append(matchedItems, dir.Path)
					}
				}
			}
		}

		// 5. Dependency Analysis
		if rule.Match.DependencyCheck {
			if mountPoint != "" && len(entrypoints) > 0 {
				scanner := NewDependencyScanner(mountPoint)
				requiredLibs, err := scanner.ScanDependencies(entrypoints)
				if err == nil {
					_, _, potentialSaving, unusedLibs := scanner.AssessFatSlim(requiredLibs)
					if potentialSaving > 0 {
						matched = true
						saving += potentialSaving
						matchedItems = append(matchedItems, unusedLibs...)
					}
				}
			}
		}

		if matched {
			recs = append(recs, types.Recommendation{
				Level:        rule.Level,
				Code:         rule.ID,
				Message:      rule.Description,
				Command:      rule.Action,
				Saving:       formatSize(saving),
				MatchedItems: matchedItems,
			})
		}
	}

	// Add Waste recommendation
	if len(waste.Duplicates) > 0 {
		dupSize := int64(0)
		for _, d := range waste.Duplicates {
			count := len(d.LayerDigest)
			if count > 1 {
				dupSize += d.Size * int64(count-1)
			}
		}
		recs = append(recs, types.Recommendation{
			Level:   "WARN",
			Code:    "MERGE_LAYERS",
			Message: fmt.Sprintf("Found %d duplicate files across layers.", len(waste.Duplicates)),
			Saving:  formatSize(dupSize),
		})
	}

	return recs
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
