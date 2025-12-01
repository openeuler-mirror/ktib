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
    "errors"
    "io/ioutil"

    "gopkg.in/yaml.v2"
    "github.com/sirupsen/logrus"
)

type Policy struct {
	PolicyRules []PolicyRule
	PolicyFile  string
}

type PolicyResult struct {
	Filename     string
	Tests        []Rule
	AuditOutcome string
	Maintainers  string
	Path         string
}

func NewDockerfilePolicy(policyFile string) (*Policy, error) {
	policy := &Policy{
		PolicyFile: policyFile,
	}
	err := policy.initRules()
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func (p *Policy) EvaluateDockerfile(dockerfileObject Dockerfile) PolicyResult {
	var testResults []Rule
	hasFailures := false

	for _, rule := range p.PolicyRules {
		testRuleResults := rule.Test(dockerfileObject.GetDirectives())
		if testRuleResults != nil && len(*testRuleResults) > 0 {
			// 转换规则类型为字符串
			for i := range *testRuleResults {
				(*testRuleResults)[i].Type = rule.GetType()
				// 检查是否有失败项
				if (*testRuleResults)[i].Status == "fail" {
					hasFailures = true
				}
			}
			testResults = append(testResults, *testRuleResults...)
		}
	}

	// 根据是否有失败项决定整体结果
	auditOutcome := "pass"
	if hasFailures {
		auditOutcome = "fail"
	}

	return PolicyResult{
		Tests:        testResults, // 现在包含合规和不合规项
		Filename:     dockerfileObject.GetFilename(),
		AuditOutcome: auditOutcome,
		Maintainers:  dockerfileObject.GetMaintainers(),
		Path:         dockerfileObject.GetPath(),
	}
}

func (p *Policy) initRules() error {
	var policyRules map[string]interface{}
    yamlFile, err := ioutil.ReadFile(p.PolicyFile)
    if err != nil {
        logrus.Errorf("Failed to read %s: %v", p.PolicyFile, err)
        return errors.New("failed to read policy file")
    }
    err = yaml.Unmarshal(yamlFile, &policyRules)
    if err != nil {
        logrus.Errorf("Failed to parse %s: %v", p.PolicyFile, err)
        return errors.New("invalid yaml file")
    }
	policies, ok := policyRules["policy"].(map[interface{}]interface{})
    if !ok {
        logrus.Error("Invalid policy file format: missing 'policy' section")
        return errors.New("invalid policy file format")
    }

	if enforceRegistries, ok := policies["enforce_authorized_registries"].(map[interface{}]interface{}); ok {
		if enabled, ok := enforceRegistries["enabled"].(bool); ok {
			registries, ok := enforceRegistries["registries"]
			if ok {
				strSlice := make([]string, len(registries.([]interface{})))
				for i, v := range registries.([]interface{}) {
					strSlice[i] = v.(string)
				}
				p.PolicyRules = append(p.PolicyRules, NewEnforceRegistryPolicy(strSlice, enabled))
			}
		}
	}

	if forbidTags, ok := policies["forbid_floating_tags"].(map[interface{}]interface{}); ok {
		if enabled, ok := forbidTags["enabled"].(bool); ok && enabled {
			tags, ok := forbidTags["forbidden_tags"]
			if ok {
				strSlice := make([]string, len(tags.([]interface{})))
				for i, v := range tags.([]interface{}) {
					strSlice[i] = v.(string)
				}
				p.PolicyRules = append(p.PolicyRules, NewForbidTags(strSlice))
			}
		}
	}

	if forbidInsecureRegistries, ok := policies["forbid_insecure_registries"].(map[interface{}]interface{}); ok {
		if enabled, ok := forbidInsecureRegistries["enabled"].(bool); ok && enabled {
			insecureRegistries := NewForbidInsecureRegistries(enabled)
			p.PolicyRules = append(p.PolicyRules, insecureRegistries)
		}
	}

	if forbidRoot, ok := policies["forbid_root"].(map[interface{}]interface{}); ok {
		if enabled, ok := forbidRoot["enabled"].(bool); ok && enabled {
			p.PolicyRules = append(p.PolicyRules, NewForbidRoot(enabled))
		}
	}

	if forbidPrivilegedPorts, ok := policies["forbid_privileged_ports"].(map[interface{}]interface{}); ok {
		if enabled, ok := forbidPrivilegedPorts["enabled"].(bool); ok && enabled {
			p.PolicyRules = append(p.PolicyRules, NewForbidPrivilegedPorts(enabled))
		}
	}

	if forbidPackages, ok := policies["forbid_packages"].(map[interface{}]interface{}); ok {
		if enabled, ok := forbidPackages["enabled"].(bool); ok && enabled {
			packages, ok := forbidPackages["forbidden_packages"]
			if ok {
				strSlice := make([]string, len(packages.([]interface{})))
				for i, v := range packages.([]interface{}) {
					strSlice[i] = v.(string)
				}
				p.PolicyRules = append(p.PolicyRules, NewForbidPackages(strSlice))
			}
		}
	}
	if forbidSecrets, ok := policies["forbid_secrets"].(map[interface{}]interface{}); ok {
		var secretsPatterns []string
		var allowedPatterns []string
		if enabled, ok := forbidSecrets["enabled"].(bool); ok && enabled {
			patterns, ok := forbidSecrets["secrets_patterns"].([]interface{})
			if ok {
				for _, pattern := range patterns {
					if p, ok := pattern.(string); ok {
						secretsPatterns = append(secretsPatterns, p)
					} else {
                        logrus.Warnf("Invalid secrets_patterns format, skipping: %v", pattern)
                        continue
                    }
                }
            } else {
                logrus.Warn("Invalid secrets_patterns format. Skipping.")
            }
            if patterns, ok := forbidSecrets["allowed_patterns"].([]interface{}); ok {
                for _, pattern := range patterns {
                    if p, ok := pattern.(string); ok {
                        allowedPatterns = append(allowedPatterns, p)
                    } else {
                        logrus.Warnf("Invalid allowed_patterns format, skipping: %v", pattern)
                        continue
                    }
                }
                forbidSecrets := NewForbidSecrets(secretsPatterns, allowedPatterns)
                p.PolicyRules = append(p.PolicyRules, forbidSecrets)
            }
        }
    } else {
        logrus.Warn("forbid_secrets rule added but no secrets_patterns defined.")
    }

	return nil
}

func (p *Policy) GetPolicyRulesEnabled() []Rule {
	enabledRules := make([]Rule, 0)
	for _, rule := range p.PolicyRules {
		var ruleInterface interface{} = rule
		switch rule.GetType() {
		case ENFORCE_REGISTRY:
			ruleDetails := ruleInterface.(*EnforceRegistryPolicy).Details()
			enabledRules = append(enabledRules, Rule{
				// "description": rule.Describe(),
				Type:        rule.GetType(),
				Mitigations: rule.Describe(),
				Details:     ruleDetails,
			})
		case FORBID_TAGS:
			ruleDetails := ruleInterface.(*ForbidTags).Details()
			enabledRules = append(enabledRules, Rule{
				Type:        rule.GetType(),
				Mitigations: rule.Describe(),
				Details:     ruleDetails,
			})
		case FORBID_PACKAGES:
			ruleDetails := ruleInterface.(*ForbidPackages).Details()
			enabledRules = append(enabledRules, Rule{
				Type:        rule.GetType(),
				Mitigations: rule.Describe(),
				Details:     ruleDetails,
			})
		case FORBID_SECRETS:
			ruleDetails := ruleInterface.(*ForbidSecrets).Details()
			enabledRules = append(enabledRules, Rule{
				Type:        rule.GetType(),
				Mitigations: rule.Describe(),
				Details:     ruleDetails,
			})
		}
	}
	return enabledRules
}
