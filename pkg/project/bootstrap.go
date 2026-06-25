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
	"text/template"

	"gitee.com/openeuler/ktib/pkg/templates"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var yumConfig = "/etc/yum.conf"
var yumCommand = "yum"

// Bootstrap defines the project bootstrap structure
type Bootstrap struct {
	DestinationDir string // Destination directory
	ImageName      string // Image name
	BuildType      string // Build type
}

// Config defines the configuration file structure
type Config struct {
	Packages struct {
		InstallPkgs []string `yaml:"install_pkgs"`
	} `yaml:"packages"`
	Network struct {
		NETWORKING string `yaml:"networking"`
		HOSTNAME   string `yaml:"hostname"`
	} `yaml:"network"`
	Locale   string `yaml:"locale"`
	Timezone string `yaml:"timezone"` // Timezone configuration
}

// NewBootstrap creates a new Bootstrap instance
func NewBootstrap(dir string) *Bootstrap {
	return &Bootstrap{DestinationDir: dir, BuildType: "platform"}
}

// InitProjectStructure initializes the project directory structure
func (b *Bootstrap) InitProjectStructure() error {
	// Create directory structure
	dirs := []string{
		filepath.Join(b.DestinationDir, "dockerfile"), // Directory for storing the Dockerfile
		filepath.Join(b.DestinationDir, "rootfs"),     // Directory for initializing the rootfs
		filepath.Join(b.DestinationDir, "files"),      // Directory for storing files needed to create rootfs
		filepath.Join(b.DestinationDir, "tests"),      // Directory for storing test scripts
	}

	for _, dir := range dirs {
		if info, err := os.Stat(dir); err == nil {
			if !info.IsDir() {
				bak := dir + ".bak"
				if err := os.Rename(dir, bak); err != nil {
					return fmt.Errorf("file with the same name exists %s, failed to rename: %v", dir, err)
				}
			} else {
				// Directory already exists, continue
				continue
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check directory %s: %v", dir, err)
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Add necessary files
	if err := b.AddDockerfile(); err != nil {
		return err
	}
	b.AddChangeInfo()
	b.AddRemoveMinimalList()
	b.AddUnmaskService()
	return nil
}

// InitWorkDir initializes the working directory
func (b *Bootstrap) InitWorkDir(types, config string) {
	baseDir := filepath.Join(b.DestinationDir, "init")

	if types == "baseimage" {
		os.MkdirAll(filepath.Join(baseDir, "baseimage"), 0700)
	} else {
		os.MkdirAll(filepath.Join(baseDir, "appimage"), 0700)
	}
}

// BuildRootfs builds the rootfs
func (b *Bootstrap) BuildRootfs(configFile string) error {
	target, err := filepath.Abs(filepath.Join(b.DestinationDir, "rootfs"))
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Check dnf and create dev directory
	if err := CheckDnfAndCreateDev(target); err != nil {
		return fmt.Errorf("failed to check dnf and create dev directory: %v", err)
	}

	// Create character and FIFO devices
	devices := DefaultDevices()
	for _, dev := range devices {
		switch dev.Type {
		case "c": // Character device
			if err := CreateCharDevice(target, dev.Name, dev.Type, dev.Major, dev.Minor, dev.Mode); err != nil {
				return fmt.Errorf("failed to create character device %s: %v", dev.Name, err)
			}
		case "fifo": // FIFO device
			if err := CreateFifoDevice(target, dev.Name); err != nil {
				return fmt.Errorf("failed to create FIFO device %s: %v", dev.Name, err)
			}
		default:
			return fmt.Errorf("unknown device type: %s", dev.Type)
		}
	}

	// Check if yum/vars directory exists
	if err := CheckVarsFile(target); err != nil {
		return fmt.Errorf("failed to check yum/vars directory: %v", err)
	}

	// Read configuration file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read configuration file %s: %v", configFile, err)
	}

	// Parse YAML configuration
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML configuration: %v", err)
	}

	// Install packages
	packages := config.Packages.InstallPkgs
	if len(packages) == 0 {
		fmt.Println("Warning: No packages specified for installation")
	} else {
		if err := InstallPackages(yumCommand, yumConfig, target, packages...); err != nil {
			return fmt.Errorf("failed to install packages: %v", err)
		}
	}

	// Configure rootfs
	if err := ConfigureRootfs(target, config); err != nil {
		return fmt.Errorf("failed to configure system: %v", err)
	}

	fmt.Println("rootfs build complete, please run 'ktib project clean-rootfs' to clean unnecessary files and packages")
	return nil
}

func (b *Bootstrap) AddDockerfile() error {
	// Create Dockerfile in the dockerfile directory
	dockerfilePath := filepath.Join(b.DestinationDir, "dockerfile")
	if err := os.MkdirAll(dockerfilePath, 0755); err != nil {
		return fmt.Errorf("failed to create dockerfile directory: %w", err)
	}

	// Select different Dockerfile templates based on the build type
	if b.BuildType == "platform" || b.BuildType == "minimal" || b.BuildType == "micro" {
		b.initialize(templates.BaseImageDockerfile, "dockerfile/Dockerfile", 0755)
	} else if b.BuildType == "init" {
		b.initialize(templates.InitImageDockerfile, "dockerfile/Dockerfile", 0755)
	} else {
		b.initialize(templates.Dockerfile, "dockerfile/Dockerfile", 0755)
	}
	return nil
}

func (b *Bootstrap) AddRemoveMinimalList() {
	b.initialize(templates.RemoveMinimalList, "files/removeminimallist", 0644)
}

func (b *Bootstrap) AddUnmaskService() {
	b.initialize(templates.UnmaskService, "files/unmaskService", 0644)
}

func (b *Bootstrap) AddChangeInfo() {
	// Create README file in the project root directory
	b.initialize(templates.README, "README.md", 0644)
}

func (b *Bootstrap) initialize(t string, file string, perm os.FileMode) {
	tpl := template.Must(template.New("").Parse(t))
	if _, err := os.Stat(b.DestinationDir + "/" + file); err == nil {
		logrus.Errorf("File already exists: %s, skipping", file)
		return
	}
	f, err := os.Create(b.DestinationDir + "/" + file)
	if err != nil {
		logrus.Errorf("Unable to create %s file, skipping: %v", file, err)
		return
	}
	if err := os.Chmod(b.DestinationDir+"/"+file, perm); err != nil {
		logrus.Errorf("Unable to chmod %s file to %v, skipping: %v", file, perm, err)
		return
	}
	defer f.Close()
	if err := tpl.Execute(f, b); err != nil {
		logrus.Errorf("Error processing %s template: %v", file, err)
	}
}

// CleanRootfs method is used to clean unnecessary files and packages in the rootfs
func (b *Bootstrap) CleanRootfs() error {
	target, _ := filepath.Abs(filepath.Join(b.DestinationDir, "rootfs"))

	// Check if the rootfs directory exists
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("rootfs directory does not exist, please run 'ktib project build-rootfs' first")
	}

	// 1. Remove unnecessary packages
	removeMinimalListPath := filepath.Join(b.DestinationDir, "files", "removeminimallist")

	fmt.Printf("Removing unnecessary packages, image type: %s\n", b.BuildType)
	if err := RemoveUnnecessaryPackages(target, b.BuildType, removeMinimalListPath); err != nil {
		fmt.Printf("Warning: Failed to remove unnecessary packages: %v\n", err)
	}
	// 2. Remove unnecessary files
	if err := RemoveUnnecessaryFiles(target); err != nil {
		fmt.Printf("Failed to remove unnecessary files: %v\n", err)
	}

	// 3. Configure pip and remove pycache
	if err := ConfigurePipAndRemovePycache(target, b.BuildType); err != nil {
		fmt.Printf("Warning: Failed to configure pip or remove pycache: %v\n", err)
	}

	// 4. Unmask services
	unmaskServicePath := filepath.Join(b.DestinationDir, "files", "unmaskService")
	fmt.Println("Unmasking services")
	if err := UnmaskServices(target, unmaskServicePath); err != nil {
		fmt.Printf("Warning: Failed to unmask services: %v\n", err)
	}

	// 5. Complete filesystem cleanup
	fmt.Println("Cleaning up filesystem")
	if err := CleanupRootfsPath(target); err != nil {
		fmt.Printf("Warning: Failed to clean up filesystem: %v\n", err)
	}

	fmt.Println("rootfs cleanup complete")
	return nil
}
