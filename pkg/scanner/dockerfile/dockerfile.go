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
	"bytes"
	"gitee.com/openeuler/ktib/pkg/scanner/parsingutils"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
)

type Error interface {
	error
}

type NotDockerfileError struct {
}

func (e *NotDockerfileError) Error() string {
	return "Not a Dockerfile error"
}

type EmptyFileError struct {
}

func (e *EmptyFileError) Error() string {
	return "Empty file error"
}

type Dockerfile struct {
	Directives  []DfDirective
	Path        string
	Filename    string
	Maintainers []string
}

func NewDockerfile(path string) (*Dockerfile, error) {
	dockerfile := &Dockerfile{
		Directives:  make([]DfDirective, 0),
		Path:        path,
		Filename:    filepath.Base(path),
		Maintainers: make([]string, 0),
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("%s does not exist or it is not a file.\n%s\n", dockerfile.Path, err)
		return nil, &NotDockerfileError{}
	}

	dockerfileContent := normalizeContent(string(content))
	if len(dockerfileContent) == 0 {
		return nil, &EmptyFileError{}
	}
	dockerfileReader := bytes.NewReader([]byte(dockerfileContent))
	vistor := NewDockerfileVisitor(dockerfile)
	parsedLines, err := parser.Parse(dockerfileReader)
	if err != nil {
		return nil, &EmptyFileError{}
	}

	vistor.VisitDockerfile(parsedLines.AST)
	return dockerfile, nil
}

func (df *Dockerfile) GetFilename() string {
	return df.Filename
}

func (df *Dockerfile) GetPath() string {
	return df.Path
}

func (df *Dockerfile) AddDirective(directive DfDirective) {
	df.Directives = append(df.Directives, directive)
}

func (df *Dockerfile) GetRunDirectivesLastStage() []DfDirective {
	directives := make([]DfDirective, len(df.Directives))
	copy(directives, df.Directives)
	runDirectives := make([]DfDirective, 0)
	for _, directive := range directives {
		if directive.GetType() == RUN {
			runDirectives = append(runDirectives, directive)
		}
		if directive.GetType() == FROM {
			break
		}
	}
	return runDirectives
}

func (df *Dockerfile) GetDirectives() map[string][]DfDirective {
	result := make(map[string][]DfDirective)
	for _, directive := range df.Directives {
		directiveType := strconv.Itoa(int(directive.GetType()))
		switch directiveType {
		case strconv.Itoa(FROM):
			result["from"] = append(result["from"], directive)
		case strconv.Itoa(USER):
			result["user"] = append(result["user"], directive)
		case strconv.Itoa(RUN):
			result["run"] = append(result["run"], directive)
		case strconv.Itoa(LABEL):
			result["labels"] = append(result["labels"], directive)
		case strconv.Itoa(EXPOSE):
			result["expose"] = append(result["expose"], directive)
		case strconv.Itoa(MAINTAINER):
			result["maintainers"] = append(result["maintainers"], directive)
		case strconv.Itoa(ADD):
			result["add"] = append(result["add"], directive)
		case strconv.Itoa(COPY):
			result["copy"] = append(result["copy"], directive)
		case strconv.Itoa(ENV):
			result["env"] = append(result["env"], directive)
		case strconv.Itoa(CMD):
			result["cmd"] = append(result["cmd"], directive)
		case strconv.Itoa(ENTRYPOINT):
			result["entrypoint"] = append(result["entrypoint"], directive)
		case strconv.Itoa(WORKDIR):
			result["workdir"] = append(result["workdir"], directive)
		case strconv.Itoa(VOLUME):
			result["volume"] = append(result["volume"], directive)
		case strconv.Itoa(SHELL):
			result["shell"] = append(result["shell"], directive)
		case strconv.Itoa(STOPSIGNAL):
			result["stopsignal"] = append(result["stopsignal"], directive)
		case strconv.Itoa(ARG):
			result["arg"] = append(result["arg"], directive)
		}
	}

	result["run_last_stage"] = df.GetRunDirectivesLastStage()
	return result
}

func (df *Dockerfile) GetRaw() []map[string]interface{} {
	raw := make([]map[string]interface{}, len(df.Directives))

	for i, directive := range df.Directives {
		raw[i] = directive.Get()
	}

	return raw
}

func (df *Dockerfile) GetMaintainers() string {
	var maintainers []string

	for _, directive := range df.Directives {
		if directive.GetType() == LABEL {
			var i interface{} = directive
			labelsDirective, ok := i.(LabelDirective)
			if ok {
				labels := labelsDirective.Get()
				for _, label := range labels {
					keyValue := strings.SplitN(label.(string), "=", 2)
					if len(keyValue) == 2 {
						key := strings.TrimSpace(keyValue[0])
						value := strings.TrimSpace(keyValue[1])
						if key == "maintainer" || key == "MAINTAINER" {
							return value
						}
					}
				}
			}
		} else if directive.GetType() == MAINTAINER {
			var i interface{} = directive
			maintainersDirective, ok := i.(MaintainerDirective)
			if ok {
				maintainers = maintainersDirective.GetMaintainers()
			}
			if len(maintainers) > 0 {
				return strings.Join(maintainers, ", ")
			}
		}
	}

	return ""
}

func normalizeContent(content string) string {
	// Perform content normalization and preprocessing
	dockerfilePreprocessor := parsingutils.NewDockerfilePreprocessor(content)
	return dockerfilePreprocessor.GetNormalizedContent()
}

