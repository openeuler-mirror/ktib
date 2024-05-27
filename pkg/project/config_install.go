package project

import (
	"fmt"
	"os"
	"os/exec"
)

var imageType = "default"

var imageTypePackages = map[string][]string{
	"micro":    {"bash", "coreutils-single"},
	"minimal":  {"microdnf", "vim-minimal", "iproute"},
	"init":     {""},
	"platform": {""},
	"default":  {"yum", "iproute", "vim-minimal", "procps-ng", "passwd"},
}

func InstallPackages(tag, yumConfig, target string, packages ...string) error {
	cmd := exec.Command("yum", "-c", yumConfig, "--installroot="+target, "--releasever=/", "--setopt=tsflags=nodocs",
		"--setopt=group_package_types=mandatory", "-y", "install")
	cmd.Args = append(cmd.Args, packages...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Error executing command: %v\n", err)
	}

	cleanCmd := exec.Command("yum", "-c", yumConfig, "--installroot="+target, "-y", "clean", "all")
	cleanCmd.Stdout = os.Stdout
	cleanCmd.Stderr = os.Stderr
	if err := cleanCmd.Run(); err != nil {
		return fmt.Errorf("Failed to clean all Packages: %v\n", err)
	}
	return nil
}
