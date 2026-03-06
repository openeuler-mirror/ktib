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
	"os"
	"path/filepath"
	"testing"

	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestLoadRules(t *testing.T) {
	// Test Default Rules Loading
	cfg, err := LoadRules("", "en")
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.Rules)
	assert.Contains(t, cfg.Strategy.EnableLevels, "SAFE")

	// Test User Config Override
	userYaml := `
version: "1.1"
strategy:
  enable_levels: ["EXPERIMENTAL"]
rules:
  - id: "TEST_RULE"
    match:
      paths: ["/test/path"]
    level: "EXPERIMENTAL"
    description: "Test Rule"
`
	tmpFile := filepath.Join(t.TempDir(), "rules.yaml")
	err = os.WriteFile(tmpFile, []byte(userYaml), 0644)
	assert.NoError(t, err)

	cfg, err = LoadRules(tmpFile, "en")
	assert.NoError(t, err)
	assert.Contains(t, cfg.Strategy.EnableLevels, "EXPERIMENTAL")

	// Check if TEST_RULE is present
	found := false
	for _, r := range cfg.Rules {
		if r.ID == "TEST_RULE" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestGenerateRecommendations(t *testing.T) {
	// Setup Analyzer with custom rules
	cfg := types.Config{
		Strategy: types.Strategy{
			EnableLevels: []string{"SAFE", "WARN"},
		},
		Rules: []types.Rule{
			{
				ID: "RM_CACHE",
				Match: types.Match{
					Paths: []string{"/var/cache"},
				},
				Level:       "SAFE",
				Description: "Remove Cache",
				Action:      "rm -rf /var/cache",
			},
			{
				ID: "RM_DOCS",
				Match: types.Match{
					PathGlobs: []string{"*.doc"},
				},
				Level:       "WARN",
				Description: "Remove Docs",
			},
			{
				ID: "RM_PKG",
				Match: types.Match{
					PkgNames: []string{"vim*"},
				},
				Level:       "SAFE",
				Description: "Remove Pkg",
			},
		},
	}

	analyzer := &Analyzer{
		Rules: cfg,
	}

	// Mock Data
	fs := types.FilesystemInfo{
		TopDirectories: []types.TopDirectory{
			{Path: "/var/cache/yum", Size: 100},
			{Path: "/usr/share/doc/readme.doc", Size: 200},
			{Path: "/usr/bin", Size: 500},
		},
	}
	pkgs := types.PackageInfo{
		RPM: []types.Package{
			{Name: "vim-enhanced", Size: 1000},
			{Name: "bash", Size: 500},
		},
	}
	waste := types.WasteDetection{}
	layers := []types.LayerInfo{}

	recs := analyzer.GenerateRecommendations(layers, pkgs, fs, waste, "", nil, types.ELFMetadata{})

	assert.Len(t, recs, 3) // Cache, Docs, Pkg

	// Check 1: Cache
	assert.Equal(t, "RM_CACHE", recs[0].Code)
	assert.Equal(t, "rm -rf /var/cache", recs[0].Command)

	// Check 2: Docs (Globs match against filepath.Base of directory path)
	// In the implementation, we match filepath.Base(dir.Path) against pattern.
	// dir.Path = "/usr/share/doc/readme.doc", Base = "readme.doc", Pattern = "*.doc" -> Match!
	assert.Equal(t, "RM_DOCS", recs[1].Code)

	// Check 3: Pkg
	// "vim-enhanced" matches "vim*"
	assert.Equal(t, "RM_PKG", recs[2].Code)
}

func TestGenerateRecommendations_OfflineDependency(t *testing.T) {
	// Setup Analyzer with custom rules
	cfg := types.Config{
		Strategy: types.Strategy{
			EnableLevels: []string{"EXPERIMENTAL"},
		},
		Rules: []types.Rule{
			{
				ID: "RM_UNUSED_LIBS",
				Match: types.Match{
					DependencyCheck: true,
				},
				Level:       "EXPERIMENTAL",
				Description: "Remove Unused Libs",
			},
		},
	}

	analyzer := &Analyzer{
		Rules: cfg,
	}

	// Mock Data
	// /bin/app depends on /lib/lib1.so
	// /lib/lib2.so is unused
	entrypoints := []string{"/bin/app"}

	elfMetadata := types.ELFMetadata{
		Dependencies: map[string][]string{
			"/bin/app": {"/lib/lib1.so"},
		},
		Libs: []types.File{
			{Path: "/lib/lib1.so", Size: 100},
			{Path: "/lib/lib2.so", Size: 200},
		},
	}

	recs := analyzer.GenerateRecommendations(nil, types.PackageInfo{}, types.FilesystemInfo{}, types.WasteDetection{}, "", entrypoints, elfMetadata)

	assert.Len(t, recs, 1)
	assert.Equal(t, "RM_UNUSED_LIBS", recs[0].Code)
	// Savings should be size of lib2.so (200)
	assert.Equal(t, "200 B", recs[0].Saving)
	assert.Contains(t, recs[0].MatchedItems, "/lib/lib2.so")
}

func TestGenerateRecommendations_Whitelist(t *testing.T) {
	// Setup Analyzer with custom rules and whitelist
	cfg := types.Config{
		Strategy: types.Strategy{
			EnableLevels: []string{"SAFE"},
		},
		Whitelist: []string{"/var/cache/yum/protected"},
		Rules: []types.Rule{
			{
				ID: "RM_CACHE",
				Match: types.Match{
					Paths: []string{"/var/cache/yum"},
				},
				Level:       "SAFE",
				Description: "Remove Cache",
			},
		},
	}

	analyzer := &Analyzer{
		Rules: cfg,
	}

	// Mock Data
	fs := types.FilesystemInfo{
		TopDirectories: []types.TopDirectory{
			{Path: "/var/cache/yum/data", Size: 100},         // Should match
			{Path: "/var/cache/yum/protected", Size: 200},    // Should be ignored
			{Path: "/var/cache/yum/protected/sub", Size: 50}, // Should be ignored
		},
	}
	pkgs := types.PackageInfo{}
	waste := types.WasteDetection{}
	layers := []types.LayerInfo{}

	recs := analyzer.GenerateRecommendations(layers, pkgs, fs, waste, "", nil, types.ELFMetadata{})

	assert.Len(t, recs, 1)
	assert.Equal(t, "RM_CACHE", recs[0].Code)
	// Only 100 bytes should be saved, not 350
	assert.Equal(t, "100 B", recs[0].Saving)
}

func TestDumpDefaultRules(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.yaml")

	err := DumpDefaultRules(path)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestLoadRules_SystemDefault(t *testing.T) {
	// Temporarily override the DefaultRulesPath
	tmpDir := t.TempDir()
	defaultPath := filepath.Join(tmpDir, "default_rules.yaml")
	originalPath := DefaultRulesPath
	DefaultRulesPath = defaultPath
	defer func() { DefaultRulesPath = originalPath }()

	// Create a dummy system default file
	systemYaml := `
version: "1.1"
strategy:
  enable_levels: ["SYSTEM_DEFAULT"]
rules: []
`
	err := os.WriteFile(defaultPath, []byte(systemYaml), 0644)
	assert.NoError(t, err)

	// Test loading with empty path (should pick up system default)
	cfg, err := LoadRules("", "en")
	assert.NoError(t, err)
	assert.Contains(t, cfg.Strategy.EnableLevels, "SYSTEM_DEFAULT")
}
