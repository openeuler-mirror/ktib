/*
Copyright (c) 2024 KylinSoft Co., Ltd.
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
	"strings"

	"gitee.com/openeuler/ktib/pkg/templates"
)

var execCommand = exec.Command

var unnecessaryFiles = []string{
	// **************locales**********************
	"/usr/share/locale",
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
	"/etc/profile.d/system-info.sh",
	//*****************i18n************************
	"/usr/share/i18n",
	//***************yum cache*********************
	"/var/cache/yum",
	//***************sln***************************
	"/sbin/sln",
	//*****************ldconfig********************
	"/var/cache/ldconfig",
	//**********other unnecessary files************8
	"/var/lib/dnf",
	"/run/nologin",
	"/var/log",
}

func ConfigureRootfs(target string, config Config) error {
	// Configure network
	network := config.Network.NETWORKING
	hostname := config.Network.HOSTNAME
	networkConfig := fmt.Sprintf("NETWORKING=%s\nHOSTNAME=%s\n", network, hostname)
	networkFilePath := filepath.Join(target, "/etc/sysconfig/network")
	err := os.WriteFile(networkFilePath, []byte(networkConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing network configuration: %v", err)
	}

	// Set DNF infra variable
	infraConfig := "container"
	infraFilePath := filepath.Join(target, "/etc/dnf/vars/infra")
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(infraFilePath), 0755); err != nil {
		return fmt.Errorf("error creating directory for infra configuration: %v", err)
	}
	err = os.WriteFile(infraFilePath, []byte(infraConfig), 0644)
	if err != nil {
		return fmt.Errorf("error writing infra configuration: %v", err)
	}

	// Configure locale environment
	if config.Locale != "" {
		localeFilePath := filepath.Join(target, "/etc/rpm/macros.image-language-conf")
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(localeFilePath), 0755); err != nil {
			return fmt.Errorf("error creating directory for locale configuration: %v", err)
		}
		err = os.WriteFile(localeFilePath, []byte(config.Locale), 0644)
		if err != nil {
			return fmt.Errorf("error writing language configuration: %v", err)
		}

		// Set system locale environment
		localePath := filepath.Join(target, "/etc/locale.conf")
		// Extract locale code from config.Locale
		// Assuming format is "%_install_langs en_US.UTF-8"
		localeParts := strings.Split(config.Locale, " ")
		localeValue := ""
		if len(localeParts) > 1 {
			localeValue = fmt.Sprintf("LANG=%s\n", localeParts[len(localeParts)-1])
		} else {
			localeValue = "LANG=en_US.UTF-8\n" // Default value
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(localePath), 0755); err != nil {
			return fmt.Errorf("error creating directory for locale.conf: %v", err)
		}
		if err := os.WriteFile(localePath, []byte(localeValue), 0644); err != nil {
			return fmt.Errorf("error writing locale.conf file: %v", err)
		}
	}

	// Configure timezone
	if config.Timezone != "" {
		// Create /etc/localtime symlink pointing to the correct timezone file
		timezonePath := filepath.Join("/usr/share/zoneinfo", config.Timezone)
		localtimePath := filepath.Join(target, "/etc/localtime")

		// Ensure target directory exists
		if err := os.MkdirAll(filepath.Dir(localtimePath), 0755); err != nil {
			return fmt.Errorf("error creating directory for localtime: %v", err)
		}

		// Create symlink
		cmd := execCommand("/usr/bin/ln", "-sf", timezonePath, localtimePath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error setting timezone: %v", err) // Add return
		}

		// Write timezone information to /etc/timezone
		timezoneFPath := filepath.Join(target, "/etc/timezone")
		if err := os.WriteFile(timezoneFPath, []byte(config.Timezone), 0644); err != nil {
			return fmt.Errorf("error writing timezone file: %v", err)
		}
	}

	// force each container to have a unique machine-id
	machineId := ""
	machineIDFilePath := filepath.Join(target, "/etc/machine-id")
	err = os.WriteFile(machineIDFilePath, []byte(machineId), 0644)
	if err != nil {
		return fmt.Errorf("error writing machine-id file: %v", err)
	}

	// Copy bash configuration file and set bash history
	if err := addCommandToScriptAndRun(target, config); err != nil {
		return fmt.Errorf("error add command to script and run: %v", err)
	}
	return nil
}

func addCommandToScriptAndRun(target string, config Config) error {
	// Copy bash configuration files
	bashCmd := execCommand("/usr/bin/sh", "-c", fmt.Sprintf("cp /etc/skel/.bash* %s/root/", target))
	if err := bashCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy bash configuration files: %v", err)
	}

	// Create empty bash history file
	historyPath := filepath.Join(target, "root", ".bash_history")
	if err := os.WriteFile(historyPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to create bash history file: %v", err)
	}

	return nil
}

func parseLocaleConfig(localeConfig string) string {
	parts := strings.Split(localeConfig, " ")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return localeConfig
}

func localeToLibDirName(locale string) string {
	return strings.Replace(locale, ".UTF-8", ".utf8", 1)
}

func containsString(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

func filterLocaleDir(target, dirPath string, keepNames []string) error {
	fullPath := filepath.Join(target, dirPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !containsString(keepNames, entry.Name()) {
			removePath := filepath.Join(fullPath, entry.Name())
			fmt.Println(removePath)
			if err := os.RemoveAll(removePath); err != nil {
				return err
			}
		}
	}
	return nil
}

func RemoveUnnecessaryFiles(target string, localeConfig string) error {
	if err := removeAllFiles(target, unnecessaryFiles); err != nil {
		return err
	}

	localeName := parseLocaleConfig(localeConfig)
	if localeName != "" {
		keepNames := []string{"C.utf8", localeToLibDirName(localeName)}
		if err := filterLocaleDir(target, "/usr/lib/locale", keepNames); err != nil {
			return err
		}
	} else {
		localePath := filepath.Join(target, "/usr/lib/locale")
		fmt.Println(localePath)
		if err := os.RemoveAll(localePath); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Join(target, "var/cache/yum"), 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(target, "var/cache/ldconfig"), 0755); err != nil {
		return err
	}
	return nil
}

func removeAllFiles(target string, files []string) error {
	for _, file := range files {
		fmt.Println(filepath.Join(target, file))
		if err := os.RemoveAll(filepath.Join(target, file)); err != nil {
			return err
		}
	}
	return nil
}

// Add the following function to complete file cleanup
func CleanupRootfsPath(target string) error {
	// 1. Clean up RPM database history
	rpmHistoryFiles, err := filepath.Glob(filepath.Join(target, "var/lib/dnf/history.*"))
	if err == nil && len(rpmHistoryFiles) > 0 {
		fmt.Println("Cleaning up RPM database history...")
		for _, file := range rpmHistoryFiles {
			if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove RPM history file %s: %w", file, err)
			}
		}
	}

	// 2. Clean up temporary files and log files
	fmt.Println("Cleaning up temporary files and log files...")

	logDir := filepath.Join(target, "var/log")
	if _, err := os.Stat(logDir); err == nil {
		fmt.Printf("Emptying directory: %s\n", logDir)
		if err := os.RemoveAll(logDir); err != nil {
			return fmt.Errorf("failed to remove log directory: %w", err)
		}
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to recreate log directory: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat log directory: %w", err)
	}

	tmpDir := filepath.Join(target, "tmp")
	if _, err := os.Stat(tmpDir); err == nil {
		fmt.Printf("Emptying directory: %s\n", tmpDir)
		if err := os.RemoveAll(tmpDir); err != nil {
			return fmt.Errorf("failed to remove tmp directory: %w", err)
		}
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return fmt.Errorf("failed to recreate tmp directory: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat tmp directory: %w", err)
	}

	// 3. Delete nologin file
	nologinFile := filepath.Join(target, "run/nologin")
	if _, err := os.Stat(nologinFile); err == nil {
		fmt.Printf("Deleting file: %s\n", nologinFile)
		if err := os.Remove(nologinFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove nologin file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat nologin file: %w", err)
	}

	// 4. Clean up bash history
	bashHistoryPath := filepath.Join(target, "root/.bash_history")
	if _, err := os.Stat(bashHistoryPath); err == nil {
		fmt.Printf("Emptying file: %s\n", bashHistoryPath)
		if err := os.WriteFile(bashHistoryPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to clear bash history: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat bash history file: %w", err)
	}

	return nil
}

// Add the following function to remove unnecessary packages
// Modify function to accept file path parameter
func RemoveUnnecessaryPackages(target string, imageType string, removeMinimalListPath string) error {
	var packagesToRemove []string
	var err error
	var data []byte

	// Check for root privilege
	if os.Geteuid() != 0 {
		return fmt.Errorf("root privileges are required to execute chroot command")
	}

	// Select the list of packages to remove based on the image type
	if imageType == "minimal" {
		// Read removeminimallist file
		data, err = os.ReadFile(removeMinimalListPath)
		if err != nil {
			return fmt.Errorf("unable to read removeminimallist file: %v", err)
		}
	} else {
		// micro type does not require package removal
		return nil
	}

	packagesToRemove = strings.Split(string(data), "\n")

	// Check if there are packages to remove
	hasPackagesToRemove := false
	for _, pkg := range packagesToRemove {
		pkg = strings.TrimSpace(pkg)
		if pkg != "" && !strings.HasPrefix(pkg, "#") {
			hasPackagesToRemove = true
			break
		}
	}

	if !hasPackagesToRemove {
		fmt.Println("No packages need to be removed")
		return nil
	}

	// Create the package removal script
	scriptContent := "#!/bin/bash\n"
	scriptContent += "set -e\n" // Exit immediately if a command exits with a non-zero status
	scriptContent += "echo 'Starting to remove unnecessary packages...'\n"

	for _, pkg := range packagesToRemove {
		pkg = strings.TrimSpace(pkg)
		if pkg != "" && !strings.HasPrefix(pkg, "#") {
			// First check if the package is installed
			scriptContent += fmt.Sprintf("if rpm -q %s &>/dev/null; then\n", pkg)
			scriptContent += fmt.Sprintf("  echo 'Removing package: %s'\n", pkg)
			scriptContent += fmt.Sprintf("  rpm -e --nodeps %s || echo 'Warning: failed to remove %s'\n", pkg, pkg)
			scriptContent += "fi\n"
		}
	}

	scriptContent += "echo 'Package removal complete'\n"

	// Use absolute path
	scriptPath := filepath.Join(target, "remove_packages.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("unable to create package removal script: %v", err)
	}

	fmt.Println("Executing package removal script...")

	// Execute the script
	cmd := execCommand("chroot", target, "/remove_packages.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	// Clean up script
	if removeErr := os.Remove(scriptPath); removeErr != nil && !os.IsNotExist(removeErr) {
		if err != nil {
			return fmt.Errorf("failed to execute package removal script: %v; also failed to remove script: %w", err, removeErr)
		}
		return fmt.Errorf("failed to remove package removal script: %w", removeErr)
	}

	if err != nil {
		return fmt.Errorf("failed to execute package removal script: %v", err)
	}

	return nil
}

// Modify function to accept file path parameter
func UnmaskServices(target string, unmaskServicePath string) error {
	// Check for root privilege
	if os.Geteuid() != 0 {
		return fmt.Errorf("root privileges are required to execute chroot command")
	}

	// Read unmaskService file
	data, err := os.ReadFile(unmaskServicePath)
	if err != nil {
		return fmt.Errorf("unable to read unmaskService file: %v", err)
	}

	// Check if file content is empty
	if len(strings.TrimSpace(string(data))) == 0 {
		fmt.Println("unmaskService file is empty, skipping service unmasking")
		return nil
	}

	// Create the script for unmasking services
	scriptPath := filepath.Join(target, "unmask_services.sh")

	// Add script header and error handling
	scriptContent := "#!/bin/bash\n"
	scriptContent += "set -e\n" // Exit immediately if a command exits with a non-zero status
	scriptContent += "echo 'Starting to unmask services...'\n"
	scriptContent += string(data)
	scriptContent += "\necho 'Service unmasking complete'\n"

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("unable to create service unmasking script: %v", err)
	}

	fmt.Println("Executing service unmasking script...")

	// Execute the script
	cmd := execCommand("chroot", target, "/unmask_services.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	// Clean up script
	if removeErr := os.Remove(scriptPath); removeErr != nil && !os.IsNotExist(removeErr) {
		if err != nil {
			return fmt.Errorf("failed to execute service unmasking script: %v; also failed to remove script: %w", err, removeErr)
		}
		return fmt.Errorf("failed to remove service unmasking script: %w", removeErr)
	}

	if err != nil {
		return fmt.Errorf("failed to execute service unmasking script: %v", err)
	}

	return nil
}

func ConfigurePipAndRemovePycache(target string, imageType string) error {
	if imageType == "micro" || imageType == "minimal" {
		return nil
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("root privileges are required to execute chroot command")
	}
	scriptPath := filepath.Join(target, "configure_python.sh")
	scriptContent := templates.PythonConfigureScript
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("unable to create Python configuration script: %v", err)
	}
	cmd := execCommand("/usr/sbin/chroot", target, "/configure_python.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if removeErr := os.Remove(scriptPath); removeErr != nil && !os.IsNotExist(removeErr) {
		if err != nil {
			return fmt.Errorf("failed to execute Python configuration script: %v; also failed to remove script: %w", err, removeErr)
		}
		return fmt.Errorf("failed to remove Python configuration script: %w", removeErr)
	}

	if err != nil {
		return fmt.Errorf("failed to execute Python configuration script: %v", err)
	}
	return nil
}
