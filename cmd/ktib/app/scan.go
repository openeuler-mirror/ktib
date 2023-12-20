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
	"context"
	"fmt"
	o "gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/spf13/cobra"
	"github.com/urfave/cli/v2"
)

func runScan(c *cobra.Command, args []string, opt o.Option) error {
	//TODO 解析context 构造scanner  in pkg/scanner
	var ctx cli.Context
	option, err := artifact.InitOption(&ctx)
	if err != nil {
		return err
	}
	//TODO: 需要对比ktib option和trivy option的区别，参数不足需要额外赋值
	opt.Option = option
	// 1111111111
	fmt.Println(option)
	runner, err := artifact.NewRunner(opt.Option)
	if err != nil {
		return err
	}
	defer runner.Close(context.Background())
	var report types.Report
	re := report.Report
	switch c.Use {
	case "Source":
		// TODO  report = runner.ScanSource()
		re, err = runner.ScanFilesystem(context.Background(), opt.Option)
		if err != nil {
			return err
		}
	case "RPMs":
		// TODO  report = runner.ScanRPMs()
		return nil
	case "Dockerfile":
		re, err = runner.ScanFilesystem(context.Background(), opt.Option)
		if err != nil {
			return err
		}
	}
	re, err = runner.Filter(context.Background(), opt.Option, report.Report)
	if err != nil {
		return err
	}
	if err = runner.Report(opt.Option, re); err != nil {
		return err
	}
	return nil

}

func newCmdScan() *cobra.Command {
	// TODO 构造命令 command args flag 等
	var option o.Option
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Run this command in order to scan source, rpms, dockerfile ...",
	}
	// TODO 添加子命令 scan source, rpms, dockerfile
	cmd.AddCommand(
		newSubCmdSource(),
		newSubCmdRPMs(),
		newSubCmdDokcerfile(),
	)

	// TODO 添加flag参数
	flag := cmd.Flags()
	flag.StringVarP(&option.Driver, "diver", "d", "kysec-CIS", "support dockerfile-audit|trivy|kysec-CIS")
	return cmd
}

func newSubCmdSource() *cobra.Command {
	var option o.Option
	cmd := &cobra.Command{
		Use:     "Source",
		Aliases: []string{"-S"},
		Short:   "Run this command in order to scan Source ...",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, option)
		},
		Args: cobra.MinimumNArgs(1),
	}
	return cmd
}

func newSubCmdRPMs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "RPMs",
		Short: "Run this command in order to scan RPMs ...",
	}
	return cmd
}

func newSubCmdDokcerfile() *cobra.Command {
	var option o.Option
	cmd := &cobra.Command{
		Use:     "Dockerfile",
		Aliases: []string{"-D"},
		Short:   "Run this command in order to scan dockerfile ...",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, option)
		},
		Args: cobra.MinimumNArgs(1),
	}
	return cmd
}
