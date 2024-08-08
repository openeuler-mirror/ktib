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

package dockerfile

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type PolicyRuleType int

const (
	GENERIC_POLICY PolicyRuleType = iota + 1
	ENFORCE_REGISTRY
	FORBID_TAGS
	FORBID_INSECURE_REGISTRIES
	FORBID_ROOT
	FORBID_PRIVILEGED_PORTS
	FORBID_PACKAGES
	FORBID_SECRETS
	FORBID_LAX_CHMOD
)

type Rule struct {
	Type        PolicyRuleType
	Details     string
	Mitigations string
	Statement   []string
}

type PolicyTestResult struct {
	Results []Rule
}

type GenericPolicyRule struct {
	Type        PolicyRuleType
	TestResult  PolicyTestResult
	Description string
}

type PolicyRule interface {
	GetType() PolicyRuleType
	Details() string
	Describe() string
	Test(directives map[string][]DfDirective) *[]Rule
}

func (r *PolicyTestResult) GetResult() *[]Rule {
	if len(r.Results) > 0 {
		return &r.Results
	}
	return nil
}

func (r *PolicyTestResult) AddResult(details, mitigations string, ruleType PolicyRuleType, content string) {
	result := Rule{
		Details:     details,
		Mitigations: mitigations,
		Type:        ruleType,
	}
	if content != "" {
		result.Statement = []string{content}
	}
	r.Results = append(r.Results, result)
}

func (t PolicyRuleType) String() string {
	names := []string{
		"GENERIC_POLICY",
		"ENFORCE_REGISTRY",
		"FORBID_TAGS",
		"FORBID_INSECURE_REGISTRIES",
		"FORBID_ROOT",
		"FORBID_PRIVILEGED_PORTS",
		"FORBID_PACKAGES",
		"FORBID_SECRETS",
		"FORBID_LAX_CHMOD",
	}
	if t < GENERIC_POLICY || t > FORBID_LAX_CHMOD {
		log.Fatalf("Invalid PolicyRuleType: %d", t)
	}
	return names[t-1]
}

func (r *GenericPolicyRule) Describe() string {
	return r.Description
}

func (r *GenericPolicyRule) GetType() PolicyRuleType {
	return r.Type
}

func (r *GenericPolicyRule) Details() string {
	return ""
}

func (r *GenericPolicyRule) Test(directives map[string][]DfDirective) *[]Rule {
	return nil
}

type EnforceRegistryPolicy struct {
	GenericPolicyRule
	AllowedRegistries []string
	Enabled           bool
}

func NewEnforceRegistryPolicy(allowedRegistries []string, enabled bool) *EnforceRegistryPolicy {
	return &EnforceRegistryPolicy{
		GenericPolicyRule: GenericPolicyRule{
			Type:        ENFORCE_REGISTRY,
			TestResult:  PolicyTestResult{},
			Description: "Allow images to be based (using the FROM command) only on images belonging to approved repositories.",
		},
		AllowedRegistries: allowedRegistries,
		Enabled:           enabled,
	}
}

func (r *EnforceRegistryPolicy) Test(dockerfileDirectives map[string][]DfDirective) *[]Rule {
	r.TestResult = NewPolicyTestResult()
	fromStatements := dockerfileDirectives["from"]
	for _, statement := range fromStatements {
		var statementInterface interface{} = statement
		if fromDirective, ok := statementInterface.(*FromDirective); ok {
			if fromDirective.ImageName == "scratch" {
				continue
			}
			registry := fromDirective.Registry
			isFromLocalImage := false
			for _, s := range fromStatements {
				var sInterface interface{} = s
				if sDirective, ok := sInterface.(*FromDirective); ok {
					if fromDirective.ImageLocalName == sDirective.ImageName {
						isFromLocalImage = true
						break
					}
				}
			}
			if !isFromLocalImage {
				found := false
				for _, allowedRegistry := range r.AllowedRegistries {
					if registry == allowedRegistry {
						found = true
						break
					}
				}
				if !found {
					r.TestResult.AddResult(
						"Registry "+registry+" 不是允许拉取镜像的注册表。",
						"应该更改 FROM 语句，使用允许的注册表之一的镜像："+
							strings.Join(r.AllowedRegistries, ", "),
						r.Type,
						fromDirective.Content,
					)
				}
			}
		}
	}
	return r.TestResult.GetResult()
}

func (r *EnforceRegistryPolicy) Details() string {
	return "The following registries are allowed: " + strings.Join(r.AllowedRegistries, ", ") + "."
}

type ForbidTags struct {
	GenericPolicyRule
	ForbiddenTags []string
}

func NewForbidTags(tags []string) *ForbidTags {
	return &ForbidTags{
		GenericPolicyRule: GenericPolicyRule{
			Type:        FORBID_TAGS,
			TestResult:  PolicyTestResult{},
			Description: "Restrict the use of certain tags for the images the build is sourced from (using FROM command)",
		},
		ForbiddenTags: tags,
	}
}

func (r *ForbidTags) Test(directives map[string][]DfDirective) *[]Rule {
	fromStatements := directives["from"]
	for _, statement := range fromStatements {
		var fromDirectiveInterface interface{} = statement
		if fromDirective, ok := fromDirectiveInterface.(*FromDirective); ok {
			image := fromDirective.ImageName
			if image == "scratch" {
				return nil
			}
			tag := fromDirective.ImageTag
			if contains(r.ForbiddenTags, tag) {
				r.TestResult.AddResult(fmt.Sprintf("Tag %s is not allowed.", tag),
					fmt.Sprintf("The FROM statements should be changed using an image with a fixed tag or without any of the following tags: %s",
						strings.Join(r.ForbiddenTags, ", ")),
					r.Type, fromDirective.Content)
			} else {
				return nil
			}
		}
	}
	return r.TestResult.GetResult()
}

func (rule *ForbidTags) Details() string {
	return fmt.Sprintf("The following tags are forbidden: %s.", strings.Join(rule.ForbiddenTags, ", "))
}

type ForbidInsecureRegistries struct {
	GenericPolicyRule
	Enabled bool
}

func NewForbidInsecureRegistries(enabled bool) *ForbidInsecureRegistries {
	return &ForbidInsecureRegistries{
		GenericPolicyRule: GenericPolicyRule{
			Type:        FORBID_INSECURE_REGISTRIES,
			TestResult:  PolicyTestResult{},
			Description: "Forbid the use of HTTP protocol for the registries from which source images are stored.",
		},
		Enabled: enabled,
	}
}

func (rule *ForbidInsecureRegistries) Test(dockerfileStatements map[string][]DfDirective) *[]Rule {
	testResult := NewPolicyTestResult()
	fromStatements := dockerfileStatements["from"]
	for _, statement := range fromStatements {
		var statementInterface interface{} = statement
		if fromDirective, ok := statementInterface.(*FromDirective); ok {
			registry := fromDirective.Registry
			if strings.HasPrefix(registry, "http://") {
				testResult.AddResult(fmt.Sprintf("Registry %s uses HTTP and therefore it is considered insecure", registry),
					"The FROM statement should be changed using images from a registry which uses HTTPS.",
					rule.Type, fromDirective.Content)
			}
		}
	}
	return testResult.GetResult()
}

func NewPolicyTestResult() PolicyTestResult {
	return PolicyTestResult{}
}

type ForbidRoot struct {
	GenericPolicyRule
	Enabled bool
}

func NewForbidRoot(enabled bool) *ForbidRoot {
	return &ForbidRoot{
		GenericPolicyRule: GenericPolicyRule{
			Description: "Forbid the container to run as a privileged (root) user.",
			Type:        FORBID_ROOT,
		},
		Enabled: enabled,
	}
}

func (rule *ForbidRoot) Test(dockerfileStatements map[string][]DfDirective) *[]Rule {
	testResult := NewPolicyTestResult()
	userStatements := dockerfileStatements["user"]
	if len(userStatements) == 0 {
		testResult.AddResult("No USER statements found. By default, if privileges are not dropped, the container will run as root.",
			"Create a user and add a USER statement before the entrypoint of the image to run the application as a non-privileged user.",
			rule.Type, "")
	} else {
		lastUserStatement := userStatements[len(userStatements)-1]
		var lastUserStatementInterface interface{} = lastUserStatement
		if userDirective, ok := lastUserStatementInterface.(*UserDirective); ok {
			lastUser := userDirective.User
			if lastUser == "0" || lastUser == "root" {
				testResult.AddResult("The last USER statement found elevates privileged to root.",
					"Add one more USER statement before the entrypoint of the image to run the application as a non-privileged user.",
					rule.Type, userDirective.Content)
			}
		}
	}
	return testResult.GetResult()
}

type ForbidPrivilegedPorts struct {
	GenericPolicyRule
	Enabled bool
}

func NewForbidPrivilegedPorts(enabled bool) *ForbidPrivilegedPorts {
	return &ForbidPrivilegedPorts{
		GenericPolicyRule: GenericPolicyRule{
			Description: "Forbid the image to expose privileged ports that require administrative permissions.",
			Type:        FORBID_PRIVILEGED_PORTS,
		},
		Enabled: enabled,
	}
}

func (rule *ForbidPrivilegedPorts) Test(dockerfileDirective map[string][]DfDirective) *[]Rule {
	testResult := NewPolicyTestResult()
	exposeStatements := dockerfileDirective["expose"]
	for _, statement := range exposeStatements {
		var exposeStatementInterface interface{} = statement
		if exposeDirective, ok := exposeStatementInterface.(*ExposeDirective); ok {
			ports := exposeDirective.Ports
			for _, port := range ports {
				portNum, err := strconv.Atoi(port)
				if err == nil && portNum <= 1024 {
					testResult.AddResult(fmt.Sprintf("The container exposes a privileged port: %s. Privileged ports require the application which uses it to run as root.", port),
						"Change the configuration for the application to bind on a port greater than 1024, and change the Dockerfile to reflect this modification.",
						rule.Type, exposeDirective.Content)
				} else {
					portNumber := rule.getPortFromEnv(port, dockerfileDirective)
					if portNumber != nil && *portNumber <= 1024 {
						testResult.AddResult(fmt.Sprintf("The container exposes a privileged port: %d. Privileged ports require the application which uses it to run as root.", *portNumber),
							"Change the configuration for the application to bind on a port greater than 1024, and change the Dockerfile to reflect this modification.",
							rule.Type, exposeDirective.Content)
					}
				}
			}
		}
	}
	return testResult.GetResult()
}

func (rule *ForbidPrivilegedPorts) getPortFromEnv(envName string, dockerfileStatements map[string][]DfDirective) *int {
	envStatements := dockerfileStatements["env"]
	for _, statement := range envStatements {
		var envDirectiveInterface interface{} = statement
		if envDirective, ok := envDirectiveInterface.(*EnvDirective); ok {
			variables := envDirective.Variables
			if portNumberStr, exist := variables[envName]; exist {
				portNumber, err := strconv.Atoi(portNumberStr)
				if err == nil {
					return &portNumber
				}
			}
		}
	}
	return nil
}

type ForbidPackages struct {
	ForbiddenPackages []string
	GenericPolicyRule
}

func NewForbidPackages(forbiddenPackages []string) *ForbidPackages {
	return &ForbidPackages{
		ForbiddenPackages: forbiddenPackages,
		GenericPolicyRule: GenericPolicyRule{
			Description: "Forbid the installation/use of dangerous packages.",
			Type:        FORBID_PACKAGES,
		},
	}
}

func (rule *ForbidPackages) Test(mapDirectives map[string][]DfDirective) *[]Rule {
	testResult := NewPolicyTestResult()
	runDirectives := mapDirectives["run"]
	entrypointDirectives := mapDirectives["entrypoint"]
	cmdDirectives := mapDirectives["cmd"]
	commands := rule.splitSingleCommands(mapDirectives["run_last_stage"])
	installedPackages := rule.getInstalledPackages(commands)
	if len(installedPackages) == 0 {
		for _, statement := range append(entrypointDirectives, append(runDirectives, cmdDirectives...)...) {
			for _, pkg := range rule.ForbiddenPackages {
				packageRegex := regexp.MustCompile(fmt.Sprintf(`(^|[^a-zA-Z0-9])%s([^a-zA-Z0-9]|$)`, pkg))
				match := packageRegex.MatchString(statement.Get()["raw_content"].(string))
				if match {
					testResult.AddResult(fmt.Sprintf("Forbidden package \"%s\" is installed or used.", pkg),
						fmt.Sprintf("The RUN/CMD/ENTRYPOINT statement should be reviewed and package \"%s\" should be removed unless absolutely necessary.", pkg),
						rule.Type, statement.Get()["raw_content"].(string))
				}
			}
		}
	} else {
		for _, pkg := range rule.ForbiddenPackages {
			if contains(installedPackages, pkg) {
				testResult.AddResult(fmt.Sprintf("Forbidden package \"%s\" is installed.", pkg),
					fmt.Sprintf("The RUN statements should be reviewed and package \"%s\" should be removed unless absolutely necessary.", pkg),
					rule.Type, "")
			}
		}
	}
	return testResult.GetResult()
}

func (rule *ForbidPackages) getInstalledPackages(commands [][]string) []string {
	packageManagerCommands := map[string]map[string][]string{
		"apt-get": {"install": {"install"}, "remove": {"remove", "purge"}},
		"apt":     {"install": {"install"}, "remove": {"remove", "purge"}},
		"dnf":     {"install": {"install"}, "remove": {"remove", "autoremove"}},
		"yum":     {"install": {"install"}, "remove": {"remove", "erase", "autoremove"}},
		"apk":     {"install": {"add"}, "remove": {"del"}},
	}
	flagRegex := regexp.MustCompile("^[-]{1,2}[\\S]+$")
	installedPackages := make([]string, 0)
	removedPackages := make([]string, 0)

	for _, command := range commands {
		for i := 0; i < len(command); i++ {
			if _, ok := packageManagerCommands[command[i]]; !ok {
				continue
			}
			key := command[i]
			for k := 0; k < len(command[i+1:]); k++ {
				nextCommand := command[i+1+k]
				if flagRegex.MatchString(nextCommand) {
					continue
				} else if contains(packageManagerCommands[key]["install"], nextCommand) {
					installedPackages = append(installedPackages, command[i+1+k+1:]...)
					break
				} else if contains(packageManagerCommands[key]["remove"], nextCommand) {
					removedPackages = append(removedPackages, command[i+1+k+1:]...)
					break
				}
			}
			break
		}
	}

	finalPackages := make([]string, 0)
	for _, p := range installedPackages {
		if !contains(removedPackages, p) && !flagRegex.MatchString(p) {
			finalPackages = append(finalPackages, p)
		}
	}
	return finalPackages
}

func (rule *ForbidPackages) splitSingleCommands(runDirectives []DfDirective) [][]string {
	commands := make([][]string, 0)
	for _, directive := range runDirectives {
		subcommand := make([]string, 0)
		parsed := strings.Fields(directive.Get()["raw_content"].(string))
		for _, word := range parsed {
			if contains([]string{"&", "&&", "|", "||", ";"}, word) {
				commands = append(commands, subcommand)
				subcommand = make([]string, 0)
			} else {
				subcommand = append(subcommand, word)
			}
		}
		commands = append(commands, subcommand)
	}
	return commands
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func (rule *ForbidPackages) Details() string {
	return fmt.Sprintf("The following packages are forbidden: %s.", strings.Join(rule.ForbiddenPackages, ", "))
}

type ForbidSecrets struct {
	GenericPolicyRule
	secretsPatterns []string
	allowedPatterns []string
}

func NewForbidSecrets(secretsPatterns, allowedPatterns []string) *ForbidSecrets {
	return &ForbidSecrets{
		GenericPolicyRule: GenericPolicyRule{
			Description: "Forbid the inclusion of secrets in the image.",
			Type:        FORBID_SECRETS,
		},
		secretsPatterns: secretsPatterns,
		allowedPatterns: allowedPatterns,
	}
}

func (fs *ForbidSecrets) Test(dockerfileStatements map[string][]DfDirective) *[]Rule {
	fs.TestResult = PolicyTestResult{}
	addStatements := dockerfileStatements["add"]
	copyStatements := dockerfileStatements["copy"]
	for _, statement := range append(addStatements, copyStatements...) {
		switch statement.GetType() {
		case ADD:
			addDirective := statement.Get()
			sources := addDirective["source"]
			isForbidden, pattern := fs.isForbiddenPattern(sources.(string))
			if isForbidden && !fs.isWhitelistedPattern(sources.(string)) {
				fs.TestResult.AddResult(
					fmt.Sprintf("Forbidden file matching pattern \"%s\" is added into the image.", pattern),
					"The ADD statement should be changed or removed. Secrets should be provisioned using a safer and stateless way (Vault, Kubernetes secrets) instead.",
					fs.Type,
					statement.Get()["raw_content"].(string),
				)
			}
		case COPY:
			copyDirective := statement.Get()
			sources := copyDirective["source"]
			isForbidden, pattern := fs.isForbiddenPattern(sources.(string))
			if isForbidden && !fs.isWhitelistedPattern(sources.(string)) {
				fs.TestResult.AddResult(
					fmt.Sprintf("Forbidden file matching pattern \"%s\" is added into the image.", pattern),
					"The COPY statement should be changed or removed. Secrets should be provisioned using a safer and stateless way (Vault, Kubernetes secrets) instead.",
					fs.Type,
					statement.Get()["raw_content"].(string),
				)
			}
		}
	}
	return fs.TestResult.GetResult()
}

func (fs *ForbidSecrets) isForbiddenPattern(source string) (bool, string) {
	for _, pattern := range fs.secretsPatterns {
		secretRegex := regexp.MustCompile(pattern)
		matchSource := secretRegex.MatchString(source)
		if matchSource {
			return true, pattern
		}
	}
	return false, ""
}

func (fs *ForbidSecrets) isWhitelistedPattern(source string) bool {
	for _, pattern := range fs.allowedPatterns {
		allowedRegex := regexp.MustCompile(pattern)
		matchAllowed := allowedRegex.MatchString(source)
		if matchAllowed {
			return true
		}
	}
	return false
}

func (fs *ForbidSecrets) Details() string {
	return fmt.Sprintf("The following patterns are forbidden: %s.\nThe following patterns are whitelisted: %s",
		joinStrings(fs.secretsPatterns), joinStrings(fs.allowedPatterns))
}

func joinStrings(strSlice []string) string {
	return strings.Join(strSlice, ", ")
}
