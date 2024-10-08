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
	"encoding/json"
	o "gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/report"
	"gitee.com/openeuler/ktib/pkg/scanner/dockerfile"
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

func newCmdScan() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Run this command in order to scan dockerfile",
	}
	cmd.AddCommand(
		newSubCmdDokcerfile(),
	)

	// TODO 添加flag参数
	flag := cmd.Flags()
	flag.StringVarP(&option.Driver, "diver", "d", "dockerfile-audit", "support dockerfile-audit|trivy|kysec-CIS")
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
	policy, err := dockerfile.NewDockerfilePolicy(policyFile)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}
	return policy, nil
}

func Parse(filesToProcess []string) []dockerfile.ParseResult {
	parsedFiles := make([]dockerfile.ParseResult, 0)
	auditor := dockerfile.NewDockerfileAuditor(dockerfile.Policy{})

	for _, file := range filesToProcess {
		content, err := auditor.ParseOnly(file)
		if err == nil {
			parsedFiles = append(parsedFiles, content)
		}
	}

	return parsedFiles
}

func Audit(filesToProcess []string, policy *dockerfile.Policy) []dockerfile.PolicyResult {
	auditor := dockerfile.NewDockerfileAuditor(*policy)
	results := make([]dockerfile.PolicyResult, 0)
	for _, file := range filesToProcess {
		result, err := auditor.Audit(file)
		if err == nil {
			results = append(results, result)
			log.Printf("Scanning file: %s\n", file)
		}
	}
	return results
}
