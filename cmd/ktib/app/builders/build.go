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

package builders

import (
	"errors"
	"fmt"
	"path/filepath"

	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func BUILDCmd() *cobra.Command {
	var op options.BuildOptions
	cmd := &cobra.Command{
		Use:   "build",
		Short: "build an image",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return build(cmd, args, &op)
		},
	}
	flags := cmd.Flags()
	flags.StringArrayVarP(&op.File, "file", "f", []string{""}, "Name of the Dockerfile (Default is 'PATH/Dockerfile')")
	flags.StringVarP(&op.Tags, "tag", "t", "none", "tagged name to apply to the build image")
	return cmd
}

func build(cmd *cobra.Command, args []string, op *options.BuildOptions) error {
	var dockerfiles []string
	dockerfiles = op.File
	contextDir := ""
	if len(args) > 0 {
		absDir, err := filepath.Abs(args[0])
		if err != nil {
			return errors.New("error determining path to directory")
		}
		contextDir = absDir
	} else {
		return errors.New("no context directory specified")
	}

	if contextDir == "" {
		return errors.New("no context directory specified, and no dockerfile specified")
	}

	if len(dockerfiles) == 0 {
		dockerfiles = append(dockerfiles, filepath.Join(contextDir, "Dockerfile"))
	}

	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	if err := builder.BuildDockerfiles(store, op, dockerfiles...); err != nil {
		fmt.Printf("error build dockerfiles %v\n", err)
	}
	return nil
}
