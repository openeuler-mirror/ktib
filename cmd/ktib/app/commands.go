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
	"gitee.com/openeuler/ktib/pkg/logging"
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
)

func NewCommand(version string) *cobra.Command {
	var logLevel string
	var logFormat string
	cmds := &cobra.Command{
		Use:   "ktib",
		Short: "Kylin Trusted Image Builder",
		Long: dedent.Dedent(`

Kylin Trusted Image Builder (ktib)

- 项目管理: 使用 ktib project 初始化项目目录、生成默认配置文件、构建/清理 rootfs、并构建镜像
- 一站式构建: 使用 ktib make 完成初始化、生成配置、构建/清理 rootfs 与构建镜像，暂时仅支持基础镜像一站式构建。
- 合规扫描: 使用 ktib scan dockerfile-audit 对 Dockerfile 进行静态合规审计，输出 JSON 格式的审计结果。
- 镜像操作: 使用 ktib images 进行 list/login/logout/pull/push/save/load/tag/inspect/manifest 等镜像操作。
- 构建器操作: 使用 ktib builders 进行 add/build/copy/commit/from/label/list/mount/run/rm/umount 等构建器操作。

反馈与问题: https://gitee.com/openeuler/ktib/issues

全局标志:
- --log-level: trace|debug|info|warn|error|fatal|panic
- --log-format: text|json
        `),
		Example: dedent.Dedent(`
  # 初始化项目骨架
  ktib project init --type minimal /path/to/project

  # 生成默认配置
  ktib project default_config --type minimal > config.yml

  # 构建和清理 rootfs，然后构建镜像
  ktib project build-rootfs --config config.yml /path/to/project
  ktib project clean-rootfs --type minimal /path/to/project
  ktib project build --name myimage --tag v1 /path/to/project

  # 一站式构建
  ktib make --init --type minimal --name myimage --tag v1 /path/to/project

  # Dockerfile 合规审计
  ktib scan dockerfile-audit --dockerfile ./Dockerfile --output audit.json

  # 常用镜像操作
  ktib images list --json
  ktib images pull docker.io/library/alpine:latest
  ktib images login docker.io --username myuser --password mypassword
  ktib images push myimage:v1
  ktib images save myimage:v1 -o myimage.tar
  ktib images load -i myimage.tar
  ktib images tag myimage:v1 myimage:latest
  ktib images inspect myimage:latest
  ktib images manifest myimage:latest

  # 生成 Shell 自动补全
  ktib completion bash
        `),
		// TODO Check that docker git is installed in your environment.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logging.Setup(logLevel, logFormat)
			return nil
		},
	}
	cmds.Version = version
	cmds.SetVersionTemplate("{{.Version}}\n")
	cmds.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level: trace|debug|info|warn|error|fatal|panic")
	cmds.PersistentFlags().StringVar(&logFormat, "log-format", "", "log format: text|json")
	// TODO init or load config
	// TODO register all commands
	cmds.AddCommand(
		newCmdProject(),
		newCmdScan(),
		newCmdImage(),
		newCmdBuilder(),
		// todo: Not implemented yet
		newCmdMake(),
		newCmdVersion(),
		// Add completion command
		&cobra.Command{
			Use:   "completion",
			Short: "Generate shell completion scripts",
			Long: `Generate shell completion scripts for ktib.

To load completions:

Bash:
  $ source <(ktib completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ ktib completion bash > /etc/bash_completion.d/ktib
  # macOS:
  $ ktib completion bash > /usr/local/etc/bash_completion.d/ktib

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ ktib completion zsh > "${fpath[1]}/_ktib"

  # You will need to start a new shell for this setup to take effect.

fish:
  $ ktib completion fish | source

  # To load completions for each session, execute once:
  $ ktib completion fish > ~/.config/fish/completions/ktib.fish

PowerShell:
  PS> ktib completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> ktib completion powershell > ktib.ps1
  # and source this file from your PowerShell profile.
`,
			DisableFlagsInUseLine: true,
			ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
			Args:                  cobra.ExactValidArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				switch args[0] {
				case "bash":
					cmd.Root().GenBashCompletion(cmd.OutOrStdout())
				case "zsh":
					cmd.Root().GenZshCompletion(cmd.OutOrStdout())
				case "fish":
					cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
				case "powershell":
					cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
				}
			},
		},
	)
	return cmds
}
