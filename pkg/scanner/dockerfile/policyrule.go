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
	"bytes"
	"encoding/json"
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
	Type        PolicyRuleType `json:"Type"`
	Details     string         `json:"Details"`
	Mitigations string         `json:"Mitigations"`
	Statement   []string       `json:"Statement,omitempty"`
	Line        int            `json:"Line,omitempty"`
	Directive   string         `json:"Directive,omitempty"`
	Level       string         `json:"Level,omitempty"`
	Status      string         `json:"Status"` // 新增："pass" 或 "fail"
}

// 添加MarshalJSON方法以自定义JSON序列化
func (r Rule) MarshalJSON() ([]byte, error) {
	type Alias Rule
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false) // 禁用HTML转义，保持特殊字符原样
	err := encoder.Encode(&struct {
		Type string `json:"Type"`
		Alias
	}{
		Type:  r.Type.String(),
		Alias: Alias(r),
	})
	if err != nil {
		return nil, err
	}
	// 移除encoder添加的换行符
	bytes := buf.Bytes()
	return bytes[:len(bytes)-1], nil
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
	// 无论是否有结果都返回，包括合规项
	if len(r.Results) > 0 {
		return &r.Results
	}
	return &[]Rule{} // 返回空切片而不是nil
}

func (r *PolicyTestResult) AddResult(details, mitigations string, ruleType PolicyRuleType, content string, directiveType ...string) {
	result := Rule{
		Details:     details,
		Mitigations: mitigations,
		Type:        ruleType,
		Status:      "fail",
	}
	if content != "" {
		result.Statement = []string{content}
	}
	r.Results = append(r.Results, result)
}

func (r *PolicyTestResult) AddPassResult(details string, ruleType PolicyRuleType, content string) {
	result := Rule{
		Details:     details,
		Mitigations: "", // 合规项不需要mitigations
		Type:        ruleType,
		Status:      "pass",
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
			Description: "仅允许使用已批准仓库中的镜像构建镜像。（使用 FROM 命令）",
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
					// 不合规项
					r.TestResult.AddResult(
						"Registry "+registry+" 不是允许拉取镜像的注册表。",
						"应该更改 FROM 语句，使用来源于允许的镜像仓库注册表的镜像："+
							strings.Join(r.AllowedRegistries, ", "),
						r.Type,
						fromDirective.Content,
					)
				} else {
					// 新增：合规项
					r.TestResult.AddPassResult(
						"Registry "+registry+" 是允许的镜像仓库注册表。",
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
	return "允许使用的镜像仓库: " + strings.Join(r.AllowedRegistries, ", ") + "。"
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
			Description: "限制使用某些标签作为构建的基础镜像（使用 FROM 命令）",
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
				r.TestResult.AddResult(fmt.Sprintf("标签 %s 不允许使用。", tag),
					fmt.Sprintf("FROM 语句应该更改为使用具有固定标签的镜像，或者不使用以下任何标签: %s",
						strings.Join(r.ForbiddenTags, ", ")),
					r.Type, fromDirective.Content)
			} else {
				r.TestResult.AddPassResult(
					fmt.Sprintf("允许使用 %s 标签。", tag),
					r.Type,
					fromDirective.Content)
			}
		}
	}
	return r.TestResult.GetResult()
}

func (rule *ForbidTags) Details() string {
	return fmt.Sprintf("以下标签被禁止使用: %s。", strings.Join(rule.ForbiddenTags, ", "))
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
			Description: "禁止使用不安全的镜像仓库。",
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
				testResult.AddResult(fmt.Sprintf("镜像仓库 %s 被视为不安全", registry),
					"FROM 语句应该更改为使用 HTTPS 协议的镜像仓库中的镜像。",
					rule.Type, fromDirective.Content)
			} else {
				testResult.AddPassResult(
					fmt.Sprintf("镜像仓库 %s 被视为安全", registry),
					rule.Type,
					fromDirective.Content)
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
			Description: "禁止容器以特权用户（root）身份运行。",
			Type:        FORBID_ROOT,
		},
		Enabled: enabled,
	}
}

func (rule *ForbidRoot) Test(dockerfileStatements map[string][]DfDirective) *[]Rule {
	testResult := NewPolicyTestResult()
	userStatements := dockerfileStatements["user"]
	if len(userStatements) == 0 {
		testResult.AddResult("未找到 USER 指令。默认情况下，如果不降低权限，容器将以 root 用户身份运行。",
			"创建一个用户并在镜像的入口点之前添加 USER 指令，以非特权用户身份运行应用程序。",
			rule.Type, "")
	} else {
		lastUserStatement := userStatements[len(userStatements)-1]
		var lastUserStatementInterface interface{} = lastUserStatement
		if userDirective, ok := lastUserStatementInterface.(*UserDirective); ok {
			lastUser := userDirective.User
			if lastUser == "0" || lastUser == "root" {
				testResult.AddResult("最后一个 USER 指令将权限提升为 root。",
					"在镜像的入口点之前添加另一个 USER 指令，以非特权用户身份运行应用程序。",
					rule.Type, userDirective.Content)
			} else {
				testResult.AddPassResult(
					"最后一个 USER 指令指定用户非特权用户。",
					rule.Type,
					userDirective.Content)
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
			Description: "禁止镜像暴露需要管理员权限的特权端口。",
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
				if err == nil {
					// 端口号解析成功
					if portNum <= 1024 {
						// 特权端口 - 不合规
						testResult.AddResult(
							fmt.Sprintf("容器暴露了特权端口: %s。特权端口要求使用它的应用程序以 root 身份运行。", port),
							"更改应用程序的配置，使其绑定到大于 1024 的端口，并更改 Dockerfile 以反映此修改。",
							rule.Type,
							exposeDirective.Content)
					} else {
						// 非特权端口 - 合规
						testResult.AddPassResult(
							fmt.Sprintf("容器没有暴露特权端口，仅暴露了非特权端口: %s，符合安全要求。", port),
							rule.Type,
							exposeDirective.Content)
					}
				} else {
					// 端口号解析失败，可能是环境变量，尝试从环境变量获取
					portNumber := rule.getPortFromEnv(port, dockerfileDirective)
					if portNumber != nil {
						if *portNumber <= 1024 {
							// 从环境变量解析出的特权端口 - 不合规
							testResult.AddResult(
								fmt.Sprintf("容器暴露了特权端口: %d。特权端口要求使用它的应用程序以 root 身份运行。", *portNumber),
								"更改应用程序的配置，使其绑定到大于 1024 的端口，并更改 Dockerfile 以反映此修改。",
								rule.Type,
								exposeDirective.Content)
						} else {
							// 从环境变量解析出的非特权端口 - 合规
							testResult.AddPassResult(
								fmt.Sprintf("容器没有暴露特权端口，仅暴露了非特权端口: %d，符合安全要求。", *portNumber),
								rule.Type,
								exposeDirective.Content)
						}
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
			Description: "禁止安装/使用危险的软件包。",
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
					testResult.AddResult(fmt.Sprintf("禁止的软件包 \"%s\" 被安装或使用。", pkg),
						fmt.Sprintf("应该审查 RUN/CMD/ENTRYPOINT 指令并移除软件包 \"%s\"，除非绝对必要。", pkg),
						rule.Type, statement.Get()["raw_content"].(string))
				}
			}
		}
	} else {
		for _, pkg := range rule.ForbiddenPackages {
			if contains(installedPackages, pkg) {
				testResult.AddResult(fmt.Sprintf("禁止的软件包 \"%s\" 已安装。", pkg),
					fmt.Sprintf("应该审查 RUN 指令并移除软件包 \"%s\"，除非绝对必要。", pkg),
					rule.Type, "")
			} else {
				testResult.AddPassResult(fmt.Sprintf("禁止的软件包 \"%s\" 未安装。", pkg),
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
	return fmt.Sprintf("以下软件包被禁止使用: %s。", strings.Join(rule.ForbiddenPackages, ", "))
}

type ForbidSecrets struct {
	GenericPolicyRule
	secretsPatterns []string
	allowedPatterns []string
}

func NewForbidSecrets(secretsPatterns, allowedPatterns []string) *ForbidSecrets {
	return &ForbidSecrets{
		GenericPolicyRule: GenericPolicyRule{
			Description: "禁止在镜像中包含敏感信息。",
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
					fmt.Sprintf("符合模式 \"%s\" 的禁止文件被添加到镜像中。", pattern),
					"应该更改或移除 ADD 指令。应该使用更安全和无状态的方式（如 Vault、Kubernetes secrets）来提供敏感信息。",
					fs.Type,
					statement.Get()["raw_content"].(string),
				)
			} else {
				fs.TestResult.AddPassResult(
					"没有识别到敏感信息。",
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
					fmt.Sprintf("符合模式 \"%s\" 的禁止文件被添加到镜像中。", pattern),
					"应该更改或移除 COPY 指令。应该使用更安全和无状态的方式（如 Vault、Kubernetes secrets）来提供敏感信息。",
					fs.Type,
					statement.Get()["raw_content"].(string),
				)
			} else {
				fs.TestResult.AddPassResult(
					"没有识别到敏感信息。",
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
	return fmt.Sprintf("以下模式被禁止: %s。\n以下模式被允许: %s",
		joinStrings(fs.secretsPatterns), joinStrings(fs.allowedPatterns))
}

func joinStrings(strSlice []string) string {
	return strings.Join(strSlice, ", ")
}
