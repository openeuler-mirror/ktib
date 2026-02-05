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
package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDnfAndCreateDev(t *testing.T) {
	target := filepath.Join(t.TempDir(), "rootfs")
	err := CheckDnfAndCreateDev(target)
	if err != nil {
		t.Errorf("TestCheckDnfAndCreateDev failed:%v", err)
	}
	_, err = os.Stat(filepath.Join(target, "dev"))
	if err != nil {
		t.Errorf("TestCheckDnfAndCreateDev failed to create /dev director:%v", err)
	}
}

func TestCheckVarsFile(t *testing.T) {
	dir := t.TempDir()
	err := CheckVarsFile(dir)
	if err != nil {
		t.Errorf("Unexpected error return: %s", err)
	}
	_, err = os.Stat(filepath.Join(dir, "etc", "yum", "vars"))
	if os.IsNotExist(err) {
		t.Errorf("CheckVarsFile() failed to create /etc/yum/vars directory")
	}
}
