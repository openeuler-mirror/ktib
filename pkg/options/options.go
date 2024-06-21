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

package options

import (
	"io"

	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/containers/buildah"
	"github.com/containers/podman/v4/pkg/domain/entities"
)

type Option struct {
	artifact.Option
	Driver string
}

type ImagesOption struct {
	Quiet    bool
	Digests  bool
	Truncate bool
	Json     bool
	// TODO
	Format string
	//Filter map[string]string
}

type LoginOption struct {
	ServerAddress string
	Password      string
	Username      string
	TLSVerify     bool
	PasswordStdin bool
	Stdin         io.Reader
	Stdout        io.Writer
	GetLoginSet   bool
}

type PullOption struct {
	Remote   string
	Platform string
}

type PushOption struct {
	SignBy string
}

type RemoveOption struct {
	Filters map[string][]string
	All     bool
	Depend  bool
	Force   bool
	Ignore  bool
	Latest  bool
	Timeout *uint
	Volumes bool
}

type BuildersOption struct {
	Json bool
	entities.BuildOptions
	//common.BuildFlagsWrapper
}

type FromOption struct {
	Names      string
	ID         string
	Digest     string
	Layer      string
	CreateRO   bool
	HostUIDMap bool
	HostGIDMap bool
	UIDMap     string
	GIDMap     string
	SubUIDMap  string
	SubGIDMap  string
	ReadOnly   bool
	PullPolicy bool
}

type RUNOption struct {
	Interactive bool
	TTY         bool
	Workdir     string
	Runtime     string
	Detach      bool
	Rm          bool
}

type CreateOption struct {
	entities.ContainerCreateOptions
}

type MountOption struct {
	Json bool
}

type CommitOption struct {
	Maintainer string
	Message    string
	Remove     bool
	EntryPoint string
	CMD        []string
	Env        []string
	entities.CommitOptions
}

type CopyOption struct {
	entities.ContainerCpOptions
}

type ExistOption struct {
	entities.ContainerExistsOptions
}
type IFIOptions struct {
	buildah.ImportFromImageOptions
}

type Arguments struct {
	PolicyFile     string
	Dockerfile     string
	ParseOnly      bool
	GenerateJSON   bool
	GenerateReport bool
	JSONOutfile    string
	ReportName     string
	ReportTemplate string
	Verbose        bool
}
