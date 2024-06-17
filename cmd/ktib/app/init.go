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
	//"gopkg.in/yaml.v2"
)

type InitOption struct {
	BuildType  string
	configFile string
}

var PackagesToCheck = []string{"containers-common"}

func runInit(c *cobra.Command, args []string, option InitOption) error {
	if len(args) < 1 {
		logrus.Println("The number of parameters passed in is incorrect")
		return c.Help()
	}
	boot := project.NewBootstrap(args[0])
	boot.InitWorkDir(option.BuildType, option.configFile)
	boot.AddDockerfile()
	boot.AddScript()
	boot.AddTestcase()
	boot.AddChangeInfo()
	return nil
}

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
		newSubCmdInit(),
		newSubCmdDefaultConfig(),
	)
	return cmd
}

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

func newSubCmdInit() *cobra.Command {
	var option InitOption
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run this command in order to init a project",
		Long: `Init command helps you create an empty project with specified options. 
It creates the necessary directory structure and files to kickstart your project.`,
		Example: ` # Create a project with appImage options
  ktib project init  --buildType appImage /path/to/project (default is appImage)
  # Create a project with baseImage build type by specifying congfig
  ktib project init --buildType baseImage --config config.yml /path/to/project `,
		RunE: func(cmd *cobra.Command, args []string) error {
			if option.BuildType == "baseimage" && option.configFile == "" {
				return fmt.Errorf("when building baseimage rootfs,you need to specify the --config")
			}
			return runInit(cmd, args, option)
		},
		Args: cobra.MinimumNArgs(1),
	}
	flags := cmd.Flags()
	flags.StringVar(&option.BuildType, "buildType", "appimage", "")
	flags.StringVar(&option.configFile, "config", "config.yml", "path to config file")
	return cmd
}

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

func runDefaultConfig(outputFileName string) error {
	yamlContent := `packages:
  install_pkgs:
    - yum
    - iproute
    - vim-minimal
    - procps-ng
    - passwd
network: "NETWORKING=yes\nHOSTNAME=localhost.localdomain\n"
infra: "container"
locale: "%_install_langs en_US.UTF-8"
machine-id: ""
`
	data := []byte(yamlContent)
	err := ioutil.WriteFile(outputFileName, data, 0644)
	if err != nil {
		fmt.Printf("failed to write file %v\n", err)
	}
	return nil
}
