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
	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/podman/v4/cmd/podman/common"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func commit(cmd *cobra.Command, args []string, option *options.CommitOption) error {
	containerid := strings.TrimPrefix(args[0], "/")
	if len(args) == 2 {
		option.ImageName = args[1]
	}
	if !option.Quiet {
		option.Writer = os.Stderr
	}
	store, err := utils.GetStore(cmd)
	store.GraphRoot()
	if err != nil {
		return err
	}
	op := builder.BuilderOptions{
		FromImage: args[1],
		Container: containerid,
	}
	cmBuilder, err := builder.NewBuilder(store, op)
	if err != nil {
		return err
	}
	return cmBuilder.Commit(containerid, option)
}

func COMMITCmd() *cobra.Command {
	var op options.CommitOption
	cmd := &cobra.Command{
		Use:  "commit",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commit(cmd, args, &op)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&op.Maintainer, "maintianer", "", "")
	flags.StringVar(&op.Message, "message", "", "")
	flags.BoolVar(&op.Remove, "rm", false, "")
	flags.StringVar(&op.EntryPoint, "entrypoint", "", "")
	flags.StringArrayVar(&op.CMD, "CMD", []string{}, "")
	flags.StringArrayVar(&op.Env, "env", []string{}, "")
	formatFlagName := "format"
	flags.StringVarP(&op.Format, formatFlagName, "f", "oci", "`Format` of the image manifest and metadata")
	_ = cmd.RegisterFlagCompletionFunc(formatFlagName, common.AutocompleteImageFormat)
	return cmd
}
