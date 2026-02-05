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
	"os/exec"
	"path/filepath"
	"testing"
)

// TestHelperProcessDevice mimics the behavior of external commands for device operations.
// It is called by the mocked execCommand.
func TestHelperProcessDevice(t *testing.T) {
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
	switch cmd {
	case "/usr/bin/mknod":
		// usage: mknod -m 666 path type major minor
		// args: ["/usr/bin/mknod", "-m", "666", path, type, major, minor]
		if len(args) >= 4 {
			path := args[3]
			// Simulate device creation by creating a regular file
			if err := os.WriteFile(path, []byte("mock device"), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create mock device: %v\n", err)
				os.Exit(1)
			}
		}
		os.Exit(0)
	default:
		os.Exit(0)
	}
}

// mockExecCommandDevice replaces exec.Command with a call to TestHelperProcessDevice
func mockExecCommandDevice(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcessDevice", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestCreateCharDevice(t *testing.T) {
	// Mock execCommand
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = mockExecCommandDevice

	target := t.TempDir()
	// Create dev directory since CreateCharDevice expects it or creates files in it
	if err := os.MkdirAll(filepath.Join(target, "dev"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	type args struct {
		target   string
		name     string
		nodeType string
		major    uint32
		minor    uint32
		mode     os.FileMode
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestCreateCharDevice",
			args: args{
				target:   target,
				name:     "random",
				nodeType: "c",
				major:    5,
				minor:    1,
				mode:     os.FileMode(0644),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateCharDevice(tt.args.target, tt.args.name, tt.args.nodeType, tt.args.major, tt.args.minor, tt.args.mode); (err != nil) != tt.wantErr {
				t.Errorf("CreateCharDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
			if _, err := os.Lstat(filepath.Join(tt.args.target, "dev", tt.args.name)); err != nil {
				t.Fatalf("expected device to exist: %v", err)
			}
		})
	}
}

func TestCreateFifoDevice(t *testing.T) {
	target := t.TempDir()
	if err := os.MkdirAll(filepath.Join(target, "dev"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	type args struct {
		target string
		name   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestCreateFifoDevice",
			args: args{
				target: target,
				name:   "initctl",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateFifoDevice(tt.args.target, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("CreateFifoDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
			if _, err := os.Lstat(filepath.Join(tt.args.target, "dev", tt.args.name)); err != nil {
				t.Fatalf("expected fifo to exist: %v", err)
			}
		})
	}
}

func TestMknod(t *testing.T) {
	// Mock execCommand
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = mockExecCommandDevice

	target := t.TempDir()

	type args struct {
		path     string
		nodeType string
		major    uint32
		minor    uint32
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestMknod",
			args: args{
				path:     filepath.Join(target, "dev", "random"),
				nodeType: "c",
				major:    5,
				minor:    1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mknod(tt.args.path, tt.args.nodeType, tt.args.major, tt.args.minor); (err != nil) != tt.wantErr {
				t.Errorf("mknod() error = %v, wantErr %v", err, tt.wantErr)
			}
			if _, err := os.Lstat(tt.args.path); err != nil {
				t.Fatalf("expected node to exist: %v", err)
			}
		})
	}
}

func TestDefaultDevices(t *testing.T) {
	devices := DefaultDevices()
	if len(devices) == 0 {
		t.Error("DefaultDevices() returned empty map")
	}

	expectedDevices := []string{
		"console", "initctl", "full", "null", "ptmx",
		"random", "tty", "tty0", "urandom", "zero",
	}

	for _, name := range expectedDevices {
		if _, ok := devices[name]; !ok {
			t.Errorf("DefaultDevices() missing device: %s", name)
		}
	}

	// Verify a specific device details (e.g., null)
	nullDev, ok := devices["null"]
	if ok {
		if nullDev.Name != "null" || nullDev.Type != "c" || nullDev.Major != 1 || nullDev.Minor != 3 {
			t.Errorf("DefaultDevices() null device mismatch: got %v", nullDev)
		}
	}
}
