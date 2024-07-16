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
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"reflect"
	"testing"
)

func TestNewDockerfileVisitor(t *testing.T) {
	dockerfile := &Dockerfile{}
	visitor := NewDockerfileVisitor(dockerfile)
	if visitor.Dockerfile != dockerfile {
		t.Errorf("NewDockerfileVisitor() returned unexpected Dockerfile object")
	}
}

func TestVisitDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		input    *parser.Node
		expected *Dockerfile
	}{
		{
			name: "Dockerfile",
			input: &parser.Node{
				Children: []*parser.Node{
					{
						Value: "FROM",
						Next: &parser.Node{
							Value: "ubuntu:latest",
						},
					},
					{
						Value: "RUN",
						Next: &parser.Node{
							Value: "apt-get update && apt-get install -y nginx",
						},
					},
					{
						Value: "LABEL",
						Next: &parser.Node{
							Value: "maintainer=\"example user <user@example.com>\"",
						},
					},
					{
						Value: "USER",
						Next: &parser.Node{
							Value: "admin",
						},
					},
					{
						Value: "EXPOSE",
						Next: &parser.Node{
							Value: "8080",
						},
					},
					{
						Value: "MAINTAINER",
						Next: &parser.Node{
							Value: "alice",
						},
					},
					{
						Value: "ADD",
						Next: &parser.Node{
							Value: "src /des",
						},
					},
					{
						Value: "COPY",
						Next: &parser.Node{
							Value: "src /des",
						},
					},
					{
						Value: "ENV",
						Next: &parser.Node{
							Value: "version=v0.1",
						},
					},
					{
						Value: "ENTRYPOINT",
						Next: &parser.Node{
							Value: "/bin/bash",
						},
					},
					{
						Value: "WORKDIR",
						Next: &parser.Node{
							Value: "/root",
						},
					},
					{
						Value: "VOLUME",
						Next: &parser.Node{
							Value: "/hostpath",
						},
					},
					{
						Value: "STOPSIGNAL",
						Next: &parser.Node{
							Value: "3",
						},
					},
					{
						Value: "ARG",
						Next: &parser.Node{
							Value: "arch=x86",
						},
					},
					{
						Value: "CMD",
						Next: &parser.Node{
							Value: "/bin/bash",
						},
					},
				},
			},
			expected: &Dockerfile{
				Directives: []DfDirective{
					NewFromDirective("ubuntu:latest"),
					NewRunDirective("apt-get update && apt-get install -y nginx"),
					NewLabelDirective("maintainer=\"example user <user@example.com>\""),
					NewUserDirective("admin"),
					NewExposeDirective("8080"),
					NewMaintainerDirective("alice"),
					NewAddDirective("src /des"),
					NewCopyDirective("src /des"),
					NewEnvDirective("version=v0.1"),
					NewEntrypointDirective("/bin/bash"),
					NewWorkdirDirective("/root"),
					NewVolumeDirective("/hostpath"),
					NewStopsignalDirective("3"),
					NewArgDirective("arch=x86"),
					NewCmdDirective("/bin/bash"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := NewDockerfileVisitor(&Dockerfile{})
			result := visitor.VisitDockerfile(tt.input).(*Dockerfile)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("VisitDockerfile() returned unexpected result.\nGot: %+v\nExpected: %+v", result, tt.expected)
			}
		})
	}
}
