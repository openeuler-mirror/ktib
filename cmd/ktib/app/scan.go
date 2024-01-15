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
	o "gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	tt "github.com/aquasecurity/trivy/pkg/types"
	"github.com/aquasecurity/trivy/pkg/utils"
	"github.com/spf13/cobra"
)

var option o.Option

func runScan(c *cobra.Command, args []string, opt o.Option) error {
	scanOption, err := o.InitScanOption(args, opt)
	runner, err := artifact.NewRunner(scanOption)
	if err != nil {
		return err
	}
	defer runner.Close(context.Background())
	var report types.Report
	re := report.Report
	switch c.Use {
	case "Source":
		re, err = sourceRun(runner, context.Background(), scanOption)
		if err != nil {
			return err
		}
	case "RPMs":
		// TODO  report = runner.ScanRPMs()
		return nil
	case "Dockerfile":
		re, err = configRun(runner, context.Background(), scanOption)
		if err != nil {
			return err
		}
	}
	re, err = runner.Filter(context.Background(), scanOption, re)
	if err != nil {
		return err
	}
	if err = runner.Report(scanOption, re); err != nil {
		return err
	}
	return nil
}

func newCmdScan() *cobra.Command {
	// TODO 构造命令 command args flag 等
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
	cmd := &cobra.Command{
		Use:   "Source",
		Short: "Run this command in order to scan Source ...",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, option)
		},
		Args: cobra.MinimumNArgs(1),
	}
	initFlags(cmd)
	return cmd
}

func newSubCmdDokcerfile() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "Dockerfile",
		Short: "Run this command in order to scan dockerfile ...",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, option)
		},
		Args: cobra.MinimumNArgs(1),
	}
	initFlags(cmd)
	return cmd
}

func newSubCmdRPMs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "RPMs",
		Short: "Run this command in order to scan RPMs ...",
	}
	initFlags(cmd)
	return cmd
}

func initFlags(cmd *cobra.Command) {
	flag := cmd.Flags()
	flag.StringArrayVar(&option.PolicyNamespaces, "namespaces", []string{"users"}, "Rego namespaces")
	flag.StringVar(&option.CacheDir, "cache-dir", utils.DefaultCacheDir(), "cache directory")
	flag.StringVar(&option.Format, "format", "table", "report format table")
}

func configRun(runner artifact.Runner, ctx context.Context, sop artifact.Option) (tt.Report, error) {
	sop.DisabledAnalyzers = append(analyzer.TypeOSes, analyzer.TypeLanguages...)
	sop.VulnType = nil
	sop.SecurityChecks = []string{tt.SecurityCheckConfig}
	report, err := runner.ScanFilesystem(ctx, sop)
	return report, err
}

func sourceRun(runner artifact.Runner, ctx context.Context, sop artifact.Option) (tt.Report, error) {
	sop.VulnType = []string{tt.VulnTypeOS, tt.VulnTypeLibrary}
	sop.SecurityChecks = []string{tt.SecurityCheckVulnerability, tt.SecurityCheckSecret}
	sop.SkipDBUpdate = true
	//TODO: 数据库来源需要定
	sop.DBRepository = ""
	//TODO: DB PANIC
	report, err := runner.ScanFilesystem(ctx, sop)
	return report, err
}
