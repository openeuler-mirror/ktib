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
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDnfAndCreateDev(t *testing.T) {
	err := CheckDnfAndCreateDev("/tmp/test")
	if err != nil {
		t.Errorf("TestCheckDnfAndCreateDev failed:%v", err)
	}
	_, err = os.Stat("/tmp/test/dev")
	if err != nil {
		t.Errorf("TestCheckDnfAndCreateDev failed to create /dev director:%v", err)
	}
	os.RemoveAll("/tmp/test")
}

func TestCheckVarsFile(t *testing.T) {
	dir := t.TempDir()
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
	err := CheckVarsFile(dir)
	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
	_, err = os.Stat(filepath.Join(dir, "/etc/yum/vars"))
	if os.IsNotExist(err) {
		t.Errorf("CheckVarsFile() failed to create /etc/yum/vars directory")
	}
	// 如果原来的 /etc/yum/vars 目录存在，则不删除新创建的目录
	if !originalVarsExists {
		// 确保新创建的目录可以被删除
		require.NoError(t, os.RemoveAll(varsDir))
	}
}
