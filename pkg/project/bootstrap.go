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
	"gitee.com/openeuler/ktib/cmd/ktib/app"
	"gitee.com/openeuler/ktib/pkg/templates"
	"github.com/sirupsen/logrus"
	"os"
	"text/template"
)

type Bootstrap struct {
	DestinationDir string
	ImageName      string
}

func NewBootstrap(dir, imageName string) *Bootstrap {
	return &Bootstrap{DestinationDir: dir, ImageName: imageName}
}

func (b *Bootstrap) AddDockerfile() {
	b.initialize(templates.Dockerfile, "Dockerfile", 0600)
}

func (b *Bootstrap) AddTestcase() {
	// TODO
}

func (b *Bootstrap) AddScript() {
	// TODO
}

func (b *Bootstrap) AddChangeInfo() {
	// TODO
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

func (b *Bootstrap) InitWorkDir(initOption app.InitOption) {
	switch initOption.BuildType {
	case "source":
		os.MkdirAll(b.DestinationDir+"/"+"init"+"/"+"source", 0700)
	case "rpm":
		os.MkdirAll(b.DestinationDir+"/"+"init"+"/"+"rpm", 0700)
	}
}
