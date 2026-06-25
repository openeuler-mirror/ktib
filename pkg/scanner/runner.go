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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	o "gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/scanner/dockerfile"
	"github.com/sirupsen/logrus"
)

type FileProcessFailure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type AuditRunSummary struct {
	FilesDiscovered int                  `json:"files_discovered"`
	FilesSucceeded  int                  `json:"files_succeeded"`
	FilesFailed     []FileProcessFailure `json:"files_failed,omitempty"`
	OutputFile      string               `json:"output_file,omitempty"`
	ParseOnly       bool                 `json:"parse_only"`
}

func RunDockerfileAudit(args o.Arguments) (*AuditRunSummary, error) {
	summary := &AuditRunSummary{
		ParseOnly: args.ParseOnly,
	}

	filesToProcess, err := getFilesToProcess(args.Dockerfile)
	if err != nil {
		return summary, err
	}
	summary.FilesDiscovered = len(filesToProcess)
	if len(filesToProcess) == 0 {
		logrus.Info("No Dockerfiles found to process")
		return summary, nil
	}

	if args.ParseOnly {
		parsedFiles, failures := parse(filesToProcess)
		summary.FilesSucceeded = len(parsedFiles)
		summary.FilesFailed = failures
		if len(parsedFiles) == 0 {
			logrus.Info("No files were processed, reports will be skipped.")
			return summary, aggregateFailures("parse dockerfiles", failures)
		}
		if err := writeJSONReport(args.JSONOutfile, parsedFiles); err != nil {
			return summary, err
		}
		summary.OutputFile = args.JSONOutfile
		logrus.Infof("JSON report generated: %s", args.JSONOutfile)
		return summary, aggregateFailures("parse dockerfiles", failures)
	}

	policy, err := getPolicy(args.PolicyFile)
	if err != nil {
		return summary, err
	}

	results, failures := audit(filesToProcess, policy)
	summary.FilesSucceeded = len(results)
	summary.FilesFailed = failures
	if len(results) == 0 {
		logrus.Info("No audit results generated, reports will be skipped.")
		return summary, aggregateFailures("audit dockerfiles", failures)
	}
	if err := writeJSONReport(args.JSONOutfile, results); err != nil {
		return summary, err
	}
	summary.OutputFile = args.JSONOutfile
	logrus.Infof("JSON audit report generated: %s", args.JSONOutfile)
	return summary, aggregateFailures("audit dockerfiles", failures)
}

func getFilesToProcess(argsDockerfile string) ([]string, error) {
	filesToProcess := make([]string, 0)
	fileInfo, err := os.Stat(argsDockerfile)
	if err != nil {
		return nil, fmt.Errorf("stat dockerfile path %q: %w", argsDockerfile, err)
	}
	if fileInfo.IsDir() {
		err := filepath.Walk(argsDockerfile, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info == nil {
				return nil
			}
			if !info.IsDir() {
				filesToProcess = append(filesToProcess, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk dockerfile directory %q: %w", argsDockerfile, err)
		}
	} else {
		filesToProcess = append(filesToProcess, argsDockerfile)
	}
	logrus.Infof("Scanning %d files in %s", len(filesToProcess), argsDockerfile)
	return filesToProcess, nil
}

func getPolicy(policyFile string) (*dockerfile.Policy, error) {
	policy, err := dockerfile.NewDockerfilePolicy(policyFile)
	if err != nil {
		return nil, fmt.Errorf("get policy %q: %w", policyFile, err)
	}
	return policy, nil
}

func parse(filesToProcess []string) ([]dockerfile.ParseResult, []FileProcessFailure) {
	parsedFiles := make([]dockerfile.ParseResult, 0)
	failures := make([]FileProcessFailure, 0)
	auditor := dockerfile.NewDockerfileAuditor(dockerfile.Policy{})
	for _, file := range filesToProcess {
		content, err := auditor.ParseOnly(file)
		if err != nil {
			logrus.Warnf("Skipping file %s during parse: %v", file, err)
			failures = append(failures, FileProcessFailure{
				Path:  file,
				Error: err.Error(),
			})
			continue
		}
		parsedFiles = append(parsedFiles, content)
	}
	return parsedFiles, failures
}

func audit(filesToProcess []string, policy *dockerfile.Policy) ([]dockerfile.PolicyResult, []FileProcessFailure) {
	auditor := dockerfile.NewDockerfileAuditor(*policy)
	results := make([]dockerfile.PolicyResult, 0)
	failures := make([]FileProcessFailure, 0)
	for _, file := range filesToProcess {
		result, err := auditor.Audit(file)
		if err != nil {
			logrus.Warnf("Skipping file %s during audit: %v", file, err)
			failures = append(failures, FileProcessFailure{
				Path:  file,
				Error: err.Error(),
			})
			continue
		}
		results = append(results, result)
		logrus.Infof("Scanning file: %s", file)
	}
	return results, failures
}

func writeJSONReport(outputFile string, content interface{}) error {
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", outputFile, err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(content); err != nil {
		return fmt.Errorf("encode JSON report %q: %w", outputFile, err)
	}
	return nil
}

func aggregateFailures(action string, failures []FileProcessFailure) error {
	if len(failures) == 0 {
		return nil
	}
	errs := make([]error, 0, len(failures))
	for _, failure := range failures {
		errs = append(errs, fmt.Errorf("%s: %s", failure.Path, failure.Error))
	}
	return fmt.Errorf("%s: %w", action, errors.Join(errs...))
}
