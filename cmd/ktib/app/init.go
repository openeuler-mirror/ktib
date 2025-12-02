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

package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"gitee.com/openeuler/ktib/pkg/project"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// InitOption 定义初始化项目的选项
type InitOption struct {
	BuildType  string
	configFile string
}

// PackagesToCheck 定义需要检查的软件包列表
var PackagesToCheck = []string{"containers-common"}

// newCmdProject 创建项目主命令
func newCmdProject() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Run this command in order to create a base project or app project",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Args: cobra.NoArgs,
	}
	cmd.AddCommand(
		newSubCmdDefaultConfig(),
		newSubCmdInit(),
		newSubCmdBuildRootfs(),
		newSubCmdCleanRootfs(),
		newSubCmdBuild(),
	)
	return cmd
}

// newSubCmdDefaultConfig 创建生成默认配置的子命令
func newSubCmdDefaultConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default_config",
		Short: "Run this command in order to generate default config",
		Example: ` # generate default config example
  ktib project default_config > config.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFileName := cmd.OutOrStdout().(*os.File).Name()
			if outputFileName == "" {
				outputFileName = "config.yml"
			}
			if len(args) > 0 || cmd.Flags().NFlag() > 0 {
				return fmt.Errorf("invalid usage. Use 'ktib project default_config > config.yml'")
			}
			return runDefaultConfig(outputFileName)
		},
		Args: cobra.NoArgs,
	}
	cmd.SetOut(os.Stdout)
	return cmd
}

// runDefaultConfig 执行生成默认配置的操作
func runDefaultConfig(outputFileName string) error {
	yamlContent := `packages:
  install_pkgs:
    - yum
    - iproute
    - vim-minimal
    - procps-ng
    - passwd
    # 可以添加更多软件包
    # - package1
    # - package2
network: 
    networking: yes
    hostname: localhost.localdomain
locale: "%_install_langs en_US.UTF-8"
timezone: "Asia/Shanghai"
`
	data := []byte(yamlContent)
	err := ioutil.WriteFile(outputFileName, data, 0644)
	if err != nil {
		fmt.Printf("failed to write file %v\n", err)
	}
	return nil
}

// newSubCmdInit 创建初始化项目结构的子命令
func newSubCmdInit() *cobra.Command {
	var option InitOption
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run this command in order to init a project structure",
		Long: `Init command helps you create an empty project with specified options. 
It creates the necessary directory structure and files to kickstart your project.`,
		Example: ` # Create a project structure
  ktib project init /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, args, option)
		},
		Args: cobra.MinimumNArgs(1),
	}
	return cmd
}

// runInit 执行初始化项目结构的操作
func runInit(c *cobra.Command, args []string, option InitOption) error {
	if len(args) < 1 {
		logrus.Println("The number of parameters passed in is incorrect")
		return c.Help()
	}
	boot := project.NewBootstrap(args[0])

	// 使用新的方法初始化项目结构
	if err := boot.InitProjectStructure(); err != nil {
		return err
	}

	return nil
}

// newSubCmdBuildRootfs 创建构建rootfs的子命令
func newSubCmdBuildRootfs() *cobra.Command {
	var option struct {
		configFile string
	}
	cmd := &cobra.Command{
		Use:   "build-rootfs",
		Short: "Run this command to build rootfs for a project",
		Long: `Build-rootfs command helps you create a rootfs for your project based on the configuration.
It requires a config file that specifies the packages and settings for the rootfs.`,
		Example: ` # Build rootfs for a project
  ktib project build-rootfs --config config.yml /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Println("The number of parameters passed in is incorrect")
				return cmd.Help()
			}
			if option.configFile == "" {
				return fmt.Errorf("when building rootfs, you need to specify the --config")
			}

			// 执行 rootfs 构建
			boot := project.NewBootstrap(args[0])
			return boot.BuildRootfs(option.configFile)
		},
		Args: cobra.MinimumNArgs(1),
	}
	flags := cmd.Flags()
	flags.StringVar(&option.configFile, "config", "", "path to config file (required)")
	cmd.MarkFlagRequired("config")
	return cmd
}

// newSubCmdCleanRootfs 创建清理rootfs的子命令
func newSubCmdCleanRootfs() *cobra.Command {
	var option struct {
		imageType string
	}

	// 定义有效的镜像类型
	validImageTypes := []string{"micro", "minimal", "platform", "init"}

	cmd := &cobra.Command{
		Use:   "clean-rootfs",
		Short: "Run this command to clean unnecessary files and packages in rootfs",
		Long: `Clean-rootfs command helps you remove unnecessary files and packages from your rootfs.
It also performs additional environment configuration operations to optimize the image size.`,
		Example: ` # Clean rootfs for a project
  ktib project clean-rootfs --type minimal /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Println("The number of parameters passed in is incorrect")
				return cmd.Help()
			}

			boot := project.NewBootstrap(args[0])

			// 如果指定了镜像类型，则进行校验并设置
			if option.imageType != "" {
				// 校验镜像类型
				valid := false
				for _, t := range validImageTypes {
					if option.imageType == t {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("无效的镜像类型: %s。有效的类型包括: %s",
						option.imageType, strings.Join(validImageTypes, ", "))
				}

				// 设置镜像类型
				boot.BuildType = option.imageType
			}

			// 执行清理操作
			if err := boot.CleanRootfs(); err != nil {
				return fmt.Errorf("Failed to clean rootfs: %v", err)
			}

			logrus.Println("Successfully cleaned rootfs")
			return nil
		},
		Args: cobra.MinimumNArgs(1),
		// 为 clean-rootfs 命令添加路径补全
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// 返回目录补全
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
	}

	// 确保标志被正确添加到命令的标志集合中
	cmd.Flags().StringVar(&option.imageType, "type", "",
		fmt.Sprintf("Type of image (%s)", strings.Join(validImageTypes, ", ")))

	// 为镜像类型标志添加自动补全
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validImageTypes, cobra.ShellCompDirectiveDefault
	})

	return cmd
}

// newSubCmdBuild 创建构建容器镜像的子命令
func newSubCmdBuild() *cobra.Command {
	var option struct {
		imageName string
		tag       string
	}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Run this command to build a container image from rootfs",
		Long: `Build command helps you create a container image using the rootfs and Dockerfile.
It packages the rootfs into a container image that can be used with container runtimes.`,
		Example: ` # Build container image for a project
  ktib project build --name myimage --tag latest /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Println("The number of parameters passed in is incorrect")
				return cmd.Help()
			}

			// 执行镜像构建
			boot := project.NewBootstrap(args[0])
			return boot.BuildImage(option.imageName, option.tag)
		},
		Args: cobra.MinimumNArgs(1),
	}
	flags := cmd.Flags()
	flags.StringVar(&option.imageName, "name", "ktib-image", "name of the container image")
	flags.StringVar(&option.tag, "tag", "latest", "tag of the container image")
	return cmd
}

// checkRpmPackageInstalled 检查RPM包是否已安装
func checkRpmPackageInstalled(packageName string) error {
	// 运行 rpm 命令来检查包是否已安装
	cmd := exec.Command("rpm", "-q", packageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 运行命令时出错
		return fmt.Errorf("running cmd failed")
	}
	// 检查包是否已安装
	if !strings.Contains(string(output), packageName) {
		return fmt.Errorf("%s 包未安装", packageName)
	}
	return nil
}
