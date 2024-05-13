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
	"path/filepath"
	"strings"
	"text/template"

	"gitee.com/openeuler/ktib/pkg/templates"
	"github.com/sirupsen/logrus"
)

var yumConfig = "/etc/yum.conf"

type Bootstrap struct {
	DestinationDir string
	ImageName      string
}

func NewBootstrap(dir, imageName string) *Bootstrap {
	return &Bootstrap{DestinationDir: dir, ImageName: imageName}
}

func (b *Bootstrap) InitWorkDir(types, imageName string) {
	switch types {
	case "baseimage":
		target, _ := filepath.Abs(b.DestinationDir + "/" + "init" + "/" + "baseimage")
		//Check if dnf is available and create new dev directory
		if err := CheckDnfAndCreateDev(target); err != nil {
			fmt.Printf("Failed to checkDnfAndCreateDev: %v", err)
			return
		}

		//create char device and fifo device
		devices := DefaultDevices()
		for _, dev := range devices {
			if dev.Type == "c" {
				CreateCharDevice(target, dev.Name, dev.Type, dev.Major, dev.Minor, dev.Mode)
			} else if dev.Type == "fifo" {
				CreateFifoDevice(target, dev.Name)
			}
		}

		//Check if the/etc/yum/vars directory exists
		if err := CheckVarsFile(target); err != nil {
			fmt.Printf("Failed to checkVarsFile: %v", err)
			return
		}

		//Install different installation packages base to image type
		yumInstaller := YumPackageInstaller{}
		if strings.Contains(imageName, "micro") {
			err := yumInstaller.installPackages("micro", yumConfig, target, "bash", "coreutils-single")
			if err != nil {
				fmt.Printf("Error Install micro rpm :%v\n", err)
			}
		} else if strings.Contains(imageName, "minimal") {
			err := yumInstaller.installPackages("minimal", yumConfig, target, "microdnf", "vim-minimal", "iproute")
			if err != nil {
				fmt.Printf("Error Install minimal rpm :%v\n", err)
			}
		} else {
			err := yumInstaller.installPackages("default", yumConfig, target, "yum", "iproute", "vim-minimal", "procps-ng", "passwd")
			if err != nil {
				fmt.Printf("Error Install default rpm :%v\n", err)
			}
		}

		//Configure network settings、dnf variable、en_US.UTF-8 locale files、machine-id、delete unnecessary configurations、cp bash and time zone
		if err := ConfigureRootfs(target); err != nil {
			fmt.Printf("Error configuring system:%v", err)
			return
		}
	default:
		os.MkdirAll(b.DestinationDir+"/"+"init"+"/"+"appimage", 0700)
	}
}

func (b *Bootstrap) AddDockerfile() {
	os.MkdirAll(b.DestinationDir+"/"+"docker-build", 0700)
	b.initialize(templates.Dockerfile, "docker-build/Dockerfile", 0755)
}

func (b *Bootstrap) AddTestcase() {
	// TODO
	os.MkdirAll(b.DestinationDir+"/"+"test", 0700)
	b.initialize(templates.Testcase, "test/Testcase", 0755)
}

func (b *Bootstrap) AddScript() {
	// TODO
	os.MkdirAll(b.DestinationDir+"/"+"scripts", 0700)
	b.initialize(templates.Script, "scripts/Script", 0755)
}

func (b *Bootstrap) AddChangeInfo() {
	b.initialize(templates.README, "README", 0600)
}

func (b *Bootstrap) initialize(t string, file string, perm os.FileMode) {
	tpl := template.Must(template.New("").Parse(t))
	if _, err := os.Stat(b.DestinationDir + "/" + file); err == nil {
		logrus.Errorf("File already exists: %s, skipping", file)
		return
	}
	f, err := os.Create(b.DestinationDir + "/" + file)
	if err != nil {
		logrus.Errorf("Unable to create %s file, skipping: %v", file, err)
		return
	}
	if err := os.Chmod(b.DestinationDir+"/"+file, perm); err != nil {
		logrus.Errorf("Unable to chmod %s file to %v, skipping: %v", file, perm, err)
		return
	}
	defer f.Close()
	if err := tpl.Execute(f, b); err != nil {
		logrus.Errorf("Error processing %s template: %v", file, err)
	}
}
