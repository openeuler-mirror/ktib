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
	"path/filepath"
	"syscall"
)

// Device represents a device with its attributes
type Device struct {
	Name  string
	Type  string
	Major uint32
	Minor uint32
	Mode  os.FileMode
}

// DefaultDevices creates default devices
func DefaultDevices() map[string]Device {
	defaultDevices := map[string]Device{
		"console": {"console", "c", 5, 1, 0600},
		"initctl": {"initctl", "fifo", 0, 0, 0666},
		"full":    {"full", "c", 1, 7, 0666},
		"null":    {"null", "c", 1, 3, 0666},
		"ptmx":    {"ptmx", "c", 5, 2, 0666},
		"random":  {"random", "c", 1, 8, 0666},
		"tty":     {"tty", "c", 5, 0, 0666},
		"tty0":    {"tty0", "c", 4, 0, 0666},
		"urandom": {"urandom", "c", 1, 9, 0666},
		"zero":    {"zero", "c", 1, 5, 0666},
	}
	return defaultDevices
}

func CreateCharDevice(target, name, nodeType string, major, minor uint32, mode os.FileMode) error {
	path := fmt.Sprintf("%s/dev/%s", target, name)
	err := mknod(path, nodeType, major, minor)
	if err != nil {
		return fmt.Errorf("failed to create device %s: %v", name, err)
	}
	err = os.Chmod(path, mode)
	if err != nil {
		return fmt.Errorf("failed to set mode for device %s: %v", name, err)
	}
	return nil
}

func CreateFifoDevice(target, name string) error {
	path := fmt.Sprintf("%s/dev/%s", target, name)
	if err := os.MkdirAll(fmt.Sprintf("%s/dev", target), 0755); err != nil {
		return fmt.Errorf("failed to create dev directory: %v", err)
	}
	err := syscall.Mkfifo(path, 0600)
	if err != nil {
		return fmt.Errorf("failed to create fifo file initctl: %v", err)
	}
	return nil
}

func mknod(path, nodeType string, major, minor uint32) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	cmd := execCommand("/usr/bin/mknod", "-m", "666", path, nodeType, fmt.Sprint(major), fmt.Sprint(minor))
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
