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
	"fmt"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"log"
	"os"
)

var logger *log.Logger

type DockerfileObject struct {
	From               string
	Platform           string
	Registry           string
	ImageName          string
	ImageTag           string
	Digest             string
	LocalName          string
	User               string
	RunCommands        []string
	LabelCommands      []string
	ExposeCommands     []string
	MaintainerCommands []string
	AddCommands        []string
	CopyCommands       []string
	EnvCommands        []string
	CmdCommands        []string
	EntrypointCommands []string
	WorkdirCommand     string
	VolumeCommands     []string
	ShellCommand       string
	StopsignalCommand  string
	ArgCommands        []string
	HealthcheckCommand string
	HealthcheckOptions string
}

type ParseResult struct {
	Filename    string                   `json:"filename"`
	Path        string                   `json:"path"`
	Maintainers string                   `json:"maintainers"`
	Directives  map[string][]DfDirective `json:"directives"`
}


type DockerfileVisitor struct {
	Dockerfile *Dockerfile
}

func NewDockerfileVisitor(dockerfile *Dockerfile) *DockerfileVisitor {
	return &DockerfileVisitor{
		Dockerfile: dockerfile,
	}
}

func (v *DockerfileVisitor) VisitDockerfile(visitedChildren *parser.Node) interface{} {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	for _, parsedLine := range visitedChildren.Children {
		lineType := parseDirectiveType(parsedLine.Value)
		var lineContent string
		if parsedLine.Next != nil {
			lineContent = parsedLine.Next.Dump()
		}
		switch lineType {
		case FROM:
			v.Dockerfile.AddDirective(NewFromDirective(lineContent))
		case USER:
			v.Dockerfile.AddDirective(NewUserDirective(lineContent))
		case RUN:
			v.Dockerfile.AddDirective(NewRunDirective(lineContent))
		case LABEL:
			v.Dockerfile.AddDirective(NewLabelDirective(lineContent))
		case EXPOSE:
			v.Dockerfile.AddDirective(NewExposeDirective(lineContent))
		case MAINTAINER:
			v.Dockerfile.AddDirective(NewMaintainerDirective(lineContent))
		case ADD:
			v.Dockerfile.AddDirective(NewAddDirective(lineContent))
		case COPY:
			v.Dockerfile.AddDirective(NewCopyDirective(lineContent))
		case ENV:
			v.Dockerfile.AddDirective(NewEnvDirective(lineContent))
		case ENTRYPOINT:
			v.Dockerfile.AddDirective(NewEntrypointDirective(lineContent))
		case WORKDIR:
			v.Dockerfile.AddDirective(NewWorkdirDirective(lineContent))
		case VOLUME:
			v.Dockerfile.AddDirective(NewVolumeDirective(lineContent))
		case STOPSIGNAL:
			v.Dockerfile.AddDirective(NewStopsignalDirective(lineContent))
		case ARG:
			v.Dockerfile.AddDirective(NewArgDirective(lineContent))
		case CMD:
			v.Dockerfile.AddDirective(NewCmdDirective(lineContent))
		default:
			logger.Println(fmt.Sprintf("Directive type not recognized or not implemented yet: %v", lineType))
			continue
		}
	}
	return v.Dockerfile
}

func parseDirectiveType(name string) DockerfileDirectiveType {
	switch name {
	case "FROM":
		return FROM
	case "USER":
		return USER
	case "RUN":
		return RUN
	case "LABEL":
		return LABEL
	case "EXPOSE":
		return EXPOSE
	case "MAINTAINER":
		return MAINTAINER
	case "ADD":
		return ADD
	case "COPY":
		return COPY
	case "ENV":
		return ENV
	case "ENTRYPOINT":
		return ENTRYPOINT
	case "WORKDIR":
		return WORKDIR
	case "VOLUME":
		return VOLUME
	case "STOPSIGNAL":
		return STOPSIGNAL
	case "ARG":
		return ARG
	case "CMD":
		return CMD
	}
	return 0
}
