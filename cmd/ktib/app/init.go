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
	"gitee.com/openeuler/ktib/pkg/utils"
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
	validImageTypes := []string{"micro", "minimal", "platform", "init"}

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
					return fmt.Errorf("invalid image type: %s。Valid types include: %s",
						option.imageType, strings.Join(validImageTypes, ", "))
				}
			}

			return runDefaultConfig(outputFileName, option.timezone, option.locale, option.imageType)
		},
		Args: cobra.NoArgs,
	}
	cmd.SetOut(os.Stdout)
	// Add timezone option
	cmd.Flags().StringVar(&option.timezone, "timezone", "Asia/Shanghai", "Set the timezone for the configuration (e.g., Asia/Shanghai, America/New_York, Europe/London)")
	// Add locale option
	cmd.Flags().StringVar(&option.locale, "locale", "en_US.UTF-8", "Set the locale for the configuration (e.g., en_US.UTF-8, zh_CN.UTF-8, en_GB.UTF-8)")
	// Add type option
	cmd.Flags().StringVar(&option.imageType, "type", "",
		fmt.Sprintf("Type of image (%s)", strings.Join(validImageTypes, ", ")))

	// Add auto-completion for image type flag
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validImageTypes, cobra.ShellCompDirectiveDefault
	})

	return cmd
}

// runDefaultConfig executes the operation to generate the default configuration
func runDefaultConfig(outputFileName, timezone, locale, imageType string) error {
	// Get the corresponding package list based on the type
	packages := getPackagesByType(imageType)

	yamlContent := fmt.Sprintf(`packages:
  install_pkgs:
%s
network: 
    networking: yes
    hostname: localhost.localdomain
locale: "%%_install_langs %s"
timezone: "%s"
`, packages, locale, timezone)
	data := []byte(yamlContent)
	err := os.WriteFile(outputFileName, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file %s: %v", outputFileName, err)
	}
	return nil
}

// getPackagesByType returns the corresponding package list based on the type (in YAML content format)
func getPackagesByType(imageType string) string {
	var packages []string

	switch imageType {
	case "init":
		// init type: includes package manager, most basic tools, and systemd debugging tools
		packages = []string{
			"yum",
			"vim-minimal",
			"dbus-daemon",
			"kbd",
			"util-linux",
		}
	case "platform":
		// platform type: suitable for traditional business scenarios, includes package manager and most basic tools image
		packages = []string{
			"yum",
			"vim-minimal",
			"shadow",
		}
	case "minimal":
		// minimal type: minimal installation, includes only necessary basic packages, does not include Python
		packages = []string{
			"microdnf",
			"vim-minimal",
		}
	case "micro":
		// micro type: ultra-minimal installation, includes only the most core packages
		packages = []string{
			"coreutils",
		}
	default:
		// Default to using the package list for platform type
		packages = []string{
			"yum",
			"vim-minimal",
			"shadow",
		}
	}

	// Format the package list into YAML format
	var packagesYAML string
	for _, pkg := range packages {
		packagesYAML += fmt.Sprintf("    - %s\n", pkg)
	}
	packagesYAML += "    # You can add more packages\n    # - package1\n    # - package2"

	return packagesYAML
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
	types := utils.ValidImageTypes
	flags := cmd.Flags()
	flags.StringVar(&option.BuildType, "type", "platform", "Type of image ("+strings.Join(types, ", ")+")")
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
	boot := project.NewBootstrap(args[0])

	if option.BuildType != "" {
		validTypes := utils.ValidImageTypes
		if !utils.IsValidImageType(option.BuildType) {
			return fmt.Errorf("invalid image type: %s. Valid types include: %s", option.BuildType, strings.Join(validTypes, ", "))
		}
		boot.BuildType = option.BuildType
	}

	// Use the new method to initialize the project structure
	if err := boot.InitProjectStructure(); err != nil {
		return err
	}
	return nil
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
			if option.configFile == "" {
				return fmt.Errorf("when building rootfs, you need to specify the --config")
			}

			// Execute rootfs build
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

// newSubCmdCleanRootfs creates the subcommand for cleaning rootfs
func newSubCmdCleanRootfs() *cobra.Command {
	var option struct {
		imageType string
	}

	validImageTypes := utils.ValidImageTypes

	cmd := &cobra.Command{
		Use:     "clean-rootfs",
		Aliases: []string{"cr"},
		Short:   "Clean rootfs by removing files, optional packages, and unmasking services",
		Long:    "Clean-rootfs removes locales/docs/caches/logs/tmp, optionally removes packages per type (e.g., minimal), unmask services, and performs final cleanup. Use --type to apply type-specific rules.",
		Example: ` # Clean rootfs with minimal rules
  ktib project clean-rootfs --type minimal /path/to/project

 # Clean rootfs for init/platform types
  ktib project clean-rootfs --type init /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Println("The number of parameters passed in is incorrect")
				return cmd.Help()
			}

			boot := project.NewBootstrap(args[0])

			// If image type is specified, perform validation and set it
			if option.imageType != "" {
				if !utils.IsValidImageType(option.imageType) {
					return fmt.Errorf("invalid image type: %s. Valid types include: %s",
						option.imageType, strings.Join(validImageTypes, ", "))
				}
				boot.BuildType = option.imageType
			}

			// Execute the cleanup operation
			if err := boot.CleanRootfs(); err != nil {
				return fmt.Errorf("failed to clean rootfs: %v", err)
			}

			logrus.Println("Successfully cleaned rootfs")
			return nil
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
	}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the container image from the prepared rootfs",
		Long:  "Build packages rootfs.tar, resolves Dockerfile (prefers project/dockerfile/Dockerfile) and uses buildah to produce the image with the given name:tag.",
		Example: ` # Build container image for a project
  ktib project build --name myimage --tag latest /path/to/project

 # Build with defaults
  ktib project build /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				logrus.Println("The number of parameters passed in is incorrect")
				return cmd.Help()
			}

			// Execute image build
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
