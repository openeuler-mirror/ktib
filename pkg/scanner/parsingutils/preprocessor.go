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

package parsingutils

import (
	"log"
	"regexp"
	"strings"
)

type DockerfilePreprocessor struct {
	content string
}

func NewDockerfilePreprocessor(dockerfileContent string) *DockerfilePreprocessor {
	return &DockerfilePreprocessor{
		content: dockerfileContent,
	}
}

func (p *DockerfilePreprocessor) GetNormalizedContent() string {
	p.normalize()
	return p.content
}

func (p *DockerfilePreprocessor) normalize() {
	p.flattenLines()
	p.removeComments()
	p.removeDoubleWhitespaces()
	p.removeEmptyLines()
	p.removesLeadingNewlines()
	p.removesLeadingSpaces()
	p.removesTrailingSpaces()
	envs := p.getEnvBasic()
	p.resolveEnvs(envs)
	envs = p.getEnvKeyValue()
	p.resolveEnvs(envs)
}

func (p *DockerfilePreprocessor) resolveEnvs(envs map[string]string) {
	for key, value := range envs {
		envNames := []string{"[$]" + key, "[$]{" + key + "(-[\\S]+)?}"}
		for _, pattern := range envNames {
			regex := regexp.MustCompile(pattern)
			if regex.MatchString(p.content) {
				log.Printf("Resolving env variable %s with value %s.", key, value)
			}
			p.content = regex.ReplaceAllString(p.content, value)
		}
	}
}

func (p *DockerfilePreprocessor) removeComments() {
	comments := regexp.MustCompile(`#.*\n`)
	p.content = comments.ReplaceAllString(p.content, "")
}

func (p *DockerfilePreprocessor) flattenLines() {
	lineContinuation := regexp.MustCompile(`[\\][\n]+`)
	p.content = lineContinuation.ReplaceAllString(p.content, " ")
}

func (p *DockerfilePreprocessor) removeDoubleWhitespaces() {
	spaces := regexp.MustCompile(`[ ]{2,}`)
	p.content = spaces.ReplaceAllString(p.content, " ")
}

func (p *DockerfilePreprocessor) removeEmptyLines() {
	emptyLines := regexp.MustCompile(`[\n]{2,}`)
	p.content = emptyLines.ReplaceAllString(p.content, "\n")
}

func (p *DockerfilePreprocessor) removesLeadingSpaces() {
	p.content = strings.TrimLeft(p.content, " ")
	linesWithSpaces := regexp.MustCompile(`\n[ ]+`)
	p.content = linesWithSpaces.ReplaceAllString(p.content, "\n")
}

func (p *DockerfilePreprocessor) removesTrailingSpaces() {
	endingWhitespaces := regexp.MustCompile(`[ ]+\n`)
	p.content = endingWhitespaces.ReplaceAllString(p.content, "\n")
}

func (p *DockerfilePreprocessor) removesLeadingNewlines() {
	p.content = strings.TrimLeft(p.content, "\n")
}

func (p *DockerfilePreprocessor) getEnvBasic() map[string]string {
	envs := make(map[string]string)
	assignment := regexp.MustCompile(`(?:ENV\s+)?(\w+)\s*=\s*(?:"([^"]+)"|'([^']+)'|([^"\s]+))`)
	matches := assignment.FindAllStringSubmatch(p.content, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			key := match[1]
			value := strings.TrimSpace(match[2] + match[3] + match[4])
			envs[key] = value
		}
	}
	return envs
}

func (p *DockerfilePreprocessor) getEnvKeyValue() map[string]string {
	variables := make(map[string]string)
	dockerfileLines := strings.Split(p.content, "\n")
	envMatch := regexp.MustCompile(`^(env|ENV) .*`)
	lineWithKeyValues := regexp.MustCompile(`(env|ENV) ((([^=\s]+|(\"|')[^'\"=]+(\"|'))=([^=\s"']+|(\"|')[^="']+(\"|')[ ]*))+)`)
	for _, line := range dockerfileLines {
		if envMatch.MatchString(line) {
			if lineWithKeyValues.MatchString(line) {
				log.Printf("Key value ENV match: %s", line)
				line = strings.ReplaceAll(line, "\\ ", "#")
				line = p.replaceSpacesInQuotes(line)
				envs := strings.Split(line, " ")[1:]
				for _, env := range envs {
					parts := strings.Split(env, "=")
					key := parts[0]
					value := strings.Trim(parts[1], "\"'")
					value = strings.ReplaceAll(value, "#", " ")
					variables[key] = value
				}
			}
		}
	}
	return variables
}

func (p *DockerfilePreprocessor) replaceSpacesInQuotes(line string) string {
	inside := false
	result := ""
	for _, char := range line {
		if !inside && (char == '\'' || char == '"') {
			result += string(char)
			inside = true
		} else if inside && char == ' ' {
			result += "#"
		} else {
			result += string(char)
		}
	}
	return result
}
