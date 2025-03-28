#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

package project

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestNewBootstrap(t *testing.T) {
	dir := t.TempDir()
	bootstrap := NewBootstrap(dir)

	assert.Equal(t, dir, bootstrap.DestinationDir)
}
func TestBootstrap_InitWorkDir(t *testing.T) {
	dir := t.TempDir()
	bootstrap := NewBootstrap(dir)
	// 检查 /etc/yum/vars 是否存在，如果不存在则创建
	varsDir := "/etc/yum/vars"
	originalVarsExists := false

	if _, err := os.Stat(varsDir); os.IsNotExist(err) {
		// 创建 /etc/yum/vars 目录
		require.NoError(t, os.MkdirAll(varsDir, 0755))
		defer os.RemoveAll(varsDir) // 测试结束时删除
	} else {
		originalVarsExists = true
	}
	// 创建一个有效的配置文件
	configContent := `
packages:
  install_pkgs:
    - vim
    - curl
network:
  networking: "yes"
  hostname: "test-host"
locale: "en_US.UTF-8"
`
	configPath := filepath.Join(dir, "config.yml")
	err := ioutil.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 测试 baseimage 类型
	bootstrap.InitWorkDir("baseimage", configPath)
	expectedDirs := []string{
		filepath.Join(dir, "init", "baseimage"),
	}
	for _, d := range expectedDirs {
		_, err := os.Stat(d)
		assert.NoError(t, err, "Expected directory to exist: %s", d)
	}
	// 测试 baseimage 类型
	bootstrap.InitWorkDir("appimage", configPath)
	expectedAppDirs := []string{
		filepath.Join(dir, "init", "appimage"),
	}
	for _, d := range expectedAppDirs {
		_, err := os.Stat(d)
		assert.NoError(t, err, "Expected directory to exist: %s", d)
	}

	// 如果原来的 /etc/yum/vars 目录存在，则不删除新创建的目录
	if !originalVarsExists {
		// 确保新创建的目录可以被删除
		require.NoError(t, os.RemoveAll(varsDir))
	}
}
func TestBootstrap_AddDockerfile(t *testing.T) {
	dir := t.TempDir()
	bootstrap := NewBootstrap(dir)

	bootstrap.AddDockerfile()

	// 验证 Dockerfile 是否被创建
	dockerfilePath := filepath.Join(dir, "docker-build", "Dockerfile")
	_, err := os.Stat(dockerfilePath)
	assert.NoError(t, err, "Expected Dockerfile to be created")
}
func TestBootstrap_AddChangeInfo(t *testing.T) {
	dir := t.TempDir()
	bootstrap := NewBootstrap(dir)

	bootstrap.AddChangeInfo()

	// 验证 README 文件是否被创建
	readmePath := filepath.Join(dir, "README")
	_, err := os.Stat(readmePath)
	assert.NoError(t, err, "Expected README to be created")
}
func TestBootstrap_AddChangeInfo_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	bootstrap := NewBootstrap(dir)

	// 先创建文件
	readmePath := filepath.Join(dir, "README")
	err := ioutil.WriteFile(readmePath, []byte("This is an existing README"), 0600)
	require.NoError(t, err)

	// 调用 AddChangeInfo 方法
	bootstrap.AddChangeInfo()

	// 检查文件是否仍然存在且内容未变（意味着没有重新创建）
	content, err := ioutil.ReadFile(readmePath)
	require.NoError(t, err)
	assert.Equal(t, "This is an existing README", string(content))
}
func TestConfigParsing(t *testing.T) {
	yamlContent := `
packages:
  install_pkgs:
    - vim
    - curl
network:
  networking: "yes"
  hostname: "test-host"
locale: "en_US.UTF-8"
`
	var config Config
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	require.NoError(t, err)

	assert.Equal(t, []string{"vim", "curl"}, config.Packages.InstallPkgs)
	assert.Equal(t, "yes", config.Network.NETWORKING)
	assert.Equal(t, "test-host", config.Network.HOSTNAME)
	assert.Equal(t, "en_US.UTF-8", config.Locale)
}
func TestBootstrap_AddScript(t *testing.T) {
	dir := t.TempDir()
	bootstrap := NewBootstrap(dir)
	bootstrap.AddScript()
	assert.DirExists(t, dir+"/scripts")
	assert.FileExists(t, dir+"/scripts/Script")
}
func TestBootstrap_AddTestcase(t *testing.T) {
	dir := t.TempDir()
	bootstrap := NewBootstrap(dir)
	bootstrap.AddTestcase()
	assert.DirExists(t, dir+"/test")
	assert.FileExists(t, dir+"/test/Testcase")
}
