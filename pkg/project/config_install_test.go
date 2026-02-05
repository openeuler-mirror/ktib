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
	"testing"
)

func TestInstallPackages(t *testing.T) {
	if os.Getenv("KTIB_RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set KTIB_RUN_INTEGRATION_TESTS=1 to enable this test")
	}

	type args struct {
		yumConfig string
		target    string
		packages  []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test installation of packages",
			args: args{
				yumConfig: "/etc/yum.conf",
				target:    "/tmp/target",
				packages:  []string{"yum", "bash"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InstallPackages(tt.args.yumConfig, tt.args.target, tt.args.packages...); (err != nil) != tt.wantErr {
				t.Errorf("InstallPackages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestHelperProcessUnit mimics the behavior of external commands.
// It is called by the mocked execCommandUnit.
func TestHelperProcessUnit(t *testing.T) {
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

	// For this test, we assume all commands (yum install/clean) succeed
	os.Exit(0)
}

// mockExecCommandUnit replaces exec.Command with a call to TestHelperProcessUnit
func mockExecCommandUnit(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcessUnit", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestInstallPackagesUnit(t *testing.T) {
	// Mock execCommand
	origExec := execCommand
	defer func() { execCommand = origExec }()
	execCommand = mockExecCommandUnit

	type args struct {
		yumConfig string
		target    string
		packages  []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test successful installation",
			args: args{
				yumConfig: "/etc/yum.conf",
				target:    "/tmp/target",
				packages:  []string{"pkg1", "pkg2"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InstallPackages(tt.args.yumConfig, tt.args.target, tt.args.packages...); (err != nil) != tt.wantErr {
				t.Errorf("InstallPackages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
