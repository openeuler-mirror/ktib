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

package config

import (
	"os"
	"path/filepath"

	"gitee.com/openeuler/ktib/pkg/utils"
	"gopkg.in/yaml.v3"
)

// FusionConfig defines the configuration for the image fusion process
type FusionConfig struct {
	Fusion FusionPolicy `yaml:"fusion"`
}

// FusionPolicy holds the specific policy rules
type FusionPolicy struct {
	KeepPackages []string `yaml:"keep_packages"`
	DropPackages []string `yaml:"drop_packages"`
	Behavior     Behavior `yaml:"behavior"`
}

// Behavior defines specific behaviors for the fusion process
type Behavior struct {
	RetainDocs     bool `yaml:"retain_docs"`
	RetainWeakDeps bool `yaml:"retain_weak_deps"`
	AutoHealLibs   bool `yaml:"auto_heal_libs"`
}

// NewDefaultConfig returns a configuration with default values
func NewDefaultConfig() *FusionConfig {
	return &FusionConfig{
		Fusion: FusionPolicy{
			KeepPackages: []string{},
			DropPackages: []string{},
			Behavior: Behavior{
				RetainDocs:     false,
				RetainWeakDeps: false,
				AutoHealLibs:   true,
			},
		},
	}
}

// LoadConfig loads configuration with multi-level merging
// Precedence: CLI Config > User Global Config (~/.ktib/fusion.yaml) > Defaults
func LoadConfig(path string) (*FusionConfig, error) {
	// 1. Start with Default
	config := NewDefaultConfig()

	// 2. Merge User Global Config
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".ktib", "fusion.yaml")
		if utils.FileExists(globalPath) {
			if err := mergeConfigFile(config, globalPath); err != nil {
				return nil, err
			}
		}
	}

	// 3. Merge Image Specific Config (CLI)
	if path != "" {
		if err := mergeConfigFile(config, path); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func mergeConfigFile(base *FusionConfig, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Strategy for merging:
	// 1. Unmarshal into a temp struct to get slices for union.
	// 2. Unmarshal into base to apply scalar overrides.
	// 3. Restore the unioned slices.

	temp := &FusionConfig{}
	if err := yaml.Unmarshal(data, temp); err != nil {
		return err
	}

	// Calculate Union of slices
	mergedKeep := uniqueStrings(append(base.Fusion.KeepPackages, temp.Fusion.KeepPackages...))
	mergedDrop := uniqueStrings(append(base.Fusion.DropPackages, temp.Fusion.DropPackages...))

	// Apply scalar overrides (and potential slice overwrites)
	if err := yaml.Unmarshal(data, base); err != nil {
		return err
	}

	// Restore Union slices
	base.Fusion.KeepPackages = mergedKeep
	base.Fusion.DropPackages = mergedDrop

	return nil
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
