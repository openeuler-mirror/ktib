package project

import (
	"fmt"
	"io/ioutil"
	"os"
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

func ConfigureRootfs(target string) error {
	//rootfs network configuration
	const networkConfig = "NETWORKING=yes\nHOSTNAME=localhost.localdomain\n"
	networkFilePath := filepath.Join(target, "/etc/sysconfig/network")
	err := ioutil.WriteFile(networkFilePath, []byte(networkConfig), 0644)
	if err != nil {
		fmt.Printf("error writing network configuration: %v", err)
	}

	// set DNF infra variable to container for compatibility with KylinOS
	infraConfig := "container"
	infraFilePath := filepath.Join(target, "/etc/dnf/vars/infra")
	err = ioutil.WriteFile(infraFilePath, []byte(infraConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing infra configuration: %v", err)
	}

	// install only en_US.UTF-8 locale files, see
	// https://fedoraproject.org/wiki/Changes/Glibc_locale_subpackaging for details
	localeConfig := "%_install_langs en_US.UTF-8\n"
	localeFilePath := filepath.Join(target, "/etc/rpm/macros.image-language-conf")
	err = ioutil.WriteFile(localeFilePath, []byte(localeConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing language configuration: %v", err)
	}

	// force each container to have a unique machine-id
	machineIDFilePath := filepath.Join(target, "/etc/machine-id")
	err = ioutil.WriteFile(machineIDFilePath, []byte(""), 0644)
	if err != nil {
		return fmt.Errorf("error writing machine-id file: %v", err)
	}
	return nil
}

func RemoveUnnecessaryFiles(target string) error {
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
