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

func CheckVarsFile(target string) error {
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
