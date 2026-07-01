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
	"os"
	"strings"

	"gitee.com/openeuler/ktib/pkg/project"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	//"gopkg.in/yaml.v2"
)

// InitOption defines options for initializing a project
type InitOption struct {
	BuildType  string
	configFile string
}

// PackagesToCheck defines the list of packages that need to be checked
var PackagesToCheck = []string{"containers-common"}

// newCmdProject creates the main project command
func newCmdProject() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project",
		Aliases: []string{"p"},
		Short:   "Manage project initialization and image build workflows",
		Args:    cobra.NoArgs,
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

// newSubCmdDefaultConfig creates the subcommand for generating default configuration
func newSubCmdDefaultConfig() *cobra.Command {
	var option struct {
		timezone  string
		locale    string
		imageType string
	}

	// Define valid image types
	validImageTypes := project.ValidImageTypes()

	cmd := &cobra.Command{
		Use:     "default_config",
		Aliases: []string{"dc"},
		Short:   "Run this command in order to generate default config",
		Example: ` # generate default config example
  ktib project default_config > config.yml
  # generate default config with custom timezone
  ktib project default_config --timezone "America/New_York" > config.yml
  # generate default config with custom locale
  ktib project default_config --locale "zh_CN.UTF-8" > config.yml
  # generate default config with custom type
  ktib project default_config --type minimal > config.yml
  # generate default config with all custom options
  ktib project default_config --timezone "Europe/London" --locale "en_GB.UTF-8" --type platform > config.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFileName := cmd.OutOrStdout().(*os.File).Name()
			if outputFileName == "" {
				outputFileName = "config.yml"
			}

			// If image type is specified, perform validation
			if option.imageType != "" {
				// Validate image type
				valid := false
				for _, t := range validImageTypes {
					if option.imageType == t {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("invalid image type: %s. Valid types include: %s",
						option.imageType, strings.Join(validImageTypes, ", "))
				}
			}

			return project.WriteDefaultConfig(outputFileName, option.timezone, option.locale, option.imageType)
		},
		Args: cobra.NoArgs,
	}
	cmd.SetOut(os.Stdout)
	// Add timezone option
	cmd.Flags().StringVar(&option.timezone, "timezone", project.DefaultTimezone, "Set the timezone for the configuration (e.g., Asia/Shanghai, America/New_York, Europe/London)")
	// Add locale option
	cmd.Flags().StringVar(&option.locale, "locale", project.DefaultLocale, "Set the locale for the configuration (e.g., C.UTF-8, zh_CN.UTF-8, en_GB.UTF-8)")
	// Add type option
	cmd.Flags().StringVar(&option.imageType, "type", "",
		fmt.Sprintf("Type of image (%s)", strings.Join(validImageTypes, ", ")))

	// Add auto-completion for image type flag
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validImageTypes, cobra.ShellCompDirectiveDefault
	})

	return cmd
}

// newSubCmdInit creates the subcommand for initializing the project structure
func newSubCmdInit() *cobra.Command {
	var option InitOption
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize an image project directory structure",
		Long:  "Init creates the directory skeleton (dockerfile/, rootfs/, files/, tests/) and lays down default templates like Dockerfile, README, removeminimallist and unmaskService.",
		Example: ` # Create a new project skeleton
  ktib project init /path/to/project

 # Init with type and write a default config file
  ktib project init --type init /path/to/project

 # Build rootfs using the generated config
  ktib project build-rootfs --config /path/to/project/config.yml /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, args, option)
		},
		Args: cobra.MinimumNArgs(1),
	}
	types := project.ValidImageTypes()
	flags := cmd.Flags()
	flags.StringVar(&option.BuildType, "type", project.DefaultProjectImageType, "Type of image ("+strings.Join(types, ", ")+")")
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return types, cobra.ShellCompDirectiveDefault
	})
	return cmd
}

// runInit executes the operation to initialize the project structure
func runInit(c *cobra.Command, args []string, option InitOption) error {
	if len(args) < 1 {
		logrus.Println("The number of parameters passed in is incorrect")
		return c.Help()
	}
	return project.NewWorkflowService().InitProject(project.ProjectWorkflowRequest{
		ProjectDir: args[0],
		ImageType:  option.BuildType,
	})
}

// newSubCmdBuildRootfs creates the subcommand for building rootfs
func newSubCmdBuildRootfs() *cobra.Command {
	var option struct {
		configFile string
	}
	cmd := &cobra.Command{
		Use:     "build-rootfs",
		Aliases: []string{"br"},
		Short:   "Build rootfs according to the given config.yml",
		Long:    "Build-rootfs installs packages into the project rootfs using yum/dnf with nodocs, sets network/locale/timezone and initializes machine-id. A config file is required.",
		Example: ` # Build rootfs for a project
  ktib project build-rootfs --config /path/to/project/config.yml /path/to/project

 # Generate a default config first
  ktib project default_config > /path/to/project/config.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Println("The number of parameters passed in is incorrect")
				return cmd.Help()
			}
			return project.NewWorkflowService().BuildRootfs(project.ProjectWorkflowRequest{
				ProjectDir: args[0],
				ConfigPath: option.configFile,
			})
		},
		Args: cobra.MinimumNArgs(1),
	}
	flags := cmd.Flags()
	flags.StringVar(&option.configFile, "config", "", "path to config file (required)")
	cmd.MarkFlagRequired("config")
	return cmd
}

// newSubCmdCleanRootfs creates the subcommand for cleaning rootfs
func newSubCmdCleanRootfs() *cobra.Command {
	var option struct {
		imageType string
		locale    string
	}

	validImageTypes := project.ValidImageTypes()

	cmd := &cobra.Command{
		Use:     "clean-rootfs",
		Aliases: []string{"cr"},
		Short:   "Clean rootfs by removing files, optional packages, and unmasking services",
		Long:    "Clean-rootfs removes locales/docs/caches/logs/tmp, optionally removes packages per type (e.g., minimal), unmask services, and performs final cleanup. Use --type to apply type-specific rules. Use --locale to preserve locale data for the specified locale.",
		Example: ` # Clean rootfs with minimal rules
  ktib project clean-rootfs --type minimal /path/to/project

 # Clean rootfs for init/platform types
  ktib project clean-rootfs --type init /path/to/project

 # Clean rootfs preserving locale data
  ktib project clean-rootfs --locale C.UTF-8 /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Println("The number of parameters passed in is incorrect")
				return cmd.Help()
			}

			return project.NewWorkflowService().CleanRootfs(project.ProjectWorkflowRequest{
				ProjectDir: args[0],
				ImageType:  option.imageType,
				Locale:     option.locale,
			})
		},
		Args: cobra.MinimumNArgs(1),
		// Add path completion for the clean-rootfs command
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// Return directory completion
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
	}

	// Ensure flags are correctly added to the command's flag set
	cmd.Flags().StringVar(&option.imageType, "type", "",
		fmt.Sprintf("Type of image (%s)", strings.Join(validImageTypes, ", ")))
	cmd.Flags().StringVar(&option.locale, "locale", "", "保留指定 locale 的数据目录（如 C.UTF-8），未指定则全删")

	// Add auto-completion for image type flag
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validImageTypes, cobra.ShellCompDirectiveDefault
	})

	return cmd
}

// newSubCmdBuild creates the subcommand for building a container image
func newSubCmdBuild() *cobra.Command {
	var option struct {
		imageName string
		tag       string
		locale    string
	}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the container image from the prepared rootfs",
		Long:  "Build packages rootfs.tar, resolves Dockerfile (prefers project/dockerfile/Dockerfile) and uses buildah to produce the image with the given name:tag. Use --locale to inject ENV LANG into the Dockerfile.",
		Example: ` # Build container image for a project
  ktib project build --name myimage --tag latest /path/to/project

 # Build with locale injection
  ktib project build --locale zh_CN.UTF-8 /path/to/project

 # Build with defaults
  ktib project build /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Println("The number of parameters passed in is incorrect")
				return cmd.Help()
			}
			return project.NewWorkflowService().BuildImage(project.ProjectWorkflowRequest{
				ProjectDir: args[0],
				ImageName:  option.imageName,
				Tag:        option.tag,
				Locale:     option.locale,
			})
		},
		Args: cobra.MinimumNArgs(1),
	}
	flags := cmd.Flags()
	flags.StringVar(&option.imageName, "name", project.DefaultImageName, "name of the container image")
	flags.StringVar(&option.tag, "tag", project.DefaultImageTag, "tag of the container image")
	flags.StringVar(&option.locale, "locale", "", "Inject ENV LANG into Dockerfile (e.g., C.UTF-8, zh_CN.UTF-8)")
	return cmd
}

/*
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
*/
