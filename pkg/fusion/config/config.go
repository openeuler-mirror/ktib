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

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*FusionConfig, error) {
	if path == "" {
		return NewDefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := NewDefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}
