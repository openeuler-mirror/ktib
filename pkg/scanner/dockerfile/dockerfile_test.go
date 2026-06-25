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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNewDockerfile(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func() string
		wantErr bool
		errMsg  string
		check   func(*Dockerfile)
	}{
		{
			name: "Valid Dockerfile",
			setup: func() string {
				content := `
FROM ubuntu:latest
LABEL maintainer="John Doe"
RUN apt-get update && apt-get install -y curl
`
				path := filepath.Join(tmpDir, "valid_dockerfile")
				err := os.WriteFile(path, []byte(content), 0644)
				if err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr: false,
			check: func(d *Dockerfile) {
				if len(d.Directives) != 3 {
					t.Errorf("expected 3 directives, got %d", len(d.Directives))
				}
				if d.GetMaintainers() != "John Doe" {
					t.Errorf("expected maintainer 'John Doe', got %s", d.GetMaintainers())
				}
			},
		},
		{
			name: "Non-existent Dockerfile",
			setup: func() string {
				return filepath.Join(tmpDir, "non_existent")
			},
			wantErr: true,
			errMsg:  "read dockerfile",
		},
		{
			name: "Empty Dockerfile",
			setup: func() string {
				path := filepath.Join(tmpDir, "empty_dockerfile")
				err := os.WriteFile(path, []byte{}, 0644)
				if err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr: true,
			errMsg:  "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			got, err := NewDockerfile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDockerfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewDockerfile() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
			if !tt.wantErr && got != nil && tt.check != nil {
				tt.check(got)
			}
		})
	}
}

func TestDockerfile_GetFilename(t *testing.T) {
	// 测试用例 1: 检查 GetFilename 函数是否能正确返回文件名
	dockerfile := &Dockerfile{
		Filename: "Dockerfile",
	}
	if filename := dockerfile.GetFilename(); filename != "Dockerfile" {
		t.Errorf("预期文件名为 'Dockerfile'，实际为 %s", filename)
	}
}

func TestDockerfile_GetPath(t *testing.T) {
	// 测试用例 1: 检查 GetPath 函数是否能正确返回文件路径
	dockerfile := &Dockerfile{
		Path: "/path/to/Dockerfile",
	}
	if path := dockerfile.GetPath(); path != "/path/to/Dockerfile" {
		t.Errorf("预期路径为 '/path/to/Dockerfile'，实际为 %s", path)
	}
}

func TestDockerfile_AddDirective(t *testing.T) {
	// 测试用例 1: 检查 AddDirective 函数是否能正确添加指令
	dockerfile := &Dockerfile{}
	directive1 := &RunDirective{
		Type:         RUN,
		Content:      "apt-get update",
		RunLastStage: []map[string]string{},
	}
	directive2 := &RunDirective{
		Type:         RUN,
		Content:      "apt-get install -y curl",
		RunLastStage: []map[string]string{},
	}
	directive3 := &ExposeDirective{
		Type:         EXPOSE,
		Content:      "8080",
		RunLastStage: []map[string]string{},
		Ports:        []string{"8080"},
	}
	directive4 := &MaintainerDirective{
		Type:         MAINTAINER,
		Content:      "Alice",
		RunLastStage: nil,
		Maintainers:  []string{"Alice"},
	}
	directive5 := &AddDirective{
		Type:         ADD,
		Content:      "./src /root",
		RunLastStage: nil,
		Chown:        "",
		Source:       "./src",
		Destination:  "/root",
	}
	directive6 := &CopyDirective{
		Type:         COPY,
		Content:      "./src /root",
		RunLastStage: nil,
		Chown:        "",
		Source:       "./src",
		Destination:  "/root",
	}
	directive7 := &EnvDirective{
		Type:         ENV,
		Content:      "https_proxy=nil",
		RunLastStage: nil,
		Variables:    map[string]string{"https_proxy": "nil"},
	}
	directive8 := &CmdDirective{
		Type:         CMD,
		Content:      "/bin/bash",
		RunLastStage: nil,
	}
	directive9 := &EntrypointDirective{
		Type:         ENTRYPOINT,
		Content:      "/bin/bash",
		RunLastStage: nil,
	}
	directive10 := &WorkdirDirective{
		Type:         WORKDIR,
		Content:      "/",
		RunLastStage: nil,
	}
	directive11 := &VolumeDirective{
		Type:         VOLUME,
		Content:      "/data",
		RunLastStage: nil,
		Volumes:      []string{"/data"},
	}
	directive12 := &StopsignalDirective{
		Type:         STOPSIGNAL,
		Content:      "SIGTERM",
		RunLastStage: nil,
		Signal:       "SIGTERM",
	}
	directive13 := &ArgDirective{
		Type:         ARG,
		Content:      "version=v1.0",
		RunLastStage: nil,
		Argument:     "version=v1.0",
	}
	directives := []DfDirective{
		directive1,
		directive2,
		directive3,
		directive4,
		directive5,
		directive6,
		directive7,
		directive8,
		directive9,
		directive10,
		directive11,
		directive12,
		directive13,
	}
	for _, directive := range directives {
		dockerfile.AddDirective(directive)
	}
	if len(dockerfile.Directives) != 13 {
		t.Errorf("预期指令数为 13，实际为 %d", len(dockerfile.Directives))
	}
	// 检查是否添加了所有的指令
	for i, d := range dockerfile.Directives {
		switch d := d.(type) {
		case *RunDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *ExposeDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *MaintainerDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *AddDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *CopyDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *EnvDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *CmdDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *EntrypointDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *WorkdirDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *VolumeDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *StopsignalDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		case *ArgDirective:
			if !reflect.DeepEqual(d, directives[i]) {
				t.Errorf("预期第 %d 个指令为 %+v，实际为 %+v", i+1, directives[i], d)
			}
		default:
			t.Errorf("未知的指令类型: %T", d)
		}
	}
}

func TestDockerfile_GetRunDirectivesLastStage(t *testing.T) {
	// 测试用例 1: 检查 GetRunDirectivesLastStage 函数是否能正确返回最后一个阶段的 RUN 指令
	dockerfile := &Dockerfile{
		Directives: []DfDirective{
			&FromDirective{
				Type:           FROM,
				Content:        "FROM golang:1.22 AS builder",
				RunLastStage:   []map[string]string{},
				Registry:       "",
				ImageLocalName: "builder",
				ImageTag:       "1.22",
				ImageName:      "golang",
			},
			&RunDirective{
				Type:         RUN,
				Content:      "apt-get update",
				RunLastStage: []map[string]string{},
			},
			&RunDirective{
				Type:         RUN,
				Content:      "apt-get install -y curl",
				RunLastStage: []map[string]string{},
			},
			&FromDirective{
				Type:           FROM,
				Content:        "FROM ubuntu:latest",
				RunLastStage:   []map[string]string{},
				Platform:       "",
				Registry:       "",
				ImageLocalName: "",
				ImageTag:       "latest",
				ImageName:      "ubuntu",
			},
			&RunDirective{
				Type:         RUN,
				Content:      "echo 'Hello, World!'",
				RunLastStage: []map[string]string{},
			},
		},
	}

	runDirectives := dockerfile.GetRunDirectivesLastStage()

	if len(runDirectives) != 1 {
		t.Errorf("GetRunDirectiesLastStage() received 1 instruction count, but it was actually %d", len(runDirectives))
	}
	runDirective := runDirectives[0].(*RunDirective)
	if runDirective.Content != "echo 'Hello, World!'" {
		t.Errorf("预期最后一个阶段的 RUN 指令为 'echo 'Hello, World!''，实际为 %+v", runDirective)
	}
}

func TestDockerfile_GetDirectives(t *testing.T) {
	// 创建一个示例 Dockerfile
	df := &Dockerfile{
		Directives: []DfDirective{
			&FromDirective{
				Type:      FROM,
				Content:   "FROM ubuntu:latest",
				ImageName: "ubuntu",
				ImageTag:  "latest",
			},
			&RunDirective{
				Type:    RUN,
				Content: "apt-get update && apt-get install -y nodejs",
			},
			&RunDirective{
				Type:    RUN,
				Content: "npm install -g express",
			},
			&UserDirective{
				Type: USER,
				User: "1000",
			},
			&LabelDirective{
				Type: LABEL,
				Labels: map[string]string{
					"app": "my-app",
					"env": "production",
				},
			},
			&ExposeDirective{
				Type:         EXPOSE,
				Content:      "8080",
				RunLastStage: []map[string]string{},
				Ports:        []string{"8080"},
			},
			&MaintainerDirective{
				Type:         MAINTAINER,
				Content:      "Alice",
				RunLastStage: nil,
				Maintainers:  []string{"Alice"},
			},
			&AddDirective{
				Type:         ADD,
				Content:      "./src /root",
				RunLastStage: nil,
				Chown:        "",
				Source:       "./src",
				Destination:  "/root",
			},
			&CopyDirective{
				Type:         COPY,
				Content:      "./src /root",
				RunLastStage: nil,
				Chown:        "",
				Source:       "./src",
				Destination:  "/root",
			},
			&EnvDirective{
				Type:         ENV,
				Content:      "https_proxy=nil",
				RunLastStage: nil,
				Variables:    map[string]string{"https_proxy": "nil"},
			},
			&CmdDirective{
				Type:         CMD,
				Content:      "/bin/bash",
				RunLastStage: nil,
			},
			&EntrypointDirective{
				Type:         ENTRYPOINT,
				Content:      "/bin/bash",
				RunLastStage: nil,
			},
			&WorkdirDirective{
				Type:         WORKDIR,
				Content:      "/",
				RunLastStage: nil,
			},
			&VolumeDirective{
				Type:         VOLUME,
				Content:      "/data",
				RunLastStage: nil,
				Volumes:      []string{"/data"},
			},
			&StopsignalDirective{
				Type:         STOPSIGNAL,
				Content:      "SIGTERM",
				RunLastStage: nil,
				Signal:       "SIGTERM",
			},
			&ArgDirective{
				Type:         ARG,
				Content:      "version=v1.0",
				RunLastStage: nil,
				Argument:     "version=v1.0",
			},
		},
	}

	// 调用 GetDirectives() 函数并检查结果
	result := df.GetDirectives()
	expected := map[string][]DfDirective{
		"from":           {df.Directives[0]},
		"run":            {df.Directives[1], df.Directives[2]},
		"user":           {df.Directives[3]},
		"labels":         {df.Directives[4]},
		"expose":         {df.Directives[5]},
		"maintainers":    {df.Directives[6]},
		"add":            {df.Directives[7]},
		"copy":           {df.Directives[8]},
		"env":            {df.Directives[9]},
		"cmd":            {df.Directives[10]},
		"entrypoint":     {df.Directives[11]},
		"workdir":        {df.Directives[12]},
		"volume":         {df.Directives[13]},
		"stopsignal":     {df.Directives[14]},
		"arg":            {df.Directives[15]},
		"run_last_stage": df.GetRunDirectivesLastStage(),
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GetDirectives() returned unexpected result:\n got: %v\nwant: %v", result, expected)
	}
}

func TestGetRaw(t *testing.T) {
	// 创建一个示例 Dockerfile
	df := &Dockerfile{
		Directives: []DfDirective{
			&AddDirective{
				Type:         ADD,
				Content:      "app.tar.gz /app",
				RunLastStage: []map[string]string{{"user": "1000"}},
				Chown:        "1000:1000",
				Source:       "app.tar.gz",
				Destination:  "/app",
			},
			&CopyDirective{
				Type:         COPY,
				Content:      "requirements.txt /app",
				RunLastStage: []map[string]string{{"user": "1000"}},
				Chown:        "1000:1000",
				Source:       "requirements.txt",
				Destination:  "/app",
			},
		},
	}

	// 调用 GetRaw() 函数并检查结果
	raw := df.GetRaw()
	expected := []map[string]interface{}{
		{
			"type":        "ADD",
			"raw_content": "app.tar.gz /app",
			"chown":       "1000:1000",
			"source":      "app.tar.gz",
			"destination": "/app",
		},
		{
			"type":        "COPY",
			"raw_content": "requirements.txt /app",
			"chown":       "1000:1000",
			"source":      "requirements.txt",
			"destination": "/app",
		},
	}

	if !reflect.DeepEqual(raw, expected) {
		t.Errorf("GetRaw() returned unexpected result:\n got: %v\nwant: %v", raw, expected)
	}
}

func TestGetMaintainers(t *testing.T) {
	testCases := []struct {
		name       string
		directives []DfDirective
		expected   string
	}{
		{
			name: "Single maintainer in LABEL",
			directives: []DfDirective{
				&LabelDirective{
					Type:   LABEL,
					Labels: map[string]string{"maintainer": "John Doe"},
				},
			},
			expected: "John Doe",
		},
		{
			name: "Maintainer in MAINTAINER directive",
			directives: []DfDirective{
				&MaintainerDirective{
					Type:        MAINTAINER,
					Content:     "MAINTAINER John Doe, Jane Smith",
					Maintainers: []string{"John Doe", "Jane Smith"},
				},
			},
			expected: "John Doe, Jane Smith",
		},
		{
			name:       "No maintainer found",
			directives: []DfDirective{},
			expected:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			df := &Dockerfile{
				Directives: tc.directives,
			}

			maintainers := df.GetMaintainers()
			if maintainers != tc.expected {
				t.Errorf("GetMaintainers() returned %q, expected %q", maintainers, tc.expected)
			}
		})
	}
}
