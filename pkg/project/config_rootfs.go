package project

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

var unnecessaryFiles = []string{
	// **************locales**********************
	"/usr/lib/local",
	"/usr/share/local",
	"/lib/gconv",
	"/lib64/gconv",
	"/bin/localedef",
	"/sbin/build-locale-archive",
	//************docs and man pages**************
	"/usr/share/man",
	"/usr/share/doc",
	"/usr/share/info",
	"/usr/share/gnome/help",
	//**************profile.d**********************
	"/etc/profile.d/lang.sh",
	"/etc/profile.d/system-info.sh",
	//*****************i18n************************
	"/usr/share/i18n",
	//***************yum cache*********************
	"/var/cache/yum",
	//***************sln***************************
	"/sbin/sln",
	//*****************ldconfig********************
	"/etc/ld.so.cache",
	"/var/cache/ldconfig",
	//**********other unnecessary files************8
	"/var/lib/dnf",
	"/run/nologin",
	"/var/log",
	"/tmp",
}

func ConfigureRootfs(target string, config Config) error {
	//rootfs network configuration
	networkConfig := config.Network
	networkFilePath := filepath.Join(target, "/etc/sysconfig/network")
	err := ioutil.WriteFile(networkFilePath, []byte(networkConfig), 0644)
	if err != nil {
		fmt.Printf("error writing network configuration: %v", err)
	}

	// set DNF infra variable to container for compatibility with KylinOS
	infraConfig := config.Infra
	infraFilePath := filepath.Join(target, "/etc/dnf/vars/infra")
	err = ioutil.WriteFile(infraFilePath, []byte(infraConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing infra configuration: %v", err)
	}

	// install only en_US.UTF-8 locale files, see
	// https://fedoraproject.org/wiki/Changes/Glibc_locale_subpackaging for details
	localeConfig := config.Locale
	localeFilePath := filepath.Join(target, "/etc/rpm/macros.image-language-conf")
	err = ioutil.WriteFile(localeFilePath, []byte(localeConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing language configuration: %v", err)
	}

	// force each container to have a unique machine-id
	machineId := config.MachineID
	machineIDFilePath := filepath.Join(target, "/etc/machine-id")
	err = ioutil.WriteFile(machineIDFilePath, []byte(machineId), 0644)
	if err != nil {
		return fmt.Errorf("error writing machine-id file: %v", err)
	}

	//Delete unnecessary configurations is to reduce the volume of the base image
	if err := removeUnnecessaryFiles(target); err != nil {
		return fmt.Errorf("Error remove unnecessary file :%v\n", err)
	}

	// cp bash && local settings and time zone to chroot_script.sh and run the script.sh
	if err := addCommandToScriptAndRun(target); err != nil {
		return fmt.Errorf("Error add command to script and run: %v\n", err)
	}
	return nil
}

func removeUnnecessaryFiles(target string) error {
	if err := removeAllFiles(target, unnecessaryFiles); err != nil {
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

func removeAllFiles(target string, files []string) error {
	for _, file := range files {
		if err := os.RemoveAll(filepath.Join(target, file)); err != nil {
			return err
		}
	}
	return nil
}

func addCommandToScriptAndRun(target string) error {
	//cp bash
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
