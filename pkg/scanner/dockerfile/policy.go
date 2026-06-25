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
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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

type policyFileConfig struct {
	Policy *policyConfig `yaml:"policy"`
}

type policyConfig struct {
	EnforceAuthorizedRegistries registryPolicyConfig `yaml:"enforce_authorized_registries"`
	ForbidFloatingTags          tagsPolicyConfig     `yaml:"forbid_floating_tags"`
	ForbidInsecureRegistries    enabledPolicyConfig  `yaml:"forbid_insecure_registries"`
	ForbidRoot                  enabledPolicyConfig  `yaml:"forbid_root"`
	ForbidPrivilegedPorts       enabledPolicyConfig  `yaml:"forbid_privileged_ports"`
	ForbidPackages              packagesPolicyConfig `yaml:"forbid_packages"`
	ForbidSecrets               secretsPolicyConfig  `yaml:"forbid_secrets"`
}

type enabledPolicyConfig struct {
	Enabled bool `yaml:"enabled"`
}

type registryPolicyConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Registries []string `yaml:"registries"`
}

type tagsPolicyConfig struct {
	Enabled       bool     `yaml:"enabled"`
	ForbiddenTags []string `yaml:"forbidden_tags"`
}

type packagesPolicyConfig struct {
	Enabled           bool     `yaml:"enabled"`
	ForbiddenPackages []string `yaml:"forbidden_packages"`
}

type secretsPolicyConfig struct {
	Enabled         bool     `yaml:"enabled"`
	SecretsPatterns []string `yaml:"secrets_patterns"`
	AllowedPatterns []string `yaml:"allowed_patterns"`
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
			// Convert rule type to string
			for i := range *testRuleResults {
				(*testRuleResults)[i].Type = rule.GetType()
				// Check for failures
				if (*testRuleResults)[i].Status == "fail" {
					hasFailures = true
				}
			}
			testResults = append(testResults, *testRuleResults...)
		}
	}

	// Determine the overall result based on whether there are failures
	auditOutcome := "pass"
	if hasFailures {
		auditOutcome = "fail"
	}

	return PolicyResult{
		Tests:        testResults, // Now includes compliant and non-compliant items
		Filename:     dockerfileObject.GetFilename(),
		AuditOutcome: auditOutcome,
		Maintainers:  dockerfileObject.GetMaintainers(),
		Path:         dockerfileObject.GetPath(),
	}
}

func (p *Policy) initRules() error {
	yamlFile, err := os.ReadFile(p.PolicyFile)
	if err != nil {
		logrus.Errorf("Failed to read %s: %v", p.PolicyFile, err)
		return errors.New("failed to read policy file")
	}
	var config policyFileConfig
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		logrus.Errorf("Failed to parse %s: %v", p.PolicyFile, err)
		return fmt.Errorf("invalid yaml file: %w", err)
	}
	if config.Policy == nil {
		logrus.Error("Invalid policy file format: missing 'policy' section")
		return errors.New("invalid policy file format")
	}

	p.PolicyRules = p.PolicyRules[:0]
	p.addRegistryRule(config.Policy.EnforceAuthorizedRegistries)
	p.addForbidTagsRule(config.Policy.ForbidFloatingTags)
	p.addEnabledRule(config.Policy.ForbidInsecureRegistries, func(enabled bool) PolicyRule {
		return NewForbidInsecureRegistries(enabled)
	})
	p.addEnabledRule(config.Policy.ForbidRoot, func(enabled bool) PolicyRule {
		return NewForbidRoot(enabled)
	})
	p.addEnabledRule(config.Policy.ForbidPrivilegedPorts, func(enabled bool) PolicyRule {
		return NewForbidPrivilegedPorts(enabled)
	})
	p.addPackagesRule(config.Policy.ForbidPackages)
	p.addSecretsRule(config.Policy.ForbidSecrets)
	return nil
}

func (p *Policy) addRegistryRule(cfg registryPolicyConfig) {
	if !cfg.Enabled || len(cfg.Registries) == 0 {
		return
	}
	p.PolicyRules = append(p.PolicyRules, NewEnforceRegistryPolicy(cfg.Registries, cfg.Enabled))
}

func (p *Policy) addForbidTagsRule(cfg tagsPolicyConfig) {
	if !cfg.Enabled || len(cfg.ForbiddenTags) == 0 {
		return
	}
	p.PolicyRules = append(p.PolicyRules, NewForbidTags(cfg.ForbiddenTags))
}

func (p *Policy) addEnabledRule(cfg enabledPolicyConfig, factory func(bool) PolicyRule) {
	if !cfg.Enabled {
		return
	}
	p.PolicyRules = append(p.PolicyRules, factory(cfg.Enabled))
}

func (p *Policy) addPackagesRule(cfg packagesPolicyConfig) {
	if !cfg.Enabled || len(cfg.ForbiddenPackages) == 0 {
		return
	}
	p.PolicyRules = append(p.PolicyRules, NewForbidPackages(cfg.ForbiddenPackages))
}

func (p *Policy) addSecretsRule(cfg secretsPolicyConfig) {
	if !cfg.Enabled {
		return
	}
	if len(cfg.SecretsPatterns) == 0 {
		logrus.Warn("forbid_secrets is enabled but no secrets_patterns are defined.")
		return
	}
	p.PolicyRules = append(p.PolicyRules, NewForbidSecrets(cfg.SecretsPatterns, cfg.AllowedPatterns))
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
