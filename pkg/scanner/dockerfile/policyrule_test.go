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
	"reflect"
	"testing"
)

func TestNewForbidTags(t *testing.T) {
	tags := []string{"latest", "dev"}
	testForbidTag := NewForbidTags(tags)

	if testForbidTag.Type != FORBID_TAGS {
		t.Errorf("Expected Type to be FORBID_TAGS, got %d", testForbidTag.Type)
	}

	if !reflect.DeepEqual(testForbidTag.ForbiddenTags, tags) {
		t.Errorf("Expected ForbiddenTags to be %v, got %v", tags, testForbidTag.ForbiddenTags)
	}
}

func TestForbidTags_Test(t *testing.T) {
	testCases := []struct {
		name        string
		directives  map[string][]DfDirective
		expectedRes *[]Rule
	}{
		{
			name: "Forbidden tag is found",
			directives: map[string][]DfDirective{
				"from": {
					&FromDirective{
						ImageName: "myimage",
						ImageTag:  "latest",
						Content:   "FROM myimage:latest",
					},
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        FORBID_TAGS,
					Details:     "Tag latest is not allowed.",
					Mitigations: "The FROM statements should be changed using an image with a fixed tag or without any of the following tags: latest, dev",
					Statement:   []string{"FROM myimage:latest"},
				},
			},
		},
		{
			name: "Scratch image is skipped",
			directives: map[string][]DfDirective{
				"from": {
					&FromDirective{
						ImageName: "scratch",
						ImageTag:  "",
						Content:   "FROM scratch",
					},
				},
			},
			expectedRes: nil,
		},
		{
			name: "No forbidden tags found",
			directives: map[string][]DfDirective{
				"from": {
					&FromDirective{
						ImageName: "myimage",
						ImageTag:  "v1.0.0",
						Content:   "FROM myimage:v1.0.0",
					},
				},
			},
			expectedRes: nil,
		},
	}

	ft := NewForbidTags([]string{"latest", "dev"})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := ft.Test(tc.directives)
			if !reflect.DeepEqual(res, tc.expectedRes) {
				t.Errorf("Expected %v, got %v", tc.expectedRes, res)
			}
		})
	}
}

func TestForbidTags_Details(t *testing.T) {
	testFt := NewForbidTags([]string{"latest", "dev"})
	expected := "The following tags are forbidden: latest, dev."
	if testFt.Details() != expected {
		t.Errorf("Expected %s, got %s", expected, testFt.Details())
	}
}

func TestNewEnforceRegistryPolicy(t *testing.T) {
	allowedRegistries := []string{"docker.registry"}
	enabled := false
	testEnforceRegistryPolicy := NewEnforceRegistryPolicy(allowedRegistries, enabled)

	if testEnforceRegistryPolicy.Type != ENFORCE_REGISTRY {
		t.Errorf("Expected Type to be ENFORCE_REGISTRY, got %d", testEnforceRegistryPolicy.Type)
	}
	if testEnforceRegistryPolicy.Enabled != enabled {
		t.Errorf("Expected Enabled to be false, got %v", testEnforceRegistryPolicy.Enabled)
	}
	if !reflect.DeepEqual(testEnforceRegistryPolicy.AllowedRegistries, allowedRegistries) {
		t.Errorf("Expected AllowedRegistries to be %v, got %v", allowedRegistries, testEnforceRegistryPolicy.AllowedRegistries)
	}
}

func TestEnforceRegistryPolicy_Test(t *testing.T) {
	testCases := []struct {
		name        string
		directives  map[string][]DfDirective
		expectedRes *[]Rule
	}{
		{
			name: "EnforceRegistryPolicy isn't set",
			directives: map[string][]DfDirective{
				"from": {
					&FromDirective{
						ImageName: "myimage",
						ImageTag:  "latest",
						Content:   "FROM myimage:latest",
					},
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        ENFORCE_REGISTRY,
					Details:     "Registry  不是允许拉取镜像的注册表。",
					Mitigations: "应该更改 FROM 语句，使用允许的注册表之一的镜像：docker.registry",
					Statement:   []string{"FROM myimage:latest"},
				},
			},
		},
		{
			name: "Scratch image is skipped",
			directives: map[string][]DfDirective{
				"from": {
					&FromDirective{
						ImageName: "scratch",
						ImageTag:  "",
						Content:   "FROM scratch",
					},
				},
			},
			expectedRes: nil,
		},
		{
			name: "EnforceRegistryPolicy is set",
			directives: map[string][]DfDirective{
				"from": {
					&FromDirective{
						Registry:  "docker.registry",
						ImageName: "docker.registry/myimage",
						ImageTag:  "dev",
						Content:   "FROM docker.registry/myimage:dev",
					},
				},
			},
			expectedRes: nil,
		},
	}
	erp := NewEnforceRegistryPolicy([]string{"docker.registry"}, true)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := erp.Test(tc.directives)
			if !reflect.DeepEqual(res, tc.expectedRes) {
				t.Errorf("Expected %v, got %v", tc.expectedRes, res)
			}
		})
	}
}

func TestEnforceRegistryPolicy_Details(t *testing.T) {
	testErp := NewEnforceRegistryPolicy([]string{"docker.registry"}, true)
	expected := "The following registries are allowed: docker.registry."
	if testErp.Details() != expected {
		t.Errorf("Expected %s, got %s", expected, testErp.Details())
	}
}

func TestNewForbidInsecureRegistries(t *testing.T) {
	enabled := true
	testForbidInsecureRegistries := NewForbidInsecureRegistries(enabled)
	if testForbidInsecureRegistries.Type != FORBID_INSECURE_REGISTRIES {
		t.Errorf("Expected Type to be FORBID_INSECURE_REGISTRIES, got %d", testForbidInsecureRegistries.Type)
	}
	if testForbidInsecureRegistries.Enabled != enabled {
		t.Errorf("Expected Enabled to be true, got %v", testForbidInsecureRegistries.Enabled)
	}
}

func TestNewForbidInsecureRegistries_Test(t *testing.T) {
	testCases := []struct {
		name        string
		directives  map[string][]DfDirective
		expectedRes *[]Rule
	}{
		{
			name: "ForbidInsecureRegistries is true, http",
			directives: map[string][]DfDirective{
				"from": {
					&FromDirective{
						Registry:  "http://docker.registry",
						ImageName: "myimage",
						ImageTag:  "dev",
						Content:   "FROM http://docker.registry/myimage:dev",
					},
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        FORBID_INSECURE_REGISTRIES,
					Details:     "Registry http://docker.registry uses HTTP and therefore it is considered insecure",
					Mitigations: "The FROM statement should be changed using images from a registry which uses HTTPS.",
					Statement:   []string{"FROM http://docker.registry/myimage:dev"},
				},
			},
		},
		{
			name: "ForbidInsecureRegistries is true, https",
			directives: map[string][]DfDirective{
				"from": {
					&FromDirective{
						Registry:  "https://docker.registry",
						ImageName: "myimage",
						ImageTag:  "dev",
						Content:   "FROM https://docker.registry/myimage:dev",
					},
				},
			},
			expectedRes: nil,
		},
	}
	fir := NewForbidInsecureRegistries(true)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := fir.Test(tc.directives)
			if !reflect.DeepEqual(res, tc.expectedRes) {
				t.Errorf("Expected %v, got %v", tc.expectedRes, res)
			}
		})
	}
}

func TestNewForbidRoot(t *testing.T) {
	enabled := false
	testForbidRoot := NewForbidRoot(enabled)
	if testForbidRoot.Type != FORBID_ROOT {
		t.Errorf("Expected Type to be FORBID_ROOT, got %d", testForbidRoot.Type)
	}
	if testForbidRoot.Enabled != enabled {
		t.Errorf("Expected Enabled to be false, got %v", testForbidRoot.Enabled)
	}
}

func TestForbidRoot_Test(t *testing.T) {
	testCases := []struct {
		name        string
		directives  map[string][]DfDirective
		expectedRes *[]Rule
	}{
		{
			name: "ForbidRoot root",
			directives: map[string][]DfDirective{
				"user": {
					&UserDirective{
						Content: "USER root",
						User:    "root",
						Group:   "root",
					},
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        FORBID_ROOT,
					Details:     "The last USER statement found elevates privileged to root.",
					Mitigations: "Add one more USER statement before the entrypoint of the image to run the application as a non-privileged user.",
					Statement:   []string{"USER root"},
				},
			},
		},
		{
			name: "ForbidRoot rootless",
			directives: map[string][]DfDirective{
				"user": {
					&UserDirective{
						Content: "USER rootless",
						User:    "rootless",
						Group:   "rootless",
					},
				},
			},
			expectedRes: nil,
		},
		// todo，func (rule *ForbidRoot) Test逻辑修复后，补充对user为空情景的单元测试
	}
	fr := NewForbidRoot(true)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := fr.Test(tc.directives)
			if !reflect.DeepEqual(res, tc.expectedRes) {
				t.Errorf("Expected %v, got %v", tc.expectedRes, res)
			}
		})
	}
}

func TestNewForbidPrivilegedPorts(t *testing.T) {
	enabled := false
	testForbidPrivilegedPorts := NewForbidPrivilegedPorts(enabled)
	if testForbidPrivilegedPorts.Type != FORBID_PRIVILEGED_PORTS {
		t.Errorf("Expected Type to be FORBID_PRIVILEGED_PORTS, got %d", testForbidPrivilegedPorts.Type)
	}
	if testForbidPrivilegedPorts.Enabled != enabled {
		t.Errorf("Expected Enabled to be false, got %v", testForbidPrivilegedPorts.Enabled)
	}
}

func TestForbidPrivilegedPorts_Test(t *testing.T) {
	testCases := []struct {
		name        string
		directives  map[string][]DfDirective
		expectedRes *[]Rule
	}{
		{
			name: "ForbidPrivilegedPorts 1024",
			directives: map[string][]DfDirective{
				"expose": {
					&ExposeDirective{
						Content: "1024",
						Ports:   []string{"1024"},
					},
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        FORBID_PRIVILEGED_PORTS,
					Details:     "The container exposes a privileged port: 1024. Privileged ports require the application which uses it to run as root.",
					Mitigations: "Change the configuration for the application to bind on a port greater than 1024, and change the Dockerfile to reflect this modification.",
					Statement:   []string{"1024"},
				},
			},
		},
		{
			name: "ForbidPrivilegedPorts 8080",
			directives: map[string][]DfDirective{
				"expose": {
					&ExposeDirective{
						Content: "8080",
						Ports:   []string{"8080"},
					},
				},
			},
			expectedRes: nil,
		},
	}
	fpr := NewForbidPrivilegedPorts(true)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := fpr.Test(tc.directives)
			if !reflect.DeepEqual(res, tc.expectedRes) {
				t.Errorf("Expected %v, got %v", tc.expectedRes, res)
			}
		})
	}
}

func TestNewForbidPackages(t *testing.T) {
	testPackages := []string{"make", "iputils"}
	testNewForbidPackages := NewForbidPackages(testPackages)
	if testNewForbidPackages.Type != FORBID_PACKAGES {
		t.Errorf("Expected Type to be FORBID_PACKAGES, got %d", testNewForbidPackages.Type)
	}
	if !reflect.DeepEqual(testNewForbidPackages.ForbiddenPackages, testPackages) {
		t.Errorf("Expected ForbidPackages to be %v, got %v", testPackages, testNewForbidPackages.ForbiddenPackages)
	}
}

func TestNewForbidPackages_Test(t *testing.T) {
	testCases := []struct {
		name        string
		directives  map[string][]DfDirective
		expectedRes *[]Rule
	}{
		{
			name: "Forbidden package found in RUN/CMD/ENTRYPOINT",
			directives: map[string][]DfDirective{
				"run": {
					&RunDirective{
						Type:    RUN,
						Content: "RUN apt-get install -y forbidden_package",
					},
				},
				"entrypoint": {
					&EntrypointDirective{
						Type:    ENTRYPOINT,
						Content: "ENTRYPOINT [\"forbidden_package\"]",
					},
				},
				"cmd": {
					&CmdDirective{
						Type:    CMD,
						Content: "CMD [\"forbidden_package\"]",
					},
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        FORBID_PACKAGES,
					Details:     "Forbidden package \"forbidden_package\" is installed or used.",
					Mitigations: "The RUN/CMD/ENTRYPOINT statement should be reviewed and package \"forbidden_package\" should be removed unless absolutely necessary.",
					Statement:   []string{"ENTRYPOINT [\"forbidden_package\"]"},
				},
				{
					Type:        FORBID_PACKAGES,
					Details:     "Forbidden package \"forbidden_package\" is installed or used.",
					Mitigations: "The RUN/CMD/ENTRYPOINT statement should be reviewed and package \"forbidden_package\" should be removed unless absolutely necessary.",
					Statement:   []string{"RUN apt-get install -y forbidden_package"},
				},
				{
					Type:        FORBID_PACKAGES,
					Details:     "Forbidden package \"forbidden_package\" is installed or used.",
					Mitigations: "The RUN/CMD/ENTRYPOINT statement should be reviewed and package \"forbidden_package\" should be removed unless absolutely necessary.",
					Statement:   []string{"CMD [\"forbidden_package\"]"},
				},
			},
		},
		{
			name: "Forbidden package not found in RUN/CMD/ENTRYPOINT",
			directives: map[string][]DfDirective{
				"run": {
					&RunDirective{
						Type:    RUN,
						Content: "RUN apt-get install -y package",
					},
				},
				"entrypoint": {
					&EntrypointDirective{
						Type:    ENTRYPOINT,
						Content: "ENTRYPOINT [\"package\"]",
					},
				},
				"cmd": {
					&CmdDirective{
						Type:    CMD,
						Content: "CMD [\"package\"]",
					},
				},
			},
			expectedRes: nil,
		},
	}
	fpt := NewForbidPackages([]string{"forbidden_package"})
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := fpt.Test(tc.directives)
			if !reflect.DeepEqual(res, tc.expectedRes) {
				t.Errorf("Expected %v, got %v", tc.expectedRes, res)
			}
		})
	}
}

func TestNewForbidSecrets(t *testing.T) {
	secretsPatterns := []string{"123456"}
	allowedPatterns := []string{"654321"}
	testNewForbidSecrets := NewForbidSecrets(secretsPatterns, allowedPatterns)
	if testNewForbidSecrets.Type != FORBID_SECRETS {
		t.Errorf("Expected Type to be FORBID_SECRETS, got %d", testNewForbidSecrets.Type)
	}
	if !reflect.DeepEqual(testNewForbidSecrets.secretsPatterns, secretsPatterns) {
		t.Errorf("Expected secretsPatterns to be %v, got %v", secretsPatterns, testNewForbidSecrets.secretsPatterns)
	}
	if !reflect.DeepEqual(testNewForbidSecrets.allowedPatterns, allowedPatterns) {
		t.Errorf("Expected allowedPatterns to be %v, got %v", allowedPatterns, testNewForbidSecrets.allowedPatterns)
	}
}

func TestNewForbidSecrets_Test(t *testing.T) {
	testCases := []struct {
		name        string
		directives  map[string][]DfDirective
		expectedRes *[]Rule
	}{
		{
			name: "Forbidden Secrets not found in ADD/COPY",
			directives: map[string][]DfDirective{
				"add": {
					&AddDirective{
						Type:        ADD,
						Content:     "testsrc1 testdes1",
						Source:      "testsrc1",
						Destination: "testdes1",
					},
				},
				"copy": {
					&CopyDirective{
						Type:        COPY,
						Content:     "testsrc2 testdes2",
						Source:      "testsrc2",
						Destination: "testdes2",
					},
				},
			},
			expectedRes: nil,
		},
		{
			name: "Forbidden Secrets found in ADD/COPY",
			directives: map[string][]DfDirective{
				"add": {
					&AddDirective{
						Type:        ADD,
						Content:     "secret testdes1",
						Source:      "secret",
						Destination: "testdes1",
					},
				},
				"copy": {
					&CopyDirective{
						Type:        COPY,
						Content:     "secret testdes2",
						Source:      "secret",
						Destination: "testdes2",
					},
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        FORBID_SECRETS,
					Details:     "Forbidden file matching pattern \"secret\" is added into the image.",
					Mitigations: "The ADD statement should be changed or removed. Secrets should be provisioned using a safer and stateless way (Vault, Kubernetes secrets) instead.",
					Statement:   []string{"secret testdes1"},
				},
				{
					Type:        FORBID_SECRETS,
					Details:     "Forbidden file matching pattern \"secret\" is added into the image.",
					Mitigations: "The COPY statement should be changed or removed. Secrets should be provisioned using a safer and stateless way (Vault, Kubernetes secrets) instead.",
					Statement:   []string{"secret testdes2"},
				},
			},
		},
	}
	fs := NewForbidSecrets([]string{"secret"}, []string{"allow"})
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := fs.Test(tc.directives)
			if !reflect.DeepEqual(res, tc.expectedRes) {
				t.Errorf("Expected %v, got %v", tc.expectedRes, res)
			}
		})
	}
}

func TestNewForbidSecrets_Details(t *testing.T) {
	testFsd := NewForbidSecrets([]string{"secret"}, []string{"allow"})
	expected := "The following patterns are forbidden: secret.\nThe following patterns are whitelisted: allow"
	if testFsd.Details() != expected {
		t.Errorf("Expected %s, got %s", expected, testFsd.Details())
	}
}
