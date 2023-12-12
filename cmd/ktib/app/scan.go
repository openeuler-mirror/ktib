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
	o "gitee.com/openeuler/ktib/cmd/ktib/app/options"
	"github.com/spf13/cobra"
)

func runScan(c *cobra.Command, args []string, opt o.Option) error {
	//TODO 解析context 构造scanner  in pkg/scanner
	if len(args) > 0 {
		return nil
	}
	runner, err := NewRunner(opt)
	if err != nil {
		// TODO
		return err
	}
	switch c.Use {
	case "Source":
		// TODO  report = runner.ScanSource()
		return nil
	case "RPMs":
		// TODO  report = runner.ScanRPMs()
		return nil
	case "Dockerfile":
		// TODO  report = runner.ScanDockerfile()
		_, err := runner.ScanDockerfile(context.Background())
		if err != nil {
			return err
		}
		return nil
	default:
		// 默认走...
		return nil
	}

}

func newCmdScan() *cobra.Command {
	// TODO 构造命令 command args flag 等
	var option o.Option
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Run this command in order to scan source, rpms, dockerfile ...",
		//RunE: func(cmd *cobra.Command, args []string) error {
		//	return runScan(cmd, args, option)
		//},
		//Args: cobra.NoArgs,
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
	cmd := &cobra.Command{
		Use:   "Source",
		Short: "Run this command in order to scan Source ...",
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
		Use:   "Dockerfile",
		Short: "Run this command in order to scan dockerfile ...",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, option)
		},
		Args: cobra.NoArgs,
	}

	// TODO 添加flag
	flag := cmd.Flags()
	flag.StringVarP(&option.Driver, "diver", "d", "kysec-CIS", "support dockerfile-audit|trivy|kysec-CIS")
	return cmd
}
