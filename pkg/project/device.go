package project

import (
	"fmt"
	"os"
	"os/exec"
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
		fmt.Printf("Failed to create device %s: %s\n", name, err)
	}
	err = os.Chmod(path, mode)
	if err != nil {
		fmt.Printf("Failed to set mode for device %s: %s\n", name, err)
	}
	return nil
}

func CreateFifoDevice(target, name string) error {
	path := fmt.Sprintf("%s/dev/%s", target, name)
	err := syscall.Mkfifo(path, 0600)
	if err != nil {
		fmt.Printf("Failed to create fifo file initctl: %v", err)
	}
	return nil
}

func mknod(path, nodeType string, major, minor uint32) error {
	cmd := exec.Command("mknod", "-m", "666", path, nodeType, fmt.Sprint(major), fmt.Sprint(minor))
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
