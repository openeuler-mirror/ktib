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
	"os"
	"path/filepath"
	"testing"
)

func requireMknod(t *testing.T, target string) {
	t.Helper()
	path := filepath.Join(target, "dev", "ktib_mknod_probe")
	if err := mknod(path, "c", 1, 3); err != nil {
		t.Skipf("mknod unavailable: %v", err)
	}
	_ = os.Remove(path)
}

func TestCreateCharDevice(t *testing.T) {
	target := t.TempDir()
	requireMknod(t, target)

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
	target := t.TempDir()
	requireMknod(t, target)

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
