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
)

type Option struct {
	Driver string
}

type ImagesOption struct {
	Quiet    bool
	Digests  bool
	Truncate bool
	Json     bool
	Format   string
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
	Workdir string
	Runtime string
}

type MountOption struct {
	Json bool
}

type IFIOptions struct {
	Labels map[string]string `json:"labels"`
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

type SummaryStats struct {
	TotalTests        int
	SuccessTests      int
	FailedTests       int
	SuccessPercentage string
	FailedPercentage  string
	ComplianceLevel   string
	ComplianceColor   string
}
