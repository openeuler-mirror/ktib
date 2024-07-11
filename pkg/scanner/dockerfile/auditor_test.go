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
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"testing"
)

func TestAudit(t *testing.T) {
	// 创建一个临时目录,并在其中生成测试用的 Dockerfile
	DockerfilePath := "testdata/Dockerfile"
	DockerfileContent := `FROM ubuntu:latest
RUN apt-get update 
`
	err := os.MkdirAll("testdata", 0755)
	if err != nil {
		t.Errorf("创建 testdata 目录时出错: %v", err)
	}
	err = os.WriteFile(DockerfilePath, []byte(DockerfileContent), 0644)
	if err != nil {
		t.Errorf("创建有效 Dockerfile 文件时出错: %v", err)
	}
	defer os.Remove(DockerfilePath)

	// 准备测试数据
	expectedPolicyResult := PolicyResult{
		Filename:     "Dockerfile",
		Tests:        []Rule{},
		AuditOutcome: "fail",
		Maintainers:  "",
		Path:         "testdata/Dockerfile",
	}
	mockPolicyRules := []PolicyRule{
		&MockPolicyRule{
			GetTypeFunc:     func() PolicyRuleType { return GENERIC_POLICY },
			DetailsFunc:     func() string { return "Details for Rule 1" },
			DescribeFunc:    func() string { return "Description for Rule 1" },
			MitigationsFunc: func() string { return "Mitigations for Rule 1" },
			StatementFunc:   func() []string { return []string{"Statement 1", "Statement 2"} },
			TestFunc: func(directives map[string][]DfDirective) *[]Rule {
				return &expectedPolicyResult.Tests
			},
		},
	}
	mockPolicy := &Policy{
		PolicyRules: mockPolicyRules,
		PolicyFile:  "testdata/policy.yml",
	}
	auditor := &DockerfileAuditor{
		Policy: *mockPolicy,
	}

	// 调用 Audit 函数
	result, err := auditor.Audit("testdata/Dockerfile")

	// 断言结果
	require.NoError(t, err)
	reflect.DeepEqual(expectedPolicyResult, result)
}

func TestParseOnly(t *testing.T) {
	// 创建测试用的有效 Dockerfile
	validDockerfilePath := "testdata/valid_dockerfile"
	validDockerfileContent := `FROM ubuntu:latest
LABEL maintainer1="John Doe"
RUN apt-get update && apt-get install -y curl
`
	err := os.MkdirAll("testdata", 0755)
	if err != nil {
		t.Errorf("创建 testdata 目录时出错: %v", err)
	}
	err = os.MkdirAll("testdata", 0755)
	if err != nil {
		t.Errorf("创建 testdata 目录时出错: %v", err)
		return
	}
	err = os.WriteFile(validDockerfilePath, []byte(validDockerfileContent), 0644)
	if err != nil {
		t.Errorf("创建有效 Dockerfile 文件时出错: %v", err)
		return
	}
	defer os.Remove(validDockerfilePath)

	auditor := NewDockerfileAuditor(Policy{})
	result, err := auditor.ParseOnly(validDockerfilePath)
	if err != nil {
		t.Errorf("ParseOnly() 返回了错误: %v", err)
	}
	if result.Filename != "valid_dockerfile" {
		t.Errorf("预期文件名为 'valid_dockerfile', 实际为 %s", result.Filename)
	}

	if result.Path != validDockerfilePath {
		t.Errorf("预期路径为 '%s', 实际为 %s", validDockerfilePath, result.Path)
	}

	if result.Maintainers != "John Doe" {
		t.Errorf("预期维护者为 'John Doe', 实际为 %s", result.Maintainers)
	}

	if len(result.Directives) != 4 {
		t.Errorf("预期指令数为 4, 实际为 %d", len(result.Directives))
	}
}

func TestNewDockerfileAuditor(t *testing.T) {
	// 创建一个测试用的 Policy 实例
	testPolicy := Policy{
		PolicyRules: []PolicyRule{
			&MockPolicyRule{},
		},
		PolicyFile: "test_policy.yml",
	}

	// 创建 DockerfileAuditor 实例
	auditor := NewDockerfileAuditor(testPolicy)

	// 检查创建后的 DockerfileAuditor 实例是否包含了正确的 Policy 实例
	if !reflect.DeepEqual(auditor.Policy, testPolicy) {
		t.Errorf("NewDockerfileAuditor() did not return the expected Policy instance")
	}
}

// MockPolicyRule 是一个实现了 PolicyRule 接口的模拟实现
type MockPolicyRule struct {
	GetTypeFunc     func() PolicyRuleType
	DetailsFunc     func() string
	DescribeFunc    func() string
	MitigationsFunc func() string
	StatementFunc   func() []string
	TestFunc        func(directives map[string][]DfDirective) *[]Rule
}

func (r *MockPolicyRule) GetType() PolicyRuleType {
	return CMD
}

func (r *MockPolicyRule) Details() string {
	return "This is a mock policy rule"
}

func (r *MockPolicyRule) Describe() string {
	return "Mock policy rule description"
}

func (r *MockPolicyRule) Test(directives map[string][]DfDirective) *[]Rule {
	return &[]Rule{}
}
