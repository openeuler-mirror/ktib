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
		logrus.Fatal("failed to get policy")
	}
	results := audit(filesToProcess, policy)
	if len(results) == 0 {
		logrus.Info("No files were processed, reports will be skipped.")
		return
	}
	outFile, err := os.Create(args.JSONOutfile)
	if err != nil {
		logrus.Fatal(err)
	}
	defer outFile.Close()
	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(results); err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("JSON report generated: %s", args.JSONOutfile)
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
