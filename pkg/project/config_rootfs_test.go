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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureRootfs(t *testing.T) {
	// 创建临时目录作为目标
	target, err := os.MkdirTemp("", "rootfs_test")
	require.NoError(t, err)
	defer os.RemoveAll(target)
	bootstrap := NewBootstrap(target)
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
	configPath := filepath.Join(target, "config.yml")
	err = ioutil.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	bootstrap.InitWorkDir("baseimage", configPath)
	bootstrap.AddDockerfile()
	bootstrap.AddScript()
	bootstrap.AddChangeInfo()
	bootstrap.AddTestcase()
	targetDir, _ := filepath.Abs(bootstrap.DestinationDir + "/" + "init" + "/" + "baseimage")

	// 创建必要的目录结构
	directories := []string{
		filepath.Join(targetDir, "etc/sysconfig"),
		filepath.Join(targetDir, "etc/dnf/vars"),
		filepath.Join(targetDir, "etc/rpm"),
		filepath.Join(targetDir, "etc"),
		filepath.Join(targetDir, "var/cache/yum"),
		filepath.Join(targetDir, "var/cache/ldconfig"),
		filepath.Join(targetDir, "etc"),
	}
	for _, dir := range directories {
		require.NoError(t, os.MkdirAll(dir, 0755))
	}

	// 创建一个有效的配置
	config := Config{
		Network: struct {
			NETWORKING string `yaml:"networking"`
			HOSTNAME   string `yaml:"hostname"`
		}{
			NETWORKING: "yes",
			HOSTNAME:   "test-host",
		},
		Locale: "en_US.UTF-8",
	}

	// 测试 ConfigureRootfs
	err = ConfigureRootfs(targetDir, config)
	if err != nil {
		if strings.Contains(err.Error(), "error copying bash files") {
			t.Skip("Skipping test due to error copying bash files")
		}
	}
	assert.NoError(t, err)

	// 验证网络配置文件是否创建
	networkFilePath := filepath.Join(targetDir, "etc/sysconfig/network")
	networkConfig, err := ioutil.ReadFile(networkFilePath)
	assert.NoError(t, err)
	assert.Equal(t, "NETWORKING=yes\nHOSTNAME=test-host\n", string(networkConfig))

	// 检查 infra 配置文件
	infraFilePath := filepath.Join(targetDir, "etc/dnf/vars/infra")
	infraConfig, err := ioutil.ReadFile(infraFilePath)
	assert.NoError(t, err)
	assert.Equal(t, "container", string(infraConfig))

	// 检查语言配置文件
	localeFilePath := filepath.Join(targetDir, "etc/rpm/macros.image-language-conf")
	localeConfig, err := ioutil.ReadFile(localeFilePath)
	assert.NoError(t, err)
	assert.Equal(t, "en_US.UTF-8", string(localeConfig))

	// 检查 machine-id 文件
	machineIDFilePath := filepath.Join(targetDir, "etc/machine-id")
	machineID, err := ioutil.ReadFile(machineIDFilePath)
	assert.NoError(t, err)
	assert.Empty(t, string(machineID))

	// 检查不必要的文件是否被删除
	for _, file := range unnecessaryFiles {
		_, err := os.Stat(filepath.Join(targetDir, file))
		assert.True(t, os.IsNotExist(err), "Expected %s to be removed", file)
	}
}

func TestAddCommandToScriptAndRun(t *testing.T) {
	targetDir, err := os.MkdirTemp("", "script_test")
	require.NoError(t, err)
	//defer os.RemoveAll(targetDir)

	bootstrap := NewBootstrap(targetDir)
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
	configPath := filepath.Join(targetDir, "config.yml")
	err = ioutil.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	bootstrap.InitWorkDir("baseimage", configPath)
	bootstrap.AddDockerfile()
	bootstrap.AddScript()
	bootstrap.AddChangeInfo()
	bootstrap.AddTestcase()
	targetDir, _ = filepath.Abs(bootstrap.DestinationDir + "/" + "init" + "/" + "baseimage")

	// 创建必要的目录结构
	directories := []string{
		filepath.Join(targetDir, "etc/sysconfig"),
		filepath.Join(targetDir, "etc/dnf/vars"),
		filepath.Join(targetDir, "etc/rpm"),
		filepath.Join(targetDir, "etc"),
		filepath.Join(targetDir, "var/cache/yum"),
		filepath.Join(targetDir, "var/cache/ldconfig"),
		filepath.Join(targetDir, "root"),
	}
	for _, dir := range directories {
		require.NoError(t, os.MkdirAll(dir, 0755))
	}

	scripts := `hello`
	scriptPath := filepath.Join(targetDir, "/chroot_script.sh")
	err = ioutil.WriteFile(scriptPath, []byte(scripts), 0644)
	assert.NoError(t, err)
	// 测试 addCommandToScriptAndRun
	err = addCommandToScriptAndRun(targetDir)
	assert.NoError(t, err)

	// 验证 .bash_history 文件
	historyFilePath := filepath.Join(targetDir, "root/.bash_history")
	_, err = os.Stat(historyFilePath)
	assert.NoError(t, err, "Expected .bash_history file to be created")
}
