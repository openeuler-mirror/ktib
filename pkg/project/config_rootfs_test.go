//go:build linux

/*
   Copyright (c) 2025 KylinSoft Co., Ltd.
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
	"testing"
)

// TestHelperProcess mimics the behavior of external commands.
// It is called by the mocked execCommand.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Locate the command and arguments after "--"
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command provided\n")
		os.Exit(2)
	}

	cmd := args[0]
	// cmdArgs := args[1:] // Unused for now

	switch cmd {
	case "/usr/bin/ln":
		// usage: ln -sf target linkname
		os.Exit(0)
	case "/usr/bin/sh":
		// usage: sh -c "cp ..."
		os.Exit(0)
	case "chroot", "/usr/sbin/chroot":
		// usage: chroot target /script.sh
		// Simulate success for scripts running in chroot
		os.Exit(0)
	default:
		// Default to success for other commands
		os.Exit(0)
	}
}

// mockExecCommand replaces exec.Command with a call to TestHelperProcess
func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestConfigureRootfs(t *testing.T) {
	// Mock execCommand
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = mockExecCommand

	tempDir, err := os.MkdirTemp("", "rootfs_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Locale:   "en_US.UTF-8",
		Timezone: "Asia/Shanghai",
	}
	config.Network.NETWORKING = "yes"
	config.Network.HOSTNAME = "localhost"

	// Create necessary directories that are expected to exist (usually created by yum install)
	requiredDirs := []string{
		"/etc/sysconfig",
		"/root",
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create required dir %s: %v", dir, err)
		}
	}

	err = ConfigureRootfs(tempDir, config)
	if err != nil {
		t.Fatalf("ConfigureRootfs failed: %v", err)
	}

	// Verify file creation
	expectedFiles := []string{
		"/etc/sysconfig/network",
		"/etc/dnf/vars/infra",
		"/etc/rpm/macros.image-language-conf",
		"/etc/locale.conf",
		"/etc/timezone",
		"/etc/machine-id",
		"/root/.bash_history",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tempDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		}
	}

	// Verify content of network config
	networkPath := filepath.Join(tempDir, "/etc/sysconfig/network")
	content, err := os.ReadFile(networkPath)
	if err != nil {
		t.Errorf("Failed to read network config: %v", err)
	}
	expectedNetwork := "NETWORKING=yes\nHOSTNAME=localhost\n"
	if string(content) != expectedNetwork {
		t.Errorf("Network config mismatch. Got: %s, Want: %s", string(content), expectedNetwork)
	}
}

func TestRemoveUnnecessaryFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_files_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that should be removed
	// Using one of the files from unnecessaryFiles list, e.g., /usr/share/doc
	docDir := filepath.Join(tempDir, "usr/share/doc")
	if err := os.MkdirAll(docDir, 0755); err != nil {
		t.Fatalf("Failed to create doc dir: %v", err)
	}
	dummyFile := filepath.Join(docDir, "readme.txt")
	if err := os.WriteFile(dummyFile, []byte("dummy"), 0644); err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	err = RemoveUnnecessaryFiles(tempDir)
	if err != nil {
		t.Fatalf("RemoveUnnecessaryFiles failed: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(docDir); !os.IsNotExist(err) {
		t.Errorf("Directory %s should have been removed", docDir)
	}

	// Verify directories are recreated
	recreatedDirs := []string{
		"var/cache/yum",
		"var/cache/ldconfig",
	}
	for _, dir := range recreatedDirs {
		path := filepath.Join(tempDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s should have been recreated", dir)
		}
	}
}

func TestCleanupRootfsPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_path_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup files to be cleaned
	// 1. RPM history
	historyFile := filepath.Join(tempDir, "var/lib/dnf/history.sqlite")
	if err := os.MkdirAll(filepath.Dir(historyFile), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(historyFile, []byte("history data"), 0644)

	// 2. Log files
	logFile := filepath.Join(tempDir, "var/log/messages")
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(logFile, []byte("log data"), 0644)

	// 3. nologin
	nologinFile := filepath.Join(tempDir, "run/nologin")
	if err := os.MkdirAll(filepath.Dir(nologinFile), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(nologinFile, []byte(""), 0644)

	// 4. bash history
	bashHistory := filepath.Join(tempDir, "root/.bash_history")
	if err := os.MkdirAll(filepath.Dir(bashHistory), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(bashHistory, []byte("ls -la"), 0644)

	err = CleanupRootfsPath(tempDir)
	if err != nil {
		t.Fatalf("CleanupRootfsPath failed: %v", err)
	}

	// Verify deletion
	if _, err := os.Stat(historyFile); !os.IsNotExist(err) {
		t.Errorf("RPM history file should be deleted")
	}
	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Errorf("Log file should be deleted")
	}
	// Log dir should exist (recreated)
	if _, err := os.Stat(filepath.Dir(logFile)); os.IsNotExist(err) {
		t.Errorf("Log directory should exist")
	}

	if _, err := os.Stat(nologinFile); !os.IsNotExist(err) {
		t.Errorf("nologin file should be deleted")
	}

	// Verify bash history is empty
	content, err := os.ReadFile(bashHistory)
	if err != nil {
		t.Errorf("Failed to read bash history: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("Bash history should be empty")
	}
}

func TestRemoveUnnecessaryPackages(t *testing.T) {
	// Mock execCommand
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = mockExecCommand

	tempDir, err := os.MkdirTemp("", "remove_pkgs_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock remove list
	listPath := filepath.Join(tempDir, "remove.list")
	err = os.WriteFile(listPath, []byte("pkg1\npkg2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create list file: %v", err)
	}

	err = RemoveUnnecessaryPackages(tempDir, "minimal", listPath)

	// Check result based on permissions
	if os.Geteuid() != 0 {
		if err == nil {
			t.Errorf("Expected error when running as non-root, got nil")
		} else if !strings.Contains(err.Error(), "root privileges are required") {
			t.Errorf("Unexpected error message: %v", err)
		}
	} else {
		if err != nil {
			t.Errorf("RemoveUnnecessaryPackages failed: %v", err)
		}
		// Since we mocked execution, we can't verify the script execution side effects easily
		// unless we inspect what TestHelperProcess received, but here we assume success.
	}
}

func TestUnmaskServices(t *testing.T) {
	// Mock execCommand
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = mockExecCommand

	tempDir, err := os.MkdirTemp("", "unmask_services_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock service file
	servicePath := filepath.Join(tempDir, "unmask_services")
	err = os.WriteFile(servicePath, []byte("systemctl unmask foo"), 0644)
	if err != nil {
		t.Fatalf("Failed to create service file: %v", err)
	}

	err = UnmaskServices(tempDir, servicePath)

	if os.Geteuid() != 0 {
		if err == nil {
			t.Errorf("Expected error when running as non-root, got nil")
		} else if !strings.Contains(err.Error(), "root privileges are required") {
			t.Errorf("Unexpected error message: %v", err)
		}
	} else {
		if err != nil {
			t.Errorf("UnmaskServices failed: %v", err)
		}
	}
}

func TestConfigurePipAndRemovePycache(t *testing.T) {
	// Mock execCommand
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = mockExecCommand

	tempDir, err := os.MkdirTemp("", "pip_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = ConfigurePipAndRemovePycache(tempDir, "platform")

	if os.Geteuid() != 0 {
		if err == nil {
			t.Errorf("Expected error when running as non-root, got nil")
		} else if !strings.Contains(err.Error(), "root privileges are required") {
			t.Errorf("Unexpected error message: %v", err)
		}
	} else {
		if err != nil {
			t.Errorf("ConfigurePipAndRemovePycache failed: %v", err)
		}
	}
}
