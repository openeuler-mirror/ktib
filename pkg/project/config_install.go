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
)

func InstallPackages(yumConfig, target string, packages ...string) error {
	cmd := exec.Command("yum", "-c", yumConfig, "--installroot="+target, "--releasever=/", "--setopt=tsflags=nodocs",
		"--setopt=group_package_types=mandatory", "--setopt=install_weak_deps=False", "-y", "install")
	cmd.Args = append(cmd.Args, packages...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Error executing command: %v\n", err)
	}

	cleanCmd := exec.Command("/usr/bin/yum", "-c", yumConfig, "--installroot="+target, "-y", "clean", "all")
	cleanCmd.Stdout = os.Stdout
	cleanCmd.Stderr = os.Stderr
	if err := cleanCmd.Run(); err != nil {
		return fmt.Errorf("Failed to clean all Packages: %v\n", err)
	}
	return nil
}
