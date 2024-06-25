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

// AuditException is an error type for audit exceptions.
type AuditException struct {
	Message string
}

// DockerfileAuditor is a struct for DockerfileMsg auditing.
type DockerfileAuditor struct {
	Policy Policy
}

func (e *AuditException) Error() string {
	return e.Message
}

// NewDockerfileAuditor creates a new instance of DfAuditor with the given policy.
func NewDockerfileAuditor(policy Policy) *DockerfileAuditor {
	return &DockerfileAuditor{
		Policy: policy,
	}
}

// Audit performs the audit operation on the specified file path.
func (auditor *DockerfileAuditor) Audit(path string) (PolicyResult, error) {
	dockerfile, err := NewDockerfile(path)
	if err != nil {
		return PolicyResult{}, err
	}
	policyResult := auditor.Policy.EvaluateDockerfile(*dockerfile)
	return policyResult, nil
}

// ParseOnly 仅对指定的文件路径执行分析操作。
func (auditor *DockerfileAuditor) ParseOnly(path string) (ParseResult, error) {
	dockerfile, err := NewDockerfile(path)
	if err != nil {
		return ParseResult{}, err
	}
	result := ParseResult{
		Filename:    dockerfile.Filename,
		Path:        dockerfile.Path,
		Maintainers: dockerfile.GetMaintainers(),
		Directives:  dockerfile.GetDirectives(),
	}
	return result, nil
}
