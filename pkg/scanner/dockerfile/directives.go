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
	return [...]string{
		"FROM",
		"RUN",
		"CMD",
		"LABEL",
		"MAINTAINER",
		"EXPOSE",
		"ENV",
		"ADD",
		"COPY",
		"ENTRYPOINT",
		"VOLUME",
		"USER",
		"WORKDIR",
		"ARG",
		"ONBUILD",
		"STOPSIGNAL",
		"HEALTHCHECK",
		"SHELL",
	}[d-1]
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
	// 解析 rawContent 字符串,提取各个属性
	parts := strings.Fields(rawContent)
	if len(parts) >= 1 {
		if strings.Contains(parts[0], "/") {
			// 包含 registry 信息
			registry, imageName := parts[0], parts[1]
			directive.Registry = registry
			directive.ImageName = imageName
		} else {
			// 没有 registry 信息
			imageName := parts[0]
			directive.ImageName = imageName
		}
	}

	// 进一步解析 imageName 部分,提取 tag、platform 等信息
	if strings.Contains(directive.ImageName, ":") {
		parts := strings.Split(directive.ImageName, ":")
		directive.ImageName = parts[0]
		directive.ImageTag = parts[1]
	}
	if strings.Contains(directive.ImageName, "@") {
		parts := strings.Split(directive.ImageName, "@")
		directive.ImageName = parts[0]
		directive.Platform = parts[1]
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
	parts := strings.Split(rawContent, "\"")
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[2])
	value = strings.ReplaceAll(value, "\\", "")
	labels[key] = value
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
	ports := strings.Split(rawContent, " ")
	directive.Ports = ports
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
	// 正则匹配一下是否存在chown
	re := regexp.MustCompile(`(?:--chown=(\S+)\s+)?(\S+)\s+(\S+)`)
	matches := re.FindStringSubmatch(rawContent)
	if len(matches) == 4 {
		chown = matches[1]
		source = matches[2]
		destination = matches[3]
	} else {
		// 没chown前面是源后面是目标
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

	// 按空格拆分rawContent以获得单个键值对
	parts := strings.Fields(rawContent)

	for _, part := range parts {
		// 检查零件是否包含“=”，以确定它是键值对还是只是一个键
		if strings.Contains(part, "=") {
			// 按“=”分割零件以获得键和值
			kv := strings.SplitN(part, "=", 2)
			vars[kv[0]] = kv[1]
		} else {
			// 如果没有“=”，如果没有“=”，假设该部分是键，其余部分是值
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

type ShellDirective struct {
	Type         DockerfileDirectiveType `json:"type"`
	Content      string                  `json:"raw_content"`
	RunLastStage []map[string]string
}

func (d *ShellDirective) GetType() DockerfileDirectiveType {
	return d.Type
}

func NewShellDirective(rawContent map[string]interface{}) ShellDirective {
	return ShellDirective{
		Type:    SHELL,
		Content: rawContent["content"].(string),
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
	if len(parts) != 1 {
		return nil
	}
	signal := parts[0]
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
