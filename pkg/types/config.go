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

package types

// Strategy defines which rule levels are enabled
type Strategy struct {
	EnableLevels []string `yaml:"enable_levels"`
}

// Match defines the criteria for a rule to trigger
type Match struct {
	Paths        []string               `yaml:"paths,omitempty"`
	PathGlobs    []string               `yaml:"path_globs,omitempty"`
	Type         string                 `yaml:"type,omitempty"` // e.g., "dir_tree", "algorithm"
	Extensions   []string               `yaml:"extensions,omitempty"`
	ExcludePaths []string               `yaml:"exclude_paths,omitempty"`
	PkgNames        []string               `yaml:"pkg_names,omitempty"`
	DependencyCheck bool                   `yaml:"dependency_check,omitempty"`
	Algorithm       string                 `yaml:"algorithm,omitempty"`
	Params       map[string]interface{} `yaml:"params,omitempty"`
}

// Rule defines a single analysis rule
type Rule struct {
	ID          string `yaml:"id"`
	Match       Match  `yaml:"match"`
	Level       string `yaml:"level"`
	Action      string `yaml:"action"`
	Description string `yaml:"description"`
}

// Config represents the full configuration for the advisor
type Config struct {
	Version   string   `yaml:"version"`
	Strategy  Strategy `yaml:"strategy"`
	Whitelist []string `yaml:"whitelist"`
	Rules     []Rule   `yaml:"rules"`
}
