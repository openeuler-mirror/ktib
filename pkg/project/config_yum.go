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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func CheckDnfAndCreateDev(target string) error {
	if _, err := os.Stat("/etc/dnf/dnf.conf"); err == nil {
		yumConfig = "/etc/dnf/dnf.conf"
		setAlias("yum", "dnf")
	}
	// Remove target directory if exists and create new
	err := os.RemoveAll(target)
	if err != nil {
		return fmt.Errorf("failed to remove target directory:%v", err)
	}
	if err := os.MkdirAll(target+"/dev", 0755); err != nil {
		return fmt.Errorf("failed to create dev directory:%v", err)
	}
	return nil
}

func setAlias(alias, command string) error {
	cmd := exec.Command("alias", alias+"="+command)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func CheckVarsFile(target string) error {
	varsDir := filepath.Join(target, "etc", "yum", "vars")
	if err := os.MkdirAll(varsDir, 0755); err != nil {
		return fmt.Errorf("failed to create yum vars directory: %v", err)
	}

	if _, err := os.Stat("/etc/yum/vars"); err == nil {
		cmd := exec.Command("/usr/bin/cp", "-a", "/etc/yum/vars/.", varsDir)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to copy /etc/yum/vars: %v", err)
		}
	}
	return nil
}
