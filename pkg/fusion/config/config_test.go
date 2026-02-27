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

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// 1. Test Default
	cfg, err := LoadConfig("")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.True(t, cfg.Fusion.Behavior.AutoHealLibs)

	// 2. Test File Loading
	tmpFile, err := os.CreateTemp("", "fusion_config_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `
fusion:
  keep_packages:
    - nginx
  keep_files:
    - /etc/hosts
  behavior:
    retain_docs: true
`
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err = LoadConfig(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, []string{"nginx"}, cfg.Fusion.KeepPackages)
	assert.Equal(t, []string{"/etc/hosts"}, cfg.Fusion.KeepFiles)
	assert.True(t, cfg.Fusion.Behavior.RetainDocs)
	// Default value for unspecified field (should be false/zero value of type if not merged properly,
	// but yaml.Unmarshal overwrites. Our LoadConfig currently creates default then unmarshals,
	// so fields not in yaml remain default?
	// Wait, yaml.Unmarshal into struct with values might overwrite them with zero if not present?
	// No, Unmarshal only updates fields present in YAML.
	assert.True(t, cfg.Fusion.Behavior.AutoHealLibs) // This ensures merge logic works (initialized with defaults)
}

func TestNewExampleConfig(t *testing.T) {
	cfg := NewExampleConfig()
	assert.NotNil(t, cfg)
	assert.Contains(t, cfg.Fusion.KeepPackages, "bash")
	assert.Contains(t, cfg.Fusion.KeepFiles, "/etc/resolv.conf")
}
