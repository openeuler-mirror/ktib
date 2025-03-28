#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

package dockerfile

import (
	"reflect"
	"testing"
)

func TestNewFromDirective(t *testing.T) {
	testCases := []struct {
		name         string
		rawContent   string
		expectedFrom *FromDirective
	}{
		{
			name:       "with registry and tag",
			rawContent: "registry.example.com/myapp/my-image:v1.0",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "registry.example.com/myapp/my-image:v1.0",
				Registry:  "registry.example.com",
				ImageName: "my-image",
				ImageTag:  "v1.0",
			},
		},
		{
			name:       "with registry and platform",
			rawContent: "registry.example.com/my-image@amd64",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "registry.example.com/my-image@amd64",
				Registry:  "registry.example.com",
				ImageName: "my-image",
				Platform:  "amd64",
			},
		},
		{
			name:       "without registry, with tag",
			rawContent: "my-image:v2.0",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "my-image:v2.0",
				ImageName: "my-image",
				ImageTag:  "v2.0",
			},
		},
		{
			name:       "without registry, with platform",
			rawContent: "my-image@arm64",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "my-image@arm64",
				ImageName: "my-image",
				Platform:  "arm64",
			},
		},
		{
			name:       "without registry, tag, or platform",
			rawContent: "my-image",
			expectedFrom: &FromDirective{
				Type:      FROM,
				Content:   "my-image",
				ImageName: "my-image",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			from := NewFromDirective(tc.rawContent)
			if !reflect.DeepEqual(from, tc.expectedFrom) {
				t.Errorf("Expected %+v, got %+v", tc.expectedFrom, from)
			}
			getMsg := from.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "FROM" {
				t.Errorf("Expected type to be FROM, got %s", getMsg["type"])
			}
			getTypeMsg := from.GetType()
			if getTypeMsg.String() != "FROM" {
				t.Errorf("Expected type to be FROM, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewRunDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *RunDirective
	}{
		{
			name:       "with single argument",
			rawContent: "RUN echo 'hello world'",
			expected: &RunDirective{
				Type:    RUN,
				Content: "RUN echo 'hello world'",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run := NewRunDirective(tc.rawContent)
			if !reflect.DeepEqual(run, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, run)
			}
			getMsg := run.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "RUN" {
				t.Errorf("Expected type to be RUN, got %s", getMsg["type"])
			}
			getTypeMsg := run.GetType()
			if getTypeMsg.String() != "RUN" {
				t.Errorf("Expected type to be RUN, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewLabelDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *LabelDirective
	}{
		{
			name:       "with single label",
			rawContent: "LABEL maintainer=\"john@example.com\"",
			expected: &LabelDirective{
				Type:    LABEL,
				Content: "LABEL maintainer=\"john@example.com\"",
				Labels: map[string]string{
					"maintainer": "john@example.com",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			label := NewLabelDirective(tc.rawContent)
			if !reflect.DeepEqual(label, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, label)
			}
			getMsg := label.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "LABEL" {
				t.Errorf("Expected type to be LABEL, got %s", getMsg["type"])
			}
			getTypeMsg := label.GetType()
			if getTypeMsg.String() != "LABEL" {
				t.Errorf("Expected type to be LABEL, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewUserDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *UserDirective
	}{
		{
			name:       "with single user",
			rawContent: "USER john",
			expected: &UserDirective{
				Type:    USER,
				Content: "USER john",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user := NewUserDirective(tc.rawContent)
			if !reflect.DeepEqual(user, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, user)
			}
			getMsg := user.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "USER" {
				t.Errorf("Expected type to be USER, got %s", getMsg["type"])
			}
			getTypeMsg := user.GetType()
			if getTypeMsg.String() != "USER" {
				t.Errorf("Expected type to be USER, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewExposeDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *ExposeDirective
	}{
		{
			name:       "with single port",
			rawContent: "EXPOSE 80",
			expected: &ExposeDirective{
				Type:    EXPOSE,
				Content: "EXPOSE 80",
				Ports: []string{
					"80",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expose := NewExposeDirective(tc.rawContent)
			if !reflect.DeepEqual(expose, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, expose)
			}
			getMsg := expose.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "EXPOSE" {
				t.Errorf("Expected type to be EXPOSE, got %s", getMsg["type"])
			}
			getTypeMsg := expose.GetType()
			if getTypeMsg.String() != "EXPOSE" {
				t.Errorf("Expected type to be EXPOSE, got %s", getTypeMsg.String())
			}
		})
	}
}

//	func TestNewMaintainerDirective(t *testing.T) {
//		testCases := []struct {
//			name       string
//			rawContent string
//			expected   *MaintainerDirective
//		}{
//			{
//				name:       "with single maintainer",
//				rawContent: "MAINTAINER john@example.com",
//				expected: &MaintainerDirective{
//					Type:    MAINTAINER,
//					Content: "MAINTAINER john@example.com",
//					Maintainers: []string{
//						"MAINTAINER john@example.com",
//					},
//				},
//			},
//		}
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				maintainer := NewMaintainerDirective(tc.rawContent)
//				if !reflect.DeepEqual(maintainer, tc.expected) {
//					t.Errorf("Expected %+v, got %+v", tc.expected, maintainer)
//				}
//				getMsg := maintainer.Get()
//				if getMsg["raw_content"] != tc.rawContent {
//					t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
//				}
//				if getMsg["type"] != MAINTAINER {
//					t.Errorf("Expected type to be MAINTAINER, got %s", getMsg["type"])
//				}
//				getTypeMsg := maintainer.GetType()
//				if getTypeMsg.String() != "MAINTAINER" {
//					t.Errorf("Expected type to be MAINTAINER, got %s", getTypeMsg.String())
//				}
//			})
//		}
//	}
func TestNewAddDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *AddDirective
	}{
		{
			name:       "with single add",
			rawContent: "ADD /src/main.go /main.go",
			expected: &AddDirective{
				Type:        ADD,
				Content:     "ADD /src/main.go /main.go",
				Source:      "/src/main.go",
				Destination: "/main.go",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			add := NewAddDirective(tc.rawContent)
			if !reflect.DeepEqual(add, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, add)
			}
			getMsg := add.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "ADD" {
				t.Errorf("Expected type to be ADD, got %s", getMsg["type"])
			}
			getTypeMsg := add.GetType()
			if getTypeMsg.String() != "ADD" {
				t.Errorf("Expected type to be ADD, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewCopyDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *CopyDirective
	}{
		{
			name:       "with single copy",
			rawContent: "COPY /src/main.go /main.go",
			expected: &CopyDirective{
				Type:        COPY,
				Content:     "COPY /src/main.go /main.go",
				Source:      "/src/main.go",
				Destination: "/main.go",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCopy := NewCopyDirective(tc.rawContent)
			if !reflect.DeepEqual(testCopy, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, testCopy)
			}
			getMsg := testCopy.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "COPY" {
				t.Errorf("Expected type to be COPY, got %s", getMsg["type"])
			}
			getTypeMsg := testCopy.GetType()
			if getTypeMsg.String() != "COPY" {
				t.Errorf("Expected type to be COPY, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewEnvDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *EnvDirective
	}{
		{
			name:       "with single env",
			rawContent: "ENV VAR1=value1",
			expected: &EnvDirective{
				Type:    ENV,
				Content: "ENV VAR1=value1",
				Variables: map[string]string{
					"ENV":  "VAR1=value1",
					"VAR1": "value1",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := NewEnvDirective(tc.rawContent)
			if !reflect.DeepEqual(env, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, env)
			}
			getMsg := env.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "ENV" {
				t.Errorf("Expected type to be ENV, got %s", getMsg["type"])
			}
			getTypeMsg := env.GetType()
			if getTypeMsg.String() != "ENV" {
				t.Errorf("Expected type to be ENV, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewCmdDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *CmdDirective
	}{
		{
			name:       "with single cmd",
			rawContent: "CMD echo 'hello world'",
			expected: &CmdDirective{
				Type:    CMD,
				Content: "CMD echo 'hello world'",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCmdDirective(tc.rawContent)
			if !reflect.DeepEqual(cmd, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, cmd)
			}
			getMsg := cmd.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "CMD" {
				t.Errorf("Expected type to be CMD, got %s", getMsg["type"])
			}
			getTypeMsg := cmd.GetType()
			if getTypeMsg.String() != "CMD" {
				t.Errorf("Expected type to be CMD, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewEntrypointDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *EntrypointDirective
	}{
		{
			name:       "with single entrypoint",
			rawContent: "ENTRYPOINT echo 'hello world'",
			expected: &EntrypointDirective{
				Type:    ENTRYPOINT,
				Content: "ENTRYPOINT echo 'hello world'",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entrypoint := NewEntrypointDirective(tc.rawContent)
			if !reflect.DeepEqual(entrypoint, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, entrypoint)
			}
			getMsg := entrypoint.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "ENTRYPOINT" {
				t.Errorf("Expected type to be ENTRYPOINT, got %s", getMsg["type"])
			}
			getTypeMsg := entrypoint.GetType()
			if getTypeMsg.String() != "ENTRYPOINT" {
				t.Errorf("Expected type to be ENTRYPOINT, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewWorkdirDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *WorkdirDirective
	}{
		{
			name:       "with single workdir",
			rawContent: "WORKDIR /app",
			expected: &WorkdirDirective{
				Type:    WORKDIR,
				Content: "WORKDIR /app",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workdir := NewWorkdirDirective(tc.rawContent)
			if !reflect.DeepEqual(workdir, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, workdir)
			}
			getMsg := workdir.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "WORKDIR" {
				t.Errorf("Expected type to be WORKDIR, got %s", getMsg["type"])
			}
			getTypeMsg := workdir.GetType()
			if getTypeMsg.String() != "WORKDIR" {
				t.Errorf("Expected type to be WORKDIR, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewVolumeDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *VolumeDirective
	}{
		{
			name:       "with single volume",
			rawContent: "VOLUME /data",
			expected: &VolumeDirective{
				Type:    VOLUME,
				Content: "VOLUME /data",
				Volumes: []string{"VOLUME", "/data"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			volume := NewVolumeDirective(tc.rawContent)
			if !reflect.DeepEqual(volume, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, volume)
			}
			getMsg := volume.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "VOLUME" {
				t.Errorf("Expected type to be VOLUME, got %s", getMsg["type"])
			}
			getTypeMsg := volume.GetType()
			if getTypeMsg.String() != "VOLUME" {
				t.Errorf("Expected type to be VOLUME, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewShellDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *ShellDirective
	}{
		{
			name:       "with single shell",
			rawContent: "SHELL [\"bash\", \"-c\"]",
			expected: &ShellDirective{
				Type:    SHELL,
				Content: "SHELL [\"bash\", \"-c\"]",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shell := NewShellDirective(tc.rawContent)
			if !reflect.DeepEqual(shell, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, shell)
			}
			getMsg := shell.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "SHELL" {
				t.Errorf("Expected type to be SHELL, got %s", getMsg["type"])
			}
			getTypeMsg := shell.GetType()
			if getTypeMsg.String() != "SHELL" {
				t.Errorf("Expected type to be SHELL, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewStopsignalDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *StopsignalDirective
	}{
		{
			name:       "with single stopsignal",
			rawContent: "STOPSIGNAL SIGTERM",
			expected: &StopsignalDirective{
				Type:    STOPSIGNAL,
				Content: "STOPSIGNAL SIGTERM",
				Signal:  "SIGTERM",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stopsignal := NewStopsignalDirective(tc.rawContent)
			if !reflect.DeepEqual(stopsignal, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, stopsignal)
			}
			getMsg := stopsignal.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "STOPSIGNAL" {
				t.Errorf("Expected type to be STOPSIGNAL, got %s", getMsg["type"])
			}
			getTypeMsg := stopsignal.GetType()
			if getTypeMsg.String() != "STOPSIGNAL" {
				t.Errorf("Expected type to be STOPSIGNAL, got %s", getTypeMsg.String())
			}
		})
	}
}
func TestNewArgDirective(t *testing.T) {
	testCases := []struct {
		name       string
		rawContent string
		expected   *ArgDirective
	}{
		{
			name:       "with single arg",
			rawContent: "ARG VAR=value",
			expected: &ArgDirective{
				Type:     ARG,
				Content:  "ARG VAR=value",
				Argument: "ARG VAR=value",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			arg := NewArgDirective(tc.rawContent)
			if !reflect.DeepEqual(arg, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, arg)
			}
			getMsg := arg.Get()
			if getMsg["raw_content"] != tc.rawContent {
				t.Errorf("Expected raw_content to be %s, got %s", tc.rawContent, getMsg["raw_content"])
			}
			if getMsg["type"] != "ARG" {
				t.Errorf("Expected type to be ARG, got %s", getMsg["type"])
			}
			getTypeMsg := arg.GetType()
			if getTypeMsg.String() != "ARG" {
				t.Errorf("Expected type to be ARG, got %s", getTypeMsg.String())
			}
		})
	}
}
