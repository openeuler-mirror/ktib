/*
   Copyright (c) 2025 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"

	o "gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/scanner/dockerfile"
	"github.com/sirupsen/logrus"
)

func RunDockerfileAudit(args o.Arguments) {
	filesToProcess := getFilesToProcess(args.Dockerfile)
	if len(filesToProcess) == 0 {
		logrus.Info("No Dockerfiles found to process")
		return
	}

	if args.ParseOnly {
		parsedFiles := parse(filesToProcess)
		if len(parsedFiles) == 0 {
			logrus.Info("No files were processed, reports will be skipped.")
			return
		}
		jsonData, err := json.MarshalIndent(parsedFiles, "", "  ")
		if err != nil {
			logrus.Fatal(err)
		}
		err = os.WriteFile(args.JSONOutfile, jsonData, 0644)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("JSON report generated: %s", args.JSONOutfile)
		return
	}

	policy, err := getPolicy(args.PolicyFile)
	if err != nil {
		logrus.Fatalf("Failed to get policy: %v", err)
	}

	results := audit(filesToProcess, policy)
	if len(results) == 0 {
		logrus.Info("No audit results generated, reports will be skipped.")
		return
	}

	outFile, err := os.Create(args.JSONOutfile)
	if err != nil {
		logrus.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(results); err != nil {
		logrus.Fatalf("Failed to encode JSON: %v", err)
	}

	logrus.Infof("JSON audit report generated: %s", args.JSONOutfile)
}

func getFilesToProcess(argsDockerfile string) []string {
	filesToProcess := make([]string, 0)
	fileInfo, err := os.Stat(argsDockerfile)
	if err != nil {
		logrus.Fatal(err)
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
	logrus.Infof("Scanning %d files in %s", len(filesToProcess), argsDockerfile)
	return filesToProcess
}

func getPolicy(policyFile string) (*dockerfile.Policy, error) {
	policy, err := dockerfile.NewDockerfilePolicy(policyFile)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	return policy, nil
}

func parse(filesToProcess []string) []dockerfile.ParseResult {
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

func audit(filesToProcess []string, policy *dockerfile.Policy) []dockerfile.PolicyResult {
	auditor := dockerfile.NewDockerfileAuditor(*policy)
	results := make([]dockerfile.PolicyResult, 0)
	for _, file := range filesToProcess {
		result, err := auditor.Audit(file)
		if err == nil {
			results = append(results, result)
			logrus.Infof("Scanning file: %s", file)
		}
	}
	return results
}
