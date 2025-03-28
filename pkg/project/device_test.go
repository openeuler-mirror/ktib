#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

package project

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateCharDevice(t *testing.T) {
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
				target:   "test_dir",
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
		})
	}
}

func TestCreateFifoDevice(t *testing.T) {
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
				target: "test_dir",
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
		})
	}
}

func TestMknod(t *testing.T) {
	// 创建一个临时目录
	tmpDir, err := os.MkdirTemp("", "mknod_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir) // 测试结束时删除临时目录

	// 定义设备文件的路径
	devicePath := filepath.Join(tmpDir, "test_device")

	// 测试创建字符设备
	err = mknod(devicePath, "c", 1, 2)
	if err != nil {
		t.Skip("Skipping test: mknod might require root privileges or the OS does not support it")
	}

	// 验证设备文件是否创建成功
	_, err = os.Stat(devicePath)
	assert.NoError(t, err, "Expected device file to be created")

	// 清理创建的设备文件
	err = os.Remove(devicePath)
	assert.NoError(t, err, "Expected to remove the device file")
}
