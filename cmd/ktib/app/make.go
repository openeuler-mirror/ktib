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
	"path/filepath"
	"strings"

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
}

func runMake(cmd *cobra.Command, args []string, option makeOption) error {
	if len(args) < 1 {
		logrus.Println("The number of parameters passed in is incorrect")
		return cmd.Help()
	}
	if option.init {
		boot := project.NewBootstrap(args[0])
		if err := boot.InitProjectStructure(); err != nil {
			return err
		}
		if option.config == "" {
			option.config = filepath.Join(args[0], "config.yml")
			if err := runDefaultConfig(option.config); err != nil {
				return err
			}
		}
	}
	if option.config == "" {
		return fmt.Errorf("when building rootfs, you need to specify the --config")
	}

	validImageTypes := []string{"micro", "minimal", "platform", "init"}

	boot := project.NewBootstrap(args[0])
	if option.imageType != "" {
		valid := false
		for _, t := range validImageTypes {
			if option.imageType == t {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("无效的镜像类型: %s。有效的类型包括: %s", option.imageType, strings.Join(validImageTypes, ", "))
		}
		boot.BuildType = option.imageType
	}

	logrus.Println("Building rootfs ...")
	if err := boot.BuildRootfs(option.config); err != nil {
		return err
	}

	logrus.Println("Cleaning rootfs ...")
	if err := boot.CleanRootfs(); err != nil {
		return err
	}

	imageName := option.imageName
	if imageName == "" {
		imageName = "ktib-image"
	}
	tag := option.tag
	if tag == "" {
		tag = "latest"
	}

	logrus.Println("Building image ...")
	if err := boot.BuildImage(imageName, tag); err != nil {
		return err
	}

	logrus.Println("Make completed")
	return nil
}

func newCmdMake() *cobra.Command {
	var options makeOption
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Run this command to build a base image at once",
		Example: ` # Init and build base image in one command
  ktib make --init --type minimal --name myimage --tag latest /path/to/project

 # Or build with a specified config
  ktib make --config config.yml --type minimal --name myimage --tag latest /path/to/project`,
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
	flags.StringVar(&options.imageType, "type", "platform", "Type of image (micro|minimal|platform|init)")
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"micro", "minimal", "platform", "init"}, cobra.ShellCompDirectiveDefault
	})
	flags.StringVar(&options.imageName, "name", "ktib-image", "name of the container image")
	flags.StringVar(&options.tag, "tag", "latest", "tag of the container image")
	return cmd
}
