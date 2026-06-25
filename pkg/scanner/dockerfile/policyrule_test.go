/*
   Copyright (c) 2024 KylinSoft Co., Ltd.
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
					Mitigations: "The FROM statement should be changed to use an image with a fixed tag, or not use any of the following tags: latest, dev",
					Statement:   []string{"FROM myimage:latest"},
					Status:      "fail",
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
			expectedRes: &[]Rule{
				{
					Type:        FORBID_TAGS,
					Details:     "Tag v1.0.0 is allowed.",
					Mitigations: "",
					Statement:   []string{"FROM myimage:v1.0.0"},
					Status:      "pass",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ft := NewForbidTags([]string{"latest", "dev"})
			res := ft.Test(tc.directives)
			if !reflect.DeepEqual(res, tc.expectedRes) {
				t.Errorf("Expected %v, got %v", tc.expectedRes, res)
			}
		})
	}
}

func TestForbidTags_Details(t *testing.T) {
	testFt := NewForbidTags([]string{"latest", "dev"})
	expected := "The following tags are forbidden: latest, dev。"
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
						Registry:  "unallowed.registry",
						ImageName: "unallowed.registry/myimage",
						ImageTag:  "latest",
						Content:   "FROM unallowed.registry/myimage:latest",
					},
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        ENFORCE_REGISTRY,
					Details:     "Registry unallowed.registry is not an allowed registry for pulling images.",
					Mitigations: "The FROM statement should be changed to use an image from an allowed image repository registry：docker.registry",
					Statement:   []string{"FROM unallowed.registry/myimage:latest"},
					Status:      "fail",
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
			expectedRes: &[]Rule{},
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
			expectedRes: &[]Rule{
				{
					Type:        ENFORCE_REGISTRY,
					Details:     "Registry docker.registry is an allowed image repository registry.",
					Mitigations: "",
					Statement:   []string{"FROM docker.registry/myimage:dev"},
					Status:      "pass",
				},
			},
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
	expected := "Allowed registries: docker.registry."
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
					Details:     "Registry http://docker.registry is considered insecure",
					Mitigations: "The FROM statement should be changed to use an image from a registry using the HTTPS protocol.",
					Statement:   []string{"FROM http://docker.registry/myimage:dev"},
					Status:      "fail",
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
			expectedRes: &[]Rule{
				{
					Type:        FORBID_INSECURE_REGISTRIES,
					Details:     "Registry https://docker.registry is considered secure",
					Mitigations: "",
					Statement:   []string{"FROM https://docker.registry/myimage:dev"},
					Status:      "pass",
				},
			},
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
				NewUserDirective("USER root"),
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        FORBID_ROOT,
					Details:     "The last USER instruction elevates privileges to root.",
					Mitigations: "Add another USER instruction before the image's entrypoint to run the application as a non-privileged user.",
					Statement:   []string{"USER root"},
					Status:      "fail",
				},
			},
		},
		{
			name: "ForbidRoot rootless",
			directives: map[string][]DfDirective{
				"user": {
				NewUserDirective("USER rootless"),
				},
			},
			expectedRes: &[]Rule{
				{
					Type:        FORBID_ROOT,
					Details:     "The last USER instruction specifies a non-privileged user.",
					Mitigations: "",
					Statement:   []string{"USER rootless"},
					Status:      "pass",
				},
			},
		},
	{
		name: "ForbidRoot numeric root user",
		directives: map[string][]DfDirective{
			"user": {
				NewUserDirective("USER 0:0"),
			},
		},
		expectedRes: &[]Rule{
			{
				Type:        FORBID_ROOT,
				Details:     "The last USER instruction elevates privileges to root.",
				Mitigations: "Add another USER instruction before the image's entrypoint to run the application as a non-privileged user.",
				Statement:   []string{"USER 0:0"},
				Status:      "fail",
			},
		},
	},
	{
		name:       "ForbidRoot missing user instruction",
		directives: map[string][]DfDirective{},
		expectedRes: &[]Rule{
			{
				Type:        FORBID_ROOT,
				Details:     "USER instruction not found. By default, the container will run as the root user if privileges are not dropped.",
				Mitigations: "Create a user and add a USER instruction before the image's entrypoint to run the application as a non-privileged user.",
				Status:      "fail",
			},
		},
	},
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
					Details:     "Container exposes privileged port: 1024. Privileged ports require the application using it to run as root.",
					Mitigations: "Change the application's configuration to bind to a port greater than 1024, and change the Dockerfile to reflect this modification.",
					Statement:   []string{"1024"},
					Status:      "fail",
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
			expectedRes: &[]Rule{
				{
					Type:        FORBID_PRIVILEGED_PORTS,
					Details:     "Container does not expose privileged ports, only exposes non-privileged port: 8080, which meets security requirements.",
					Mitigations: "",
					Statement:   []string{"8080"},
					Status:      "pass",
				},
			},
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
					Mitigations: "The RUN/CMD/ENTRYPOINT instruction should be reviewed and package \"forbidden_package\" removed unless absolutely necessary.",
					Statement:   []string{"ENTRYPOINT [\"forbidden_package\"]"},
					Status:      "fail",
				},
				{
					Type:        FORBID_PACKAGES,
					Details:     "Forbidden package \"forbidden_package\" is installed or used.",
					Mitigations: "The RUN/CMD/ENTRYPOINT instruction should be reviewed and package \"forbidden_package\" removed unless absolutely necessary.",
					Statement:   []string{"RUN apt-get install -y forbidden_package"},
					Status:      "fail",
				},
				{
					Type:        FORBID_PACKAGES,
					Details:     "Forbidden package \"forbidden_package\" is installed or used.",
					Mitigations: "The RUN/CMD/ENTRYPOINT instruction should be reviewed and package \"forbidden_package\" removed unless absolutely necessary.",
					Statement:   []string{"CMD [\"forbidden_package\"]"},
					Status:      "fail",
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
			expectedRes: &[]Rule{},
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
			expectedRes: &[]Rule{
				{
					Type:        FORBID_SECRETS,
					Details:     "No sensitive information identified.",
					Mitigations: "",
					Statement:   []string{"testsrc1 testdes1"},
					Status:      "pass",
				},
				{
					Type:        FORBID_SECRETS,
					Details:     "No sensitive information identified.",
					Mitigations: "",
					Statement:   []string{"testsrc2 testdes2"},
					Status:      "pass",
				},
			},
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
					Details:     "Forbidden file matching pattern \"secret\" is added to the image.",
					Mitigations: "The ADD instruction should be changed or removed. Safer and stateless methods (like Vault, Kubernetes secrets) should be used to provide sensitive information.",
					Statement:   []string{"secret testdes1"},
					Status:      "fail",
				},
				{
					Type:        FORBID_SECRETS,
					Details:     "Forbidden file matching pattern \"secret\" is added to the image.",
					Mitigations: "The COPY instruction should be changed or removed. Safer and stateless methods (like Vault, Kubernetes secrets) should be used to provide sensitive information.",
					Statement:   []string{"secret testdes2"},
					Status:      "fail",
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
	expected := "The following patterns are forbidden: secret.\nThe following patterns are allowed: allow"
	if testFsd.Details() != expected {
		t.Errorf("Expected %s, got %s", expected, testFsd.Details())
	}
}

func TestRule_MarshalJSON(t *testing.T) {
	rule := Rule{
		Type:        GENERIC_POLICY,
		Details:     "details",
		Mitigations: "mitigations",
		Status:      "fail",
	}
	expectedJSON := `{"Type":"GENERIC_POLICY","Details":"details","Mitigations":"mitigations","Status":"fail"}`

	bytes, err := rule.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	if string(bytes) != expectedJSON {
		t.Errorf("Expected JSON %s, got %s", expectedJSON, string(bytes))
	}
}

func TestGenericPolicyRule_Methods(t *testing.T) {
	rule := GenericPolicyRule{
		Type:        GENERIC_POLICY,
		Description: "description",
	}

	if rule.Describe() != "description" {
		t.Errorf("Describe() = %v, want %v", rule.Describe(), "description")
	}

	if rule.GetType() != GENERIC_POLICY {
		t.Errorf("GetType() = %v, want %v", rule.GetType(), GENERIC_POLICY)
	}

	if rule.Details() != "" {
		t.Errorf("Details() = %v, want empty string", rule.Details())
	}

	if rule.Test(nil) != nil {
		t.Errorf("Test(nil) = %v, want nil", rule.Test(nil))
	}
}

func TestForbidPrivilegedPorts_getPortFromEnv(t *testing.T) {
	rule := &ForbidPrivilegedPorts{}

	directives := map[string][]DfDirective{
		"env": {
			&EnvDirective{
				Variables: map[string]string{
					"PORT_80":  "80",
					"PORT_STR": "invalid",
				},
			},
		},
	}

	// Case 1: Valid port in env
	port := rule.getPortFromEnv("PORT_80", directives)
	if port == nil || *port != 80 {
		t.Errorf("getPortFromEnv(PORT_80) = %v, want 80", port)
	}

	// Case 2: Invalid port value
	port = rule.getPortFromEnv("PORT_STR", directives)
	if port != nil {
		t.Errorf("getPortFromEnv(PORT_STR) = %v, want nil", port)
	}

	// Case 3: Missing env variable
	port = rule.getPortFromEnv("PORT_MISSING", directives)
	if port != nil {
		t.Errorf("getPortFromEnv(PORT_MISSING) = %v, want nil", port)
	}
}

func TestForbidPackages_getInstalledPackages(t *testing.T) {
	rule := &ForbidPackages{}

	tests := []struct {
		name     string
		commands [][]string
		want     []string
	}{
		{
			name: "apt-get install",
			commands: [][]string{
				{"apt-get", "install", "-y", "vim"},
			},
			want: []string{"vim"},
		},
		{
			name: "apt-get install multiple",
			commands: [][]string{
				{"apt-get", "install", "-y", "vim", "curl"},
			},
			want: []string{"vim", "curl"},
		},
		{
			name: "apk add",
			commands: [][]string{
				{"apk", "add", "--no-cache", "git"},
			},
			want: []string{"git"},
		},
		{
			name: "yum install and remove",
			commands: [][]string{
				{"yum", "install", "-y", "wget"},
				{"yum", "remove", "-y", "wget"},
			},
			want: []string{},
		},
		{
			name: "mixed commands",
			commands: [][]string{
				{"apt-get", "update"},
				{"apt-get", "install", "net-tools"},
			},
			want: []string{"net-tools"},
		},
		{
			name: "flags in package list",
			commands: [][]string{
				{"apt-get", "install", "vim", "-q"},
			},
			want: []string{"vim"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.getInstalledPackages(tt.commands)
			if !reflect.DeepEqual(got, tt.want) {
				if len(got) == 0 && len(tt.want) == 0 {
					return
				}
				t.Errorf("getInstalledPackages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForbidPackages_splitSingleCommands(t *testing.T) {
	rule := &ForbidPackages{}

	tests := []struct {
		name      string
		directive *RunDirective
		want      [][]string
	}{
		{
			name: "Simple command",
			directive: &RunDirective{
				Content: "RUN apt-get update",
			},
			want: [][]string{
				{"RUN", "apt-get", "update"},
			},
		},
		{
			name: "Command with &&",
			directive: &RunDirective{
				Content: "RUN apt-get update && apt-get install vim",
			},
			want: [][]string{
				{"RUN", "apt-get", "update"},
				{"apt-get", "install", "vim"},
			},
		},
		{
			name: "Command with ;",
			directive: &RunDirective{
				Content: "RUN echo hello ; echo world",
			},
			want: [][]string{
				{"RUN", "echo", "hello"},
				{"echo", "world"},
			},
		},
		{
			name: "Command with pipe",
			directive: &RunDirective{
				Content: "RUN cat file | grep something",
			},
			want: [][]string{
				{"RUN", "cat", "file"},
				{"grep", "something"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directives := []DfDirective{tt.directive}
			got := rule.splitSingleCommands(directives)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitSingleCommands() = %v, want %v", got, tt.want)
			}
		})
	}
}
