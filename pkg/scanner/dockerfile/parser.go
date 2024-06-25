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
	"regexp"
	"strings"
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

func ParseDockerfile(input string) (*DockerfileObject, error) {
	dockerfile := &DockerfileObject{}

	re := regexp.MustCompile(`(?m)\s+`)
	input = re.ReplaceAllString(input, " ")

	match, _ := regexp.MatchString(`(?i)from`, input)
	if match {
		fromMatches := regexp.MustCompile(`(?i)from\s+(--platform=[\S]+\s+)?([\S]+)?(/)?([\S]+)?(:)?([\S]+)?(@)?([\S]+)?( AS )?([\S]+)?`).FindStringSubmatch(input)
		dockerfile.From = strings.TrimSpace(fromMatches[0])
		dockerfile.Platform = strings.TrimSpace(fromMatches[1])
		dockerfile.Registry = strings.TrimSpace(fromMatches[2])
		dockerfile.ImageName = strings.TrimSpace(fromMatches[3])
		dockerfile.ImageTag = strings.TrimSpace(fromMatches[5])
		dockerfile.Digest = strings.TrimSpace(fromMatches[7])
		dockerfile.LocalName = strings.TrimSpace(fromMatches[9])
	}

	match, _ = regexp.MatchString(`(?i)user`, input)
	if match {
		userMatches := regexp.MustCompile(`(?i)user\s+([a-zA-Z0-9_-]+)?:?([a-zA-Z0-9_-]+)?`).FindStringSubmatch(input)
		dockerfile.User = strings.TrimSpace(userMatches[0])
	}

	match, _ = regexp.MatchString(`(?i)run`, input)
	if match {
		runMatches := regexp.MustCompile(`(?i)run\s+(?:\"([^\"\\\\]*(?:\\\\.[^\"\\\\]*)*)\"|'([^'\\\\]*(?:\\\\.[^'\\\\]*)*)'|([^\"'\\\\]*(?:\\\\.[^\"'\\\\]*)*))`).FindAllStringSubmatch(input, -1)
		for _, match := range runMatches {
			dockerfile.RunCommands = append(dockerfile.RunCommands, strings.TrimSpace(match[0]))
		}
	}

	match, _ = regexp.MatchString(`(?i)label`, input)
	if match {
		labelMatches := regexp.MustCompile(`(?i)label\s+([\S\s]+?)\s+`).FindAllStringSubmatch(input, -1)
		for _, match := range labelMatches {
			dockerfile.LabelCommands = append(dockerfile.LabelCommands, strings.TrimSpace(match[1]))
		}
	}

	match, _ = regexp.MatchString(`(?i)expose`, input)
	if match {
		exposeMatches := regexp.MustCompile(`(?i)expose\s+([\S\s]+?)\s+`).FindAllStringSubmatch(input, -1)
		for _, match := range exposeMatches {
			dockerfile.ExposeCommands = append(dockerfile.ExposeCommands, strings.TrimSpace(match[1]))
		}
	}

	match, _ = regexp.MatchString(`(?i)maintainer`, input)
	if match {
		maintainerMatches := regexp.MustCompile(`(?i)maintainer\s+([\S\s]+?)\s+`).FindAllStringSubmatch(input, -1)
		for _, match := range maintainerMatches {
			dockerfile.MaintainerCommands = append(dockerfile.MaintainerCommands, strings.TrimSpace(match[1]))
		}
	}

	match, _ = regexp.MatchString(`(?i)add`, input)
	if match {
		addMatches := regexp.MustCompile(`(?i)add\s+([\S\s]+?)\s+`).FindAllStringSubmatch(input, -1)
		for _, match := range addMatches {
			dockerfile.AddCommands = append(dockerfile.AddCommands, strings.TrimSpace(match[1]))
		}
	}

	match, _ = regexp.MatchString(`(?i)copy`, input)
	if match {
		copyMatches := regexp.MustCompile(`(?i)copy\s+([\S\s]+?)\s+`).FindAllStringSubmatch(input, -1)
		for _, match := range copyMatches {
			dockerfile.CopyCommands = append(dockerfile.CopyCommands, strings.TrimSpace(match[1]))
		}
	}

	match, _ = regexp.MatchString(`(?i)env`, input)
	if match {
		envMatches := regexp.MustCompile(`(?i)env\s+([\S\s]+?)\s+`).FindAllStringSubmatch(input, -1)
		for _, match := range envMatches {
			dockerfile.EnvCommands = append(dockerfile.EnvCommands, strings.TrimSpace(match[1]))
		}
	}

	match, _ = regexp.MatchString(`(?i)cmd`, input)
	if match {
		cmdMatches := regexp.MustCompile(`(?i)cmd\s+(?:\"([^\"\\\\]*(?:\\\\.[^\"\\\\]*)*)\"|'([^'\\\\]*(?:\\\\.[^'\\\\]*)*)'|([^\"'\\\\]*(?:\\\\.[^\"'\\\\]*)*))`).FindAllStringSubmatch(input, -1)
		for _, match := range cmdMatches {
			dockerfile.CmdCommands = append(dockerfile.CmdCommands, strings.TrimSpace(match[0]))
		}
	}

	match, _ = regexp.MatchString(`(?i)entrypoint`, input)
	if match {
		entrypointMatches := regexp.MustCompile(`(?i)entrypoint\s+(?:\"([^\"\\\\]*(?:\\\\.[^\"\\\\]*)*)\"|'([^'\\\\]*(?:\\\\.[^'\\\\]*)*)'|([^\"'\\\\]*(?:\\\\.[^\"'\\\\]*)*))`).FindAllStringSubmatch(input, -1)
		for _, match := range entrypointMatches {
			dockerfile.EntrypointCommands = append(dockerfile.EntrypointCommands, strings.TrimSpace(match[0]))
		}
	}

	match, _ = regexp.MatchString(`(?i)workdir`, input)
	if match {
		workdirMatches := regexp.MustCompile(`(?i)workdir\s+(?:\"([^\"\\\\]*(?:\\\\.[^\"\\\\]*)*)\"|'([^'\\\\]*(?:\\\\.[^'\\\\]*)*)'|([^\"'\\\\]*(?:\\\\.[^\"'\\\\]*)*))`).FindStringSubmatch(input)
		dockerfile.WorkdirCommand = strings.TrimSpace(workdirMatches[0])
	}

	match, _ = regexp.MatchString(`(?i)volume`, input)
	if match {
		volumeMatches := regexp.MustCompile(`(?i)volume\s+(?:\"([^\"\\\\]*(?:\\\\.[^\"\\\\]*)*)\"|'([^'\\\\]*(?:\\\\.[^'\\\\]*)*)'|([^\"'\\\\]*(?:\\\\.[^\"'\\\\]*)*))`).FindAllStringSubmatch(input, -1)
		for _, match := range volumeMatches {
			dockerfile.VolumeCommands = append(dockerfile.VolumeCommands, strings.TrimSpace(match[0]))
		}
	}

	match, _ = regexp.MatchString(`(?i)shell`, input)
	if match {
		shellMatches := regexp.MustCompile(`(?i)shell\s+(?:\"([^\"\\\\]*(?:\\\\.[^\"\\\\]*)*)\"|'([^'\\\\]*(?:\\\\.[^'\\\\]*)*)'|([^\"'\\\\]*(?:\\\\.[^\"'\\\\]*)*))`).FindStringSubmatch(input)
		dockerfile.ShellCommand = strings.TrimSpace(shellMatches[0])
	}

	match, _ = regexp.MatchString(`(?i)stopsignal`, input)
	if match {
		stopsignalMatches := regexp.MustCompile(`(?i)stopsignal\s+([\S\s]+?)\s+`).FindStringSubmatch(input)
		dockerfile.StopsignalCommand = strings.TrimSpace(stopsignalMatches[1])
	}

	match, _ = regexp.MatchString(`(?i)arg`, input)
	if match {
		argMatches := regexp.MustCompile(`(?i)arg\s+(?:\"([^\"\\\\]*(?:\\\\.[^\"\\\\]*)*)\"|'([^'\\\\]*(?:\\\\.[^'\\\\]*)*)'|([^\"'\\\\]*(?:\\\\.[^\"'\\\\]*)*))`).FindAllStringSubmatch(input, -1)
		for _, match := range argMatches {
			dockerfile.ArgCommands = append(dockerfile.ArgCommands, strings.TrimSpace(match[0]))
		}
	}

	match, _ = regexp.MatchString(`(?i)healthcheck`, input)
	if match {
		healthcheckMatches := regexp.MustCompile(`(?i)healthcheck\s+([\S\s]+?)\s+`).FindStringSubmatch(input)
		dockerfile.HealthcheckCommand = strings.TrimSpace(healthcheckMatches[1])
	}

	match, _ = regexp.MatchString(`(?i)placeholder`, input)
	if match {
		healthcheckOptionsMatches := regexp.MustCompile(`(?i)placeholder`).FindStringSubmatch(input)
		dockerfile.HealthcheckOptions = strings.TrimSpace(healthcheckOptionsMatches[0])
	}

	return dockerfile, nil
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
