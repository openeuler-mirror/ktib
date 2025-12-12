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
    "encoding/json"
    "fmt"
    "net/url"
    "regexp"
    "strings"
)

type DockerfileDirectiveType int

const (
	FROM = iota + 1
	RUN
	CMD
	LABEL
	MAINTAINER
	EXPOSE
	ENV
	ADD
	COPY
	ENTRYPOINT
	VOLUME
	USER
	WORKDIR
	ARG
	ONBUILD
	STOPSIGNAL
	HEALTHCHECK
	SHELL
)

func (d DockerfileDirectiveType) String() string {
	names := [...]string{
		"FROM", "RUN", "CMD", "LABEL", "MAINTAINER", "EXPOSE", "ENV", "ADD", "COPY",
		"ENTRYPOINT", "VOLUME", "USER", "WORKDIR", "ARG", "ONBUILD", "STOPSIGNAL", "HEALTHCHECK", "SHELL",
	}
	if int(d) < 1 || int(d) > len(names) {
		return fmt.Sprintf("UNKNOWN_DIRECTIVE(%d)", d)
	}
	return names[d-1]
}

type DfDirective interface {
	GetType() DockerfileDirectiveType
	Get() map[string]interface{}
}

type FromDirective struct {
	Type           DockerfileDirectiveType `json:"type"`
	Content        string                  `json:"raw_content"`
	RunLastStage   []map[string]string
	Platform       string `json:"platform"`
	Registry       string `json:"registry"`
	ImageLocalName string `json:"local_name"`
	ImageTag       string `json:"tag"`
	ImageName      string `json:"image"`
}

func (d *FromDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"registry":    d.Registry,
		"image":       d.ImageName,
		"tag":         d.ImageTag,
		"local_name":  d.ImageLocalName,
	}
}

func (d *FromDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewFromDirective(rawContent string) *FromDirective {
	directive := &FromDirective{
		Type:    FROM,
		Content: rawContent,
	}
    content := strings.TrimSpace(strings.TrimPrefix(rawContent, "FROM "))
    tokens := strings.Fields(content)
	idx := 0
	if len(tokens) == 0 {
		return directive
	}
	if strings.HasPrefix(tokens[0], "--platform=") {
		directive.Platform = strings.TrimPrefix(tokens[0], "--platform=")
		idx = 1
	}
    base := ""
	if idx < len(tokens) {
		base = tokens[idx]
		idx = idx + 1
	}
	if idx+1 < len(tokens) && strings.EqualFold(tokens[idx], "as") {
		directive.ImageLocalName = strings.Trim(tokens[idx+1], "\"'")
	}
    ref := base
    if ref == "" {
        return directive
    }
    // Process image reference with explicit scheme
    if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
        if u, err := url.Parse(ref); err == nil {
            directive.Registry = u.Host
            path := strings.TrimPrefix(u.Path, "/")
            ref = path
        }
    }
    if at := strings.Index(ref, "@"); at != -1 {
        directive.Platform = ref[at+1:]
        ref = ref[:at]
    }
    nameTag := ref
    if colon := strings.LastIndex(nameTag, ":"); colon != -1 {
        directive.ImageName = nameTag[:colon]
        directive.ImageTag = nameTag[colon+1:]
    } else {
        directive.ImageName = nameTag
        directive.ImageTag = "latest"
    }
    // Extract registry (no scheme case)
    if slash := strings.Index(directive.ImageName, "/"); slash != -1 {
        first := strings.Split(directive.ImageName, "/")[0]
        if strings.Contains(first, ".") || strings.Contains(first, ":") {
            directive.Registry = first
            directive.ImageName = strings.Join(strings.Split(directive.ImageName, "/")[1:], "/")
        }
    }
    return directive
}

type RunDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
}

func (d *RunDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
	}
}

func (d *RunDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewRunDirective(rawContent string) *RunDirective {
	return &RunDirective{
		Type:    RUN,
		Content: rawContent,
	}
}

type LabelDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Labels       map[string]string `json:"labels"`
}

func (d *LabelDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"labels":      d.Labels,
	}
}

func (d *LabelDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewLabelDirective(rawContent string) *LabelDirective {
	labels := make(map[string]string)
	content := strings.TrimSpace(strings.TrimPrefix(rawContent, "LABEL "))
	parts := strings.Fields(content)
	for _, p := range parts {
		if strings.Contains(p, "=") {
			kv := strings.SplitN(p, "=", 2)
			k := strings.Trim(kv[0], "\"'")
			v := strings.Trim(kv[1], "\"'")
			labels[k] = v
		}
	}
	return &LabelDirective{
		Type:    LABEL,
		Content: rawContent,
		Labels:  labels,
	}
}

type UserDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	User         string `json:"user"`
	Group        string `json:"group"`
}

func (d UserDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"user":        d.User,
		"group":       d.Group,
		"raw_content": d.Content,
	}
}

func (d *UserDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewUserDirective(rawContent string) *UserDirective {
	directive := UserDirective{
		Type:    USER,
		Content: rawContent,
	}
	data := make(map[string]interface{})
	if err := json.Unmarshal([]byte(rawContent), &data); err == nil {
		if user, ok := data["user"].(string); ok {
			directive.User = user
		}
		if group, ok := data["group"].(string); ok {
			directive.Group = group
		}
	}

	return &directive
}

type ExposeDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Ports        []string `json:"ports"`
}

func (d ExposeDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"ports":       d.Ports,
	}
}

func (d *ExposeDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewExposeDirective(rawContent string) *ExposeDirective {
    directive := &ExposeDirective{
        Type:    EXPOSE,
        Content: rawContent,
    }
    s := strings.TrimSpace(strings.TrimPrefix(rawContent, "EXPOSE "))
    if s != "" {
        ports := strings.Fields(s)
        cleaned := make([]string, 0, len(ports))
        for _, p := range ports {
            cleaned = append(cleaned, strings.Trim(p, "\"'"))
        }
        directive.Ports = cleaned
    }
    if len(directive.Ports) > 0 {
        directive.Content = "EXPOSE " + strings.Join(directive.Ports, " ")
    }
    return directive
}

type MaintainerDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Maintainers  []string `json:"maintainers"`
}

func (d MaintainerDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":           d.Type,
		"raw_content":    d.Content,
		"run_last_stage": d.RunLastStage,
		"maintainers":    d.Maintainers,
	}
}
func (d MaintainerDirective) GetMaintainers() []string {
	return d.Maintainers
}

func (d *MaintainerDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewMaintainerDirective(rawContent string) *MaintainerDirective {
	directive := &MaintainerDirective{
		Type:    MAINTAINER,
		Content: rawContent,
	}
	maintainers := strings.Split(rawContent, " ")
	directive.Maintainers = maintainers
	return directive
}

type AddDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Chown        string `json:"chown"`
	Source       string `json:"source"`
	Destination  string `json:"destination"`
}

func (d AddDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"chown":       d.Chown,
		"source":      d.Source,
		"destination": d.Destination,
	}
}

func (d *AddDirective) GetType() DockerfileDirectiveType {
	return d.Type
}
func NewAddDirective(rawContent string) *AddDirective {
	var chown, source, destination string
	// Regex match to check for the presence of chown
	re := regexp.MustCompile(`(?:--chown=(\S+)\s+)?(\S+)\s+(\S+)`)
	matches := re.FindStringSubmatch(rawContent)
	if len(matches) == 4 {
		chown = matches[1]
		source = matches[2]
		destination = matches[3]
	} else {
		// If no chown, the first is the source and the second is the destination
		parts := strings.Split(rawContent, " ")
		if len(parts) >= 2 {
			source = parts[0]
			destination = parts[1]
		}
	}
	return &AddDirective{
		Type:        ADD,
		Content:     rawContent,
		Chown:       chown,
		Source:      source,
		Destination: destination,
	}
}

type CopyDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Chown        string `json:"chown"`
	Source       string `json:"source"`
	Destination  string `json:"destination"`
}

func (d *CopyDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func (d CopyDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"chown":       d.Chown,
		"source":      d.Source,
		"destination": d.Destination,
	}
}

func NewCopyDirective(rawContent string) *CopyDirective {
	var chown, source, destination string
	re := regexp.MustCompile(`(?:--chown=(\S+)\s+)?(\S+)\s+(\S+)`)
	matches := re.FindStringSubmatch(rawContent)
	if len(matches) == 4 {
		chown = matches[1]
		source = matches[2]
		destination = matches[3]
	} else {
		parts := strings.Split(rawContent, " ")
		if len(parts) >= 2 {
			source = parts[0]
			destination = parts[1]
		}
	}
	return &CopyDirective{
		Type:        COPY,
		Content:     rawContent,
		Chown:       chown,
		Source:      source,
		Destination: destination,
	}
}

type EnvDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Variables    map[string]string `json:"variables"`
}

func (d EnvDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"variables":   d.Variables,
	}
}

func (d *EnvDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewEnvDirective(rawContent string) *EnvDirective {
	vars := make(map[string]string)

	// Split rawContent by space to get individual key-value pairs
	parts := strings.Fields(rawContent)

	for _, part := range parts {
		// Check if the part contains "=" to determine if it is a key-value pair or just a key
		if strings.Contains(part, "=") {
			// Split the part by "=" to get the key and value
			kv := strings.SplitN(part, "=", 2)
			vars[kv[0]] = kv[1]
		} else {
			// If there is no "=", assume the first part is the key and the rest is the value
			key := parts[0]
			value := strings.Join(parts[1:], " ")
			vars[key] = value
		}
	}

	return &EnvDirective{
		Type:      ENV,
		Content:   rawContent,
		Variables: vars,
	}
}

type CmdDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
}

func (d *CmdDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func (d CmdDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
	}
}

func NewCmdDirective(rawContent string) *CmdDirective {
	return &CmdDirective{
		Type:    CMD,
		Content: rawContent,
	}
}

type EntrypointDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
}

func (d *EntrypointDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func (d EntrypointDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
	}
}

func NewEntrypointDirective(rawContent string) *EntrypointDirective {
	return &EntrypointDirective{
		Type:    ENTRYPOINT,
		Content: rawContent,
	}
}

type WorkdirDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
}

func (d *WorkdirDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func (d WorkdirDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
	}
}

func NewWorkdirDirective(rawContent string) *WorkdirDirective {
	return &WorkdirDirective{
		Type:    WORKDIR,
		Content: rawContent,
	}
}

type VolumeDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Volumes      []string `json:"volumes"`
}

func (d *VolumeDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewVolumeDirective(rawContent string) *VolumeDirective {
	paths := strings.Fields(rawContent)
	return &VolumeDirective{
		Type:    VOLUME,
		Content: rawContent,
		Volumes: paths,
	}
}

func (d VolumeDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"volumes":     d.Volumes,
	}
}

type HealthcheckDirective struct {
	Type    DockerfileDirectiveType `json:"type"`
	Content string                  `json:"raw_content"`
}

func (d *HealthcheckDirective) GetType() DockerfileDirectiveType {
	return d.Type
}
func NewHealthcheckDirective(rawContent string) *HealthcheckDirective {
	return &HealthcheckDirective{
		Type:    HEALTHCHECK,
		Content: rawContent,
	}
}

func (d HealthcheckDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
	}
}

type ShellDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
}

func (d *ShellDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewShellDirective(rawContent string) *ShellDirective {
	return &ShellDirective{
		Type:    SHELL,
		Content: rawContent,
	}
}

func (d ShellDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
	}
}

type StopsignalDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Signal       string `json:"stopsignal"`
}

func (d *StopsignalDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewStopsignalDirective(rawContent string) *StopsignalDirective {
	parts := strings.Fields(rawContent)
	signal := ""
	if len(parts) >= 1 {
		signal = parts[0]
	}
	return &StopsignalDirective{
		Type:    STOPSIGNAL,
		Content: rawContent,
		Signal:  signal,
	}
}

func (d StopsignalDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"stopsignal":  d.Signal,
	}
}

type ArgDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
	Argument     string `json:"argument"`
}

func (d *ArgDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewArgDirective(rawContent string) *ArgDirective {
	return &ArgDirective{
		Type:     ARG,
		Content:  rawContent,
		Argument: rawContent,
	}
}

func (d ArgDirective) Get() map[string]interface{} {
	return map[string]interface{}{
		"type":        d.Type.String(),
		"raw_content": d.Content,
		"argument":    d.Argument,
	}
}
