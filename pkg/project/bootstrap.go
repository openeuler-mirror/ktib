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
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"gitee.com/openeuler/ktib/pkg/templates"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var yumConfig = "/etc/yum.conf"

// Bootstrap 定义项目引导结构
type Bootstrap struct {
	DestinationDir string // 目标目录
	ImageName      string // 镜像名称
	BuildType      string // 构建类型
}

// Config 定义配置文件结构
type Config struct {
	Packages struct {
		InstallPkgs []string `yaml:"install_pkgs"`
	} `yaml:"packages"`
	Network struct {
		NETWORKING string `yaml:"networking"`
		HOSTNAME   string `yaml:"hostname"`
	} `yaml:"network"`
	Locale   string `yaml:"locale"`
	Timezone string `yaml:"timezone"` // 时区配置
}

// NewBootstrap 创建新的Bootstrap实例
func NewBootstrap(dir string) *Bootstrap {
	return &Bootstrap{DestinationDir: dir, BuildType: "baseimage"}
}

// InitProjectStructure 初始化项目目录结构
func (b *Bootstrap) InitProjectStructure() error {
	// 创建目录结构
	dirs := []string{
		filepath.Join(b.DestinationDir, "dockerfile"), // 存放 Dockerfile 的目录
		filepath.Join(b.DestinationDir, "rootfs"),     // 用于初始化 rootfs 的目录
		filepath.Join(b.DestinationDir, "files"),      // 存放制作rootfs需要的文件
		filepath.Join(b.DestinationDir, "tests"),      // 存放测试脚本的目录
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %v", dir, err)
		}
	}

	// 添加必要的文件
	b.AddDockerfile()
	b.AddTestcase()
	b.AddChangeInfo()
	b.AddRemoveList()
	b.AddRemoveMinimalList()
	b.AddUnmaskService()
	return nil
}

// InitWorkDir 初始化工作目录
func (b *Bootstrap) InitWorkDir(types, config string) {
	baseDir := filepath.Join(b.DestinationDir, "init")

	if types == "baseimage" {
		os.MkdirAll(filepath.Join(baseDir, "baseimage"), 0700)
	} else {
		os.MkdirAll(filepath.Join(baseDir, "appimage"), 0700)
	}
}

// BuildRootfs 构建rootfs
func (b *Bootstrap) BuildRootfs(configFile string) error {
	target, _ := filepath.Abs(filepath.Join(b.DestinationDir, "rootfs"))

	// 检查dnf并创建dev目录
	if err := CheckDnfAndCreateDev(target); err != nil {
		return fmt.Errorf("检查dnf并创建dev目录失败: %v", err)
	}

	// 创建字符设备和FIFO设备
	devices := DefaultDevices()
	for _, dev := range devices {
		if dev.Type == "c" {
			CreateCharDevice(target, dev.Name, dev.Type, dev.Major, dev.Minor, dev.Mode)
		} else if dev.Type == "fifo" {
			CreateFifoDevice(target, dev.Name)
		}
	}

	// 检查yum/vars目录是否存在
	if err := CheckVarsFile(target); err != nil {
		return fmt.Errorf("检查yum/vars目录失败: %v", err)
	}

	// 读取配置文件
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析YAML配置
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析YAML失败: %v", err)
	}

	// 安装软件包
	packages := config.Packages.InstallPkgs
	if err := InstallPackages(yumConfig, target, packages...); err != nil {
		return fmt.Errorf("安装软件包失败: %v", err)
	}

	// 配置rootfs
	if err := ConfigureRootfs(target, config); err != nil {
		return fmt.Errorf("配置系统失败: %v", err)
	}

	fmt.Println("rootfs 构建完成，请运行 'ktib project clean-rootfs' 命令清理不必要的文件和软件包")
	return nil
}

func (b *Bootstrap) AddDockerfile() {
	// 在 dockerfile 目录中创建 Dockerfile
	dockerfilePath := filepath.Join(b.DestinationDir, "dockerfile")
	os.MkdirAll(dockerfilePath, 0755)

	// 根据构建类型选择不同的 Dockerfile 模板
	if b.BuildType == "baseimage" {
		b.initialize(templates.BaseImageDockerfile, "dockerfile/Dockerfile", 0755)
	} else {
		b.initialize(templates.Dockerfile, "dockerfile/Dockerfile", 0755)
	}
}

func (b *Bootstrap) AddTestcase() {
	// 在 tests 目录中创建测试用例
	b.initialize(templates.Testcase, "tests/test.sh", 0755)
}

func (b *Bootstrap) AddRemoveList() {
	b.initialize(templates.RemoveList, "files/removelist", 0644)
}

func (b *Bootstrap) AddRemoveMinimalList() {
	b.initialize(templates.RemoveMinimalList, "files/removeminimallist", 0644)
}

func (b *Bootstrap) AddUnmaskService() {
	b.initialize(templates.UnmaskService, "files/unmaskService", 0644)
}

func (b *Bootstrap) AddChangeInfo() {
	// 在项目根目录创建 README 文件
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

// CleanRootfs 方法用于清理 rootfs 中不必要的文件和软件包
func (b *Bootstrap) CleanRootfs() error {
	target, _ := filepath.Abs(filepath.Join(b.DestinationDir, "rootfs"))

	// 检查rootfs目录是否存在
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("rootfs 目录不存在，请先运行 'ktib project build-rootfs' 命令")
	}

	// 1. 移除不必要的包
	removeListPath := filepath.Join(b.DestinationDir, "files", "removelist")
	removeMinimalListPath := filepath.Join(b.DestinationDir, "files", "removeminimallist")

	fmt.Printf("正在移除不必要的软件包，镜像类型: %s\n", b.BuildType)
	if err := RemoveUnnecessaryPackages(target, b.BuildType, removeListPath, removeMinimalListPath); err != nil {
		fmt.Printf("警告: 移除不必要的软件包失败: %v\n", err)
	}
	// 2. 移除不必要的文件
	if err := RemoveUnnecessaryFiles(target); err != nil {
		fmt.Printf("移除不必要的文件失败: %v\n", err)
	}

	// 2. 解除服务屏蔽
	unmaskServicePath := filepath.Join(b.DestinationDir, "files", "unmaskService")
	fmt.Println("正在解除服务屏蔽")
	if err := UnmaskServices(target, unmaskServicePath); err != nil {
		fmt.Printf("警告: 解除服务屏蔽失败: %v\n", err)
	}

	// 3. 完整清理文件系统
	fmt.Println("正在清理文件系统")
	if err := CleanupRootfsPath(target); err != nil {
		fmt.Printf("警告: 清理文件系统失败: %v\n", err)
	}

	fmt.Println("rootfs 清理完成")
	return nil
}
