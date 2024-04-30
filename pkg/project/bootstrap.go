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
	"io/ioutil"
	"os"
	"os/exec"
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
	target := filepath.Join(getCurrentWorkingDirectory(), b.DestinationDir+"/"+"init"+"/"+"baseimage")

	//Check if dnf is available and create new dev directory
	if err := checkDnfAndCreateDev(target); err != nil {
		fmt.Printf("Failed to checkDnfAndCreateDev: %v", err)
		return
	}

	//create char device and fifo device
	createCharDevice(target, "console", "c", 5, 1, 0600)
	createFifoDevice(target, "initctl")
	createCharDevice(target, "full", "c", 1, 7, 0666)
	createCharDevice(target, "null", "c", 1, 3, 0666)
	createCharDevice(target, "ptmx", "c", 5, 2, 0666)
	createCharDevice(target, "random", "c", 1, 8, 0666)
	createCharDevice(target, "tty", "c", 5, 0, 0666)
	createCharDevice(target, "tty0", "c", 4, 0, 0666)
	createCharDevice(target, "urandom", "c", 1, 9, 0666)
	createCharDevice(target, "zero", "c", 1, 5, 0666)

	//Check if the/etc/yum/vars directory exists
	if err := checkVarsFile(target); err != nil {
		fmt.Printf("Failed to checkVarsFile: %v", err)
		return
	}

	//Install different installation packages base to tag
	if strings.Contains(imageName, "micro") {
		installPackages("micro", yumConfig, target, "bash", "coreutils-single")
	} else if strings.Contains(imageName, "minimal") {
		installPackages("minimal", yumConfig, target, "microdnf", "vim-minimal", "iproute")
	} else {
		installPackages("default", yumConfig, target, "yum", "iproute", "vim-minimal", "procps-ng", "passwd")
	}

	//Configure network settings、dnf variable、en_US.UTF-8 locale files and machine-id
	if err := configureSystemParam(target); err != nil {
		fmt.Printf("Error configuring system:%v", err)
		return
	}

	//Delete unnecessary configurations is to reduce the volume of the base image
	if err := removeUnnecessaryFiles(target); err != nil {
		fmt.Printf("Error remove unnecessary file :%v", err)
		return
	}

	// cp bash && local settings and time zone to chroot_script.sh and run the script.sh
	if err := addCommandToScriptAndRun(target); err != nil {
		fmt.Printf("Error add command to script and run: %v", err)
		return
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

func getCurrentWorkingDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to obtain the %s path :%v", dir, err)
	}
	return dir
}

func checkDnfAndCreateDev(target string) error {
	if _, err := os.Stat("/etc/dnf/dnf.conf"); err == nil {
		yumConfig = "/etc/dnf/dnf.conf"
		setAlias("yum", "dnf")
	}
	// Remove target directory if exists and create new
	err := os.RemoveAll(target)
	if err != nil {
		fmt.Printf("Failed to remove target directory:%v", err)
	}
	if err := os.MkdirAll(target+"/dev", 0755); err != nil {
		fmt.Printf("Failed to create dev directory:%v", err)
	}
	return nil
}

func setAlias(alias, command string) error {
	cmd := exec.Command("alias", alias+"="+command)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func createCharDevice(target, name, nodeType string, major, minor uint32, mode os.FileMode) error {

	return nil
}

func mknod(path, nodeType string, major, minor uint32) error {
	cmd := exec.Command("mknod", "-m", "666", path, nodeType, fmt.Sprint(major), fmt.Sprint(minor))
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func createFifoDevice(target, name string) error {
	return nil
}

func checkVarsFile(target string) error {
	if _, err := os.Stat("/etc/yum/vars"); err == nil {
		err := os.MkdirAll(filepath.Join(target, "/etc/yum"), 0755)
		if err != nil {
			fmt.Printf("Failed to create yum directory: %v", err)
		}
		cmd := exec.Command("cp", "-a", "/etc/yum/vars", filepath.Join(target, "etc/yum/"))
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to copy /etc/yum/vars:%v", err)
		}
	} else if os.IsNotExist(err) {
		fmt.Printf("/etc/yum/vars directory does not exist :%v", err)
	}
	return nil
}

func installPackages(tag, yumConfig, target string, packages ...string) {
	cmd := exec.Command("yum", "-c", yumConfig, "--installroot="+target, "--releasever=/", "--setopt=tsflags=nodocs",
		"--setopt=group_package_types=mandatory", "-y", "install")
	cmd.Args = append(cmd.Args, packages...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		return
	}

	cleanCmd := exec.Command("yum", "-c", yumConfig, "--installroot="+target, "-y", "clean", "all")
	cleanCmd.Stdout = os.Stdout
	cleanCmd.Stderr = os.Stderr
	if err := cleanCmd.Run(); err != nil {
		fmt.Printf("Failed to clean all packages: %v\n", err)
		return
	}
}

func configureSystemParam(target string) error {
	//network configuration
	const networkConfig = "NETWORKING=yes\nHOSTNAME=localhost.localdomain\n"
	networkFilePath := filepath.Join(target, "/etc/sysconfig/network")
	err := ioutil.WriteFile(networkFilePath, []byte(networkConfig), 0644)
	if err != nil {
		fmt.Printf("error writing network configuration: %v", err)
	}

	// set DNF infra variable to container for compatibility with KylinOS
	infraConfig := "container"
	infraFilePath := "/etc/dnf/vars/infra"
	err = ioutil.WriteFile(infraFilePath, []byte(infraConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing infra configuration: %v", err)
	}

	// install only en_US.UTF-8 locale files, see
	// https://fedoraproject.org/wiki/Changes/Glibc_locale_subpackaging for details
	localeConfig := "%_install_langs en_US.UTF-8\n"
	localeFilePath := "/etc/rpm/macros.image-language-conf"
	err = ioutil.WriteFile(localeFilePath, []byte(localeConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing language configuration: %v", err)
	}

	// force each container to have a unique machine-id
	machineIDFilePath := "/etc/machine-id"
	err = ioutil.WriteFile(machineIDFilePath, []byte(""), 0644)
	if err != nil {
		return fmt.Errorf("error writing machine-id file: %v", err)
	}
	return nil
}

func removeUnnecessaryFiles(target string) error {
	unnecessaryFileToRemove := []string{
		//locales
		filepath.Join(target, "/usr/lib/local"),
		filepath.Join(target, "/usr/share/local"),
		filepath.Join(target, "/lib/gconv"),
		filepath.Join(target, "/lib64/gconv"),
		filepath.Join(target, "/bin/localedef"),
		filepath.Join(target, "/sbin/build-locale-archive"),
		//docs and man pages
		filepath.Join(target, "/usr/share/man"),
		filepath.Join(target, "/usr/share/doc"),
		filepath.Join(target, "/usr/share/info"),
		filepath.Join(target, "/usr/share/gnome/help"),
		//profile.d
		filepath.Join(target, "/etc/profile.d/lang.sh"),
		filepath.Join(target, "/etc/profile.d/system-info.sh"),
		//i18n
		filepath.Join(target, "/usr/share/i18n"),
		//  yum cache
		filepath.Join(target, "/var/cache/yum"),
		//  sln
		filepath.Join(target, "/sbin/sln"),
		//ldconfig
		filepath.Join(target, "/etc/ld.so.cache"),
		filepath.Join(target, "/var/cache/ldconfig"),
		//other unncessary files
		filepath.Join(target, "/var/lib/dnf"),
		filepath.Join(target, "/run/nologin"), //查找不到该文件
		filepath.Join(target, "/var/log"),
		filepath.Join(target, "/tmp"),
	}
	if err := removeAll(unnecessaryFileToRemove); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(target, "var/cache/yum"), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(target, "/var/cache/ldconfig"), 0755); err != nil {
		return err
	}
	return nil
}

func removeAll(paths []string) error {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

func addCommandToScriptAndRun(target string) error {
	cmd := exec.Command("sh", "-c", "cp /etc/skel/.bash* "+target+"/root/")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error copying bash files: %v", err)
	}

	// Create an empty .bash_history file
	cmd = exec.Command("sh", "-c", "echo \"\" > "+target+"/root/.bash_history")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating .bash_history file: %v", err)
	}

	cmd = exec.Command("sh", "-c", "echo \"\" > "+target+"/chroot_script.sh")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating chroot_script.sh file: %v", err)
	}

	// add executable permissions to chroot_script.sh
	cmd = exec.Command("chmod", "+x", target+"/chroot_script.sh")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error adding executable permission to chroot_script.sh: %v", err)
	}

	//edit time zone
	cmd = exec.Command("sh", "-c", "echo \"ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime\" > "+target+"/chroot_script.sh")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error edit time zone to chroot_script.sh: %v", err)
	}

	//run chroot_script
	cmd = exec.Command("chroot", target, "/chroot_script.sh")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error chroot chroot_script.sh file: %v", err)
	}

	//rm rf chroot_script
	cmd = exec.Command("rm", "-rf", target+"/chroot_script.sh")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error rm rf chroot_script.sh: %v", err)
	}

	return nil
}
