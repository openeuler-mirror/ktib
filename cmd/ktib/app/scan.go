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

package app

import (
	o "gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/scanner"
	"github.com/spf13/cobra"
)

var args o.Arguments
var loggerInitialized bool

const PolicyYaml = "/etc/ktib/policy.yaml"

func init() {
	loggerInitialized = true
}

func newCmdScan() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Run this command to audit Dockerfile",
		Long: `The scan command only supports Dockerfile static compliance auditing.

example:
  ktib scan dockerfile-audit --dockerfile /root/Dockerfile --json`,
	}
	cmd.AddCommand(
		newSubCmdDokcerfile(),
	)
	return cmd
}

func newSubCmdDokcerfile() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dockerfile-audit",
		Aliases: []string{"df-audit"},
		Short:   "dockerfile-audit uses its own grammar to parse valid Dockerfiles and deconstruct all directives.",
		Run: func(cmd *cobra.Command, arg []string) {
			scanner.RunDockerfileAudit(args)
		},
	}
	initScanDockerfileFlags(cmd)
	return cmd
}

func initScanDockerfileFlags(cmd *cobra.Command) {
	flag := cmd.Flags()
	flag.StringVar(&args.PolicyFile, "policy", PolicyYaml, "The dockerfile policy to use for the audit.")
	flag.StringVar(&args.Dockerfile, "dockerfile", "", "The DockerfileMsg to audit. Can be both a file or a directory.")
	flag.BoolVar(&args.ParseOnly, "parse-only", false, "Simply parse Dockerfile(s) and return the content, without applying any policy.")
	flag.StringVar(&args.JSONOutfile, "output", "dockerfile-audit.json", "Path to the JSON output file.")
	flag.BoolVar(&args.Verbose, "verbose", false, "Enables debug output.")
}
