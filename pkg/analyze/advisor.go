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

// LoadRules loads default rules and optionally merges user provided rules
func LoadRules(path string) (*types.Config, error) {
	// 1. Load defaults
	var cfg types.Config
	if err := yaml.Unmarshal(rules.DefaultRules, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default rules: %w", err)
	}

	// 2. Load user config if provided
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read rule file %s: %w", path, err)
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

func (a *Analyzer) GenerateRecommendations(
	layers []types.LayerInfo,
	pkgs types.PackageInfo,
	fs types.FilesystemInfo,
	waste types.WasteDetection,
) []types.Recommendation {
	var recs []types.Recommendation

	// Helper map for enabled levels
	enabledLevels := make(map[string]bool)
	for _, l := range a.Rules.Strategy.EnableLevels {
		enabledLevels[l] = true
	}

	for _, rule := range a.Rules.Rules {
		// Skip if level not enabled
		if !enabledLevels[rule.Level] {
			continue
		}

		// Match Logic
		matched := false
		saving := int64(0)

		// 1. Path Matching (Directories)
		if len(rule.Match.Paths) > 0 {
			for _, dir := range fs.TopDirectories {
				for _, target := range rule.Match.Paths {
					if strings.HasPrefix(dir.Path, target) {
						matched = true
						saving += dir.Size
					}
				}
			}
		}

		// 2. Path Globs (File patterns)
		if len(rule.Match.PathGlobs) > 0 {
			for _, dir := range fs.TopDirectories {
				for _, pattern := range rule.Match.PathGlobs {
					if match, _ := filepath.Match(pattern, filepath.Base(dir.Path)); match {
						matched = true
						saving += dir.Size
					}
				}
			}
		}

		// 3. Package Name Matching
		if len(rule.Match.PkgNames) > 0 {
			// RPM
			for _, p := range pkgs.RPM {
				for _, pattern := range rule.Match.PkgNames {
					if match, _ := filepath.Match(pattern, p.Name); match {
						matched = true
						saving += p.Size
					}
				}
			}
			// Python
			for _, p := range pkgs.Python {
				for _, pattern := range rule.Match.PkgNames {
					if match, _ := filepath.Match(pattern, p.Name); match {
						matched = true
						saving += p.Size
					}
				}
			}
		}

		// 4. Extensions
		if len(rule.Match.Extensions) > 0 {
			for _, dir := range fs.TopDirectories {
				ext := filepath.Ext(dir.Path)
				for _, target := range rule.Match.Extensions {
					if ext == target {
						matched = true
						saving += dir.Size
					}
				}
			}
		}

		if matched {
			recs = append(recs, types.Recommendation{
				Level:   rule.Level,
				Code:    rule.ID,
				Message: rule.Description,
				Command: rule.Action,
				Saving:  formatSize(saving),
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
