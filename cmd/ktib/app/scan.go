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
	"encoding/json"
	o "gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/report"
	"gitee.com/openeuler/ktib/pkg/scanner/dockerfile"
	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	tt "github.com/aquasecurity/trivy/pkg/types"
	"github.com/aquasecurity/trivy/pkg/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var option o.Option
var args o.Arguments
var logger *log.Logger

const PolicyYaml = "/etc/ktib/policy.yaml"

func init() {
	logger = log.New(os.Stderr, "", log.LstdFlags)
}

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
		Use:   "dockerfile-audit",
		Short: "dockerfile-audit uses its own grammar to parse valid Dockerfiles and deconstruct all directives.",
		Run: func(cmd *cobra.Command, arg []string) {
			GetArgumentsCmd(args)
		},
	}
	initScanDockerfileFlags(cmd)
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

func initScanDockerfileFlags(cmd *cobra.Command) {
	flag := cmd.Flags()
	flag.StringVar(&args.PolicyFile, "policy", PolicyYaml, "The dockerfile policy to use for the audit.")
	flag.StringVar(&args.Dockerfile, "dockerfile", "", "The DockerfileMsg to audit. Can be both a file or a directory.")
	flag.BoolVar(&args.ParseOnly, "parse-only", false, "Simply Parse the DockerfileMsg(s) and return the content, without applying any policy. Only JSON report is supported for this.")
	flag.BoolVar(&args.GenerateJSON, "json", false, "Generate a JSON file with the findings.")
	flag.BoolVar(&args.GenerateReport, "report", false, "Generate a PDF report about the findings.")
	flag.StringVar(&args.JSONOutfile, "outfile", "dockerfile-audit.json", "Name of the JSON file.")
	flag.StringVar(&args.ReportName, "name", "report.pdf", "The name of the PDF report.")
	flag.StringVar(&args.ReportTemplate, "template", "templates/report-template.tex", "The template for the report to use.")
	flag.BoolVar(&args.Verbose, "verbose", false, "Enables debug output.")
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

func GetArgumentsCmd(args o.Arguments) {
	// 获取传入的所有dockerfile路径
	filesToProcess := GetFilesToProcess(args.Dockerfile)
	if args.ParseOnly {
		// 传入等待解析的dockerfile文件集合 []string，返回每个dockerfile的解析结果 []options.parseResult
		parsedFiles := Parse(filesToProcess)
		if len(parsedFiles) == 0 {
			log.Println("No files were processed, reports will be skipped.")
		} else {
			if args.GenerateJSON {
				jsonData, err := json.MarshalIndent(parsedFiles, "", "  ")
				if err != nil {
					log.Fatal(err)
				}

				err = ioutil.WriteFile(args.JSONOutfile, jsonData, 0644)
				if err != nil {
					log.Fatal(err)
				}

				log.Printf("JSON report generated: %s\n", args.JSONOutfile)
			}
		}
	} else {
		policy, err := GetPolicy(args.PolicyFile)
		if err != nil {
			logger.Fatalln("failed to get policy")
		}
		results := Audit(filesToProcess, policy)
		if len(results) == 0 {
			log.Println("No files were processed, reports will be skipped.")
		} else {
			if args.GenerateJSON {
				jsonData, err := json.MarshalIndent(results, "", "  ")
				if err != nil {
					log.Fatal(err)
				}

				err = ioutil.WriteFile(args.JSONOutfile, jsonData, 0644)
				if err != nil {
					log.Fatal(err)
				}

				log.Printf("JSON report generated: %s\n", args.JSONOutfile)
			}

			if args.GenerateReport {
				log.Println("Preparing to generate PDF report.")
				err := report.GenerateLatexReport(*policy, results, args.ReportTemplate, args.ReportName)
				if err != nil {
					return
				}
				log.Printf("PDF report generated: %s\n", args.ReportName)
			}
		}
	}
}

func GetFilesToProcess(argsDockerfile string) []string {
	filesToProcess := make([]string, 0)

	fileInfo, err := os.Stat(argsDockerfile)
	if err != nil {
		log.Fatal(err)
	}

	if fileInfo.IsDir() {
		err := filepath.Walk(argsDockerfile, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				filesToProcess = append(filesToProcess, path)
			}
			return nil
		})
		if err != nil {
			return nil
		}
	} else {
		filesToProcess = append(filesToProcess, argsDockerfile)
	}

	log.Printf("Scanning %d files in %s\n", len(filesToProcess), argsDockerfile)

	return filesToProcess
}

func GetPolicy(policyFile string) (*dockerfile.Policy, error) {
	//TODO: 根据传入的policy.yaml解析出审核策略
}

func Parse(filesToProcess []string) []dockerfile.ParseResult {
	// TODO: 解析dockerfile文件，获取内容
}

func Audit(filesToProcess []string, policy *dockerfile.Policy) []dockerfile.PolicyResult {
	// TODO: 根据策略审核dockerfile内容，获取dockerfile审核结果
}

