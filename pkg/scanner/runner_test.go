/*
   Copyright (c) 2026 KylinSoft Co., Ltd.
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
	"io"
	"os"
	"path/filepath"
	"testing"

	o "gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/scanner/dockerfile"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Disable logrus output to keep test logs clean
	logrus.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestGetFilesToProcess(t *testing.T) {
	t.Run("single file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "Dockerfile")
		err := os.WriteFile(filePath, []byte("FROM alpine"), 0644)
		assert.NoError(t, err)

		files := getFilesToProcess(filePath)
		assert.Len(t, files, 1)
		assert.Equal(t, filePath, files[0])
	})

	t.Run("directory with multiple files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create file 1
		file1 := filepath.Join(tmpDir, "Dockerfile1")
		err := os.WriteFile(file1, []byte("FROM alpine"), 0644)
		assert.NoError(t, err)

		// Create subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		assert.NoError(t, err)

		// Create file 2 in subdirectory
		file2 := filepath.Join(subDir, "Dockerfile2")
		err = os.WriteFile(file2, []byte("FROM ubuntu"), 0644)
		assert.NoError(t, err)

		files := getFilesToProcess(tmpDir)

		// Should find both files
		assert.Len(t, files, 2)
		assert.Contains(t, files, file1)
		assert.Contains(t, files, file2)
	})
}

func TestParse(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "Dockerfile")
	content := `FROM alpine:latest
RUN echo "hello world"`
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)

	filesToProcess := []string{filePath}
	results := parse(filesToProcess)

	assert.Len(t, results, 1)
	assert.Equal(t, filePath, results[0].Path)
	assert.Equal(t, "Dockerfile", results[0].Filename)
	// We expect directives: FROM and RUN to be present
	assert.Contains(t, results[0].Directives, "from")
	assert.Contains(t, results[0].Directives, "run")
}

func TestAudit(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "Dockerfile")
	content := `FROM alpine:latest
RUN echo "hello world"`
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)

	// Create an empty policy
	policy := &dockerfile.Policy{
		PolicyRules: []dockerfile.PolicyRule{},
		PolicyFile:  "test-policy.yaml",
	}

	filesToProcess := []string{filePath}
	results := audit(filesToProcess, policy)

	assert.Len(t, results, 1)
	assert.Equal(t, filePath, results[0].Path)
	assert.Equal(t, "Dockerfile", results[0].Filename)
	assert.Equal(t, "pass", results[0].AuditOutcome)
	// With empty policy, there should be no test results (rules executed)
	assert.Empty(t, results[0].Tests)
}

func TestRunDockerfileAudit_ParseOnly(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfile := filepath.Join(tmpDir, "Dockerfile")
	jsonOut := filepath.Join(tmpDir, "output.json")

	err := os.WriteFile(dockerfile, []byte("FROM alpine:latest\nRUN echo hello"), 0644)
	assert.NoError(t, err)

	args := o.Arguments{
		Dockerfile:  dockerfile,
		ParseOnly:   true,
		JSONOutfile: jsonOut,
	}

	// Run the function
	RunDockerfileAudit(args)

	// Verify output file exists
	assert.FileExists(t, jsonOut)

	// Verify content is valid JSON and contains expected data
	content, err := os.ReadFile(jsonOut)
	assert.NoError(t, err)

	// Use anonymous struct to avoid direct dependency on internal types if not exported
	var results []struct {
		Path string `json:"path"`
	}
	err = json.Unmarshal(content, &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, dockerfile, results[0].Path)
}

func TestGetPolicy_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	yamlContent := `
policy:
  forbid_root:
    enabled: true
`
	err := os.WriteFile(policyFile, []byte(yamlContent), 0644)
	assert.NoError(t, err)

	policy, err := getPolicy(policyFile)
	assert.NoError(t, err)
	assert.NotNil(t, policy)
	assert.Equal(t, policyFile, policy.PolicyFile)
	// Check if rule was actually loaded
	assert.NotEmpty(t, policy.PolicyRules)
}
