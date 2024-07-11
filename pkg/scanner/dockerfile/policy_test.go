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

func Test_NewDockerfilePolicy(t *testing.T) {
	testPolicyFilePath := "testdata/policy.yaml"
	testPolicyFileContent := `policy:
  enforce_authorized_registries:
    enabled: True
    registries:
      - Docker Hub
      - https://test.example.com:5000
  forbid_floating_tags:
    enabled: True
    forbidden_tags:
      - latest
      - stable
      - prod
      - stage
  forbid_insecure_registries:
    enabled: True
  forbid_root:
    enabled: True
  forbid_privileged_ports:
    enabled: True
  forbid_packages:
    enabled: True
    forbidden_packages:
      - sudo
      - vim
      - netcat
      - nc
      - curl
      - wget
  forbid_secrets:
    enabled: True
    secrets_patterns:
      - id_rsa
      - private_key
      - password
      - key
      - secret
    allowed_patterns:
      - id_rsa.pub`
	err := os.MkdirAll("testdata", 0755)
	if err != nil {
		t.Errorf("创建 testdata 目录时出错: %v", err)
	}
	err = os.WriteFile(testPolicyFilePath, []byte(testPolicyFileContent), 0644)
	if err != nil {
		t.Errorf("创建有效 policy.yaml 文件时出错: %v", err)
	}
	defer os.Remove(testPolicyFilePath)

	testPolicy, err := NewDockerfilePolicy(testPolicyFilePath)
	if err != nil {
		t.Errorf("NewDockerfilePolicy() 返回了错误: %v", err)
	}
	if testPolicy.PolicyFile != testPolicyFilePath {
		t.Errorf("预期路径为 '%s', 实际为 %s", testPolicyFilePath, testPolicy.PolicyFile)
	}
}

func Test_GetPolicyRulesEnabled(t *testing.T) {
	// 创建一些测试用的 PolicyRule 实现
	enforceRegistryRule := &EnforceRegistryPolicy{
		GenericPolicyRule: GenericPolicyRule{
			Type:        ENFORCE_REGISTRY,
			TestResult:  PolicyTestResult{},
			Description: "Enforce Registry",
		},
		AllowedRegistries: nil,
		Enabled:           false,
	}
	forbidTagsRule := &ForbidTags{
		GenericPolicyRule: GenericPolicyRule{
			Type:        FORBID_TAGS,
			TestResult:  PolicyTestResult{},
			Description: "Forbid Tags",
		},
		ForbiddenTags: nil,
	}
	forbidPackagesRule := &ForbidPackages{
		ForbiddenPackages: nil,
		GenericPolicyRule: GenericPolicyRule{
			Type:        FORBID_PACKAGES,
			TestResult:  PolicyTestResult{},
			Description: "Forbid Packages",
		},
	}
	forbidSecretsRule := &ForbidSecrets{
		GenericPolicyRule: GenericPolicyRule{
			Type:        FORBID_SECRETS,
			TestResult:  PolicyTestResult{},
			Description: "Forbid Secrets",
		},
		secretsPatterns: nil,
		allowedPatterns: nil,
	}
	policy := &Policy{
		PolicyRules: []PolicyRule{
			enforceRegistryRule,
			forbidTagsRule,
			forbidPackagesRule,
			forbidSecretsRule,
		},
	}
	testRules := policy.GetPolicyRulesEnabled()
	require.Len(t, testRules, 4)
	reflect.DeepEqual(Rule{
		Type:        ENFORCE_REGISTRY,
		Mitigations: "Enforce Registry",
		Details:     "Enforce Registry Details",
	}, testRules[0])

	reflect.DeepEqual(Rule{
		Type:        FORBID_TAGS,
		Mitigations: "Forbid Tags",
		Details:     "Forbid Tags Details",
	}, testRules[1])

	reflect.DeepEqual(Rule{
		Type:        FORBID_PACKAGES,
		Mitigations: "Forbid Packages",
		Details:     "Forbid Packages Details",
	}, testRules[2])

	reflect.DeepEqual(Rule{
		Type:        FORBID_SECRETS,
		Mitigations: "Forbid Secrets",
		Details:     "Forbid Secrets Details",
	}, testRules[3])
}
