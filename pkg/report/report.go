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

package report

import (
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/scanner/dockerfile"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
)

type ViolationStats map[string]map[string]int

func GenerateLatexReport(policy dockerfile.Policy, policyResults []dockerfile.PolicyResult, tmpfile string, outfile string) error {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Println("Starting report generation.")

	failureStats := getRulesViolationStats(policyResults, policy)
	summaryStats := getSummaryStats(policyResults)
	enabledPolicyRules := map[string]interface{}{
		"policy_rules_enabled": policy.GetPolicyRulesEnabled(),
	}
	for _, enabledRule := range enabledPolicyRules["policy_rules_enabled"].([]dockerfile.Rule) {
		enabledRule.Details = latexEscape(enabledRule.Details)
	}
	for _, item := range policyResults {
		item.Filename = latexEscape(item.Filename)

		for _, rule := range item.Tests {
			//rule.Type = latexEscape(strconv.Itoa(rule.Type))
			rule.Details = latexEscape(rule.Details)
			rule.Mitigations = latexEscape(rule.Mitigations)
			rule.Statement = latexEscapeTiny(rule.Statement)
		}
	}

	auditResults := map[string]interface{}{
		"audit_results": policyResults,
	}
	latexJinjaEnv := template.New("latex_template").Delims("\\BLOCK{", "}")
	latexJinjaEnv = latexJinjaEnv.Delims("\\VAR{", "}")
	latexJinjaEnv = latexJinjaEnv.Delims("\\#{", "}")
	latexJinjaEnv = latexJinjaEnv.Delims("%%", "")
	latexJinjaEnv = latexJinjaEnv.Delims("%#", "")
	latexJinjaEnv = latexJinjaEnv.Option("missingkey=error")
	latexJinjaEnv = latexJinjaEnv.Option("missingkey=invalid")
	latexJinjaEnv = latexJinjaEnv.Funcs(template.FuncMap{
		"latexEscape":     latexEscape,
		"latexEscapeTiny": latexEscapeTiny,
	})

	templateFile, err := ioutil.ReadFile(tmpfile)
	if err != nil {
		logger.Println("Failed to read template file:", err)
		return err
	}

	_, err = latexJinjaEnv.Parse(string(templateFile))
	if err != nil {
		logger.Println("Failed to parse template:", err)
		return err
	}

	buildDir := ".build"
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		err := os.Mkdir(buildDir, 0755)
		if err != nil {
			logger.Println("Failed to create build directory:", err)
			return err
		}
	}

	outFile := buildDir + "/template.tex"
	renderedTemplate, err := os.Create(outFile)
	if err != nil {
		logger.Println("Failed to create rendered template file:", err)
		return err
	}

	err = latexJinjaEnv.Execute(renderedTemplate, map[string]interface{}{
		"summary_stats":        summaryStats,
		"failure_stats":        failureStats,
		"enabled_policy_rules": enabledPolicyRules,
		"audit_results":        auditResults,
	})
	if err != nil {
		logger.Println("Failed to render template:", err)
		return err
	}

	err = copyFiles("templates/images", buildDir)
	if err != nil {
		logger.Println("Failed to copy image files:", err)
	}

	logger.Println("Running first Latex build of the report.")
	cmd := exec.Command("pdflatex", "-output-directory", buildDir, outFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		logger.Println("The first Pdflatex iteration failed:", err)
		return err
	}

	logger.Println("Rebuilding report for Table of Contents.")
	err = cmd.Run()
	if err != nil {
		logger.Println("The second Pdflatex iteration failed:", err)
		return err
	}

	err = copyFile(outFile+".pdf", outfile)
	if err != nil {
		logger.Println("Failed to copy the generated report:", err)
		return err
	}

	logger.Println("Report generated:", outfile)
	return nil
}

func getSummaryStats(policyResults []dockerfile.PolicyResult) options.SummaryStats {
	totalTests := len(policyResults)
	successfulTests := 0
	failedTests := 0

	for _, test := range policyResults {
		if test.AuditOutcome == "pass" {
			successfulTests++
		} else {
			failedTests++
		}
	}

	successPercentage := float64(successfulTests) * 100 / float64(totalTests)
	failedPercentage := float64(failedTests) * 100 / float64(totalTests)

	complianceLevel := "N/A"
	complianceColor := "red"

	if successPercentage < 10 {
		complianceLevel = "Poor"
		complianceColor = "red!50"
	} else if successPercentage < 25 {
		complianceLevel = "Low"
		complianceColor = "red!30"
	} else if successPercentage < 50 {
		complianceLevel = "Medium"
		complianceColor = "orange!50"
	} else if successPercentage < 80 {
		complianceLevel = "Fair"
		complianceColor = "green!20"
	} else if successPercentage < 100 {
		complianceLevel = "Good"
		complianceColor = "green!35"
	} else if successPercentage == 100 {
		complianceLevel = "Perfect"
		complianceColor = "green!50"
	}

	return options.SummaryStats{
		TotalTests:        totalTests,
		SuccessTests:      successfulTests,
		FailedTests:       failedTests,
		SuccessPercentage: formatFloat(successPercentage, 2),
		FailedPercentage:  formatFloat(failedPercentage, 2),
		ComplianceLevel:   complianceLevel,
		ComplianceColor:   complianceColor,
	}
}

func getRulesViolationStats(policyResults []dockerfile.PolicyResult, policy dockerfile.Policy) ViolationStats {
	total := 0
	violationStats := make(ViolationStats)
	var ruleType string
	for _, rule := range policy.GetPolicyRulesEnabled() {
		ruleType = rule.Type.String()
		violationStats[latexEscape(ruleType)] = map[string]int{"count": 0}
	}

	for _, test := range policyResults {
		if test.AuditOutcome == "fail" {
			for _, test := range test.Tests {
				// todo : panic here
				if _, ok := violationStats[ruleType]; ok {
					violationStats[latexEscape(test.Type.String())]["count"]++
					total++
				}
			}
		}
	}

	if total == 0 {
		for key := range violationStats {
			violationStats[key]["percentage"] = 0
		}
	} else {
		for key := range violationStats {
			violationStats[key]["percentage"] = violationStats[key]["count"] * 100 / total
		}
	}
	return violationStats
}

func latexEscape(str string) string {
	str = strings.ReplaceAll(str, "_", "\\_")
	str = strings.ReplaceAll(str, "$", "\\$")
	var parts []string
	for i := 0; i < len(str); i += 25 {
		end := i + 25
		if end > len(str) {
			end = len(str)
		}
		parts = append(parts, str[i:end])
	}
	return strings.Join(parts, "\\allowbreak ")
}

func latexEscapeTiny(slice []string) []string {
	var escape []string
	for _, op := range slice {
		escape = append(escape, latexEscape(op))
	}
	return escape
}

func copyFiles(srcDir, dstDir string) error {
	cmd := exec.Command("cp", "-r", srcDir, dstDir)
	return cmd.Run()
}

func copyFile(srcFile, dstFile string) error {
	input, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dstFile, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

func formatFloat(f float64, precision int) string {
	return strconv.FormatFloat(f, 'f', precision, 64)
}
