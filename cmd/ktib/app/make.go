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
	"gitee.com/openeuler/ktib/pkg/project"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type makeOption struct {
	config    string
	imageType string
	imageName string
	tag       string
	init      bool
	timezone  string
	locale    string
}

func runMake(cmd *cobra.Command, args []string, option makeOption) error {
	if len(args) < 1 {
		logrus.Println("The number of parameters passed in is incorrect")
		return cmd.Help()
	}
	return project.NewWorkflowService().MakeImage(project.ProjectWorkflowRequest{
		ProjectDir: args[0],
		ImageType:  option.imageType,
		ConfigPath: option.config,
		ImageName:  option.imageName,
		Tag:        option.tag,
		Init:       option.init,
		Timezone:   option.timezone,
		Locale:     option.locale,
	})
}

func newCmdMake() *cobra.Command {
	var options makeOption
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Run this command to build a base image at once",
		Example: ` # Init and build base image in one command
  ktib make --init --type minimal --name myimage --tag latest /path/to/project

 # Or build with a specified config
  ktib make --config config.yml --type minimal --name myimage --tag latest /path/to/project

 # Init and build with custom timezone and locale
  ktib make --init --timezone "America/New_York" --locale "zh_CN.UTF-8" /path/to/project

 # Init and build with custom timezone only
  ktib make --init --timezone "Europe/London" /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMake(cmd, args, options)
		},
		Args: cobra.MinimumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&options.config, "config", "", "path to config file (optional with --init; default writes project/config.yml)")
	flags.BoolVar(&options.init, "init", false, "init project structure before build; generate default config when not set")
	flags.StringVar(&options.imageType, "type", project.DefaultProjectImageType, "Type of image (micro|minimal|platform|init)")
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return project.ValidImageTypes(), cobra.ShellCompDirectiveDefault
	})
	flags.StringVar(&options.imageName, "name", project.DefaultImageName, "name of the container image")
	flags.StringVar(&options.tag, "tag", project.DefaultImageTag, "tag of the container image")
	// Add timezone option
	flags.StringVar(&options.timezone, "timezone", project.DefaultTimezone, "Set the timezone for the configuration (e.g., Asia/Shanghai, America/New_York, Europe/London)")
	// Add locale option
	flags.StringVar(&options.locale, "locale", project.DefaultLocale, "Set the locale for the configuration (e.g., en_US.UTF-8, zh_CN.UTF-8, en_GB.UTF-8)")
	return cmd
}
