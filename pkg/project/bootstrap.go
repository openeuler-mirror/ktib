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
	"text/template"

	"gitee.com/openeuler/ktib/pkg/templates"
	"github.com/sirupsen/logrus"
)

type Bootstrap struct {
	DestinationDir string
	ImageName      string
}

func NewBootstrap(dir, imageName string) *Bootstrap {
	return &Bootstrap{DestinationDir: dir, ImageName: imageName}
}

func (b *Bootstrap) InitWorkDir(types string) {
	switch types {
	case "source":
		os.MkdirAll(b.DestinationDir+"/"+"init"+"/"+"source", 0700)
	case "rpm":
		os.MkdirAll(b.DestinationDir+"/"+"init"+"/"+"rpm", 0700)
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
