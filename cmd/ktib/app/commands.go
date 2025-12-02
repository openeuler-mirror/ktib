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
		Short: "ktib: Trusted access control tool for kcr ",
		Long: dedent.Dedent(`

			    ┌──────────────────────────────────────────────────────────┐
			    │ KTIB                                                     │
			    │ Trusted access control tool for kcr                     │
			    │                                                          │
			    │ Please give us feedback at:                              │
			    │ https://gitee.com/openeuler/ktib/issues                  │
			    └──────────────────────────────────────────────────────────┘

			Example usage:

			    Initial an empty project.

			    ┌──────────────────────────────────────────────────────────┐
			    │ Initialization phase:                                    │
			    ├──────────────────────────────────────────────────────────┤
			    │ builder-machine# git pull http://gitlab.com/test/test.git│
			    │ builder-machine# cd test                                 │
			    │ builder-machine# ktib init --buildType=source            │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ Scan phase:                                              │
			    ├──────────────────────────────────────────────────────────┤
			    │ builder-machine# ktib scan --check-test=true             │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ build phase:                                             │
			    ├──────────────────────────────────────────────────────────┤
			    │ builder-machine# ktib make                               │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ check phase:                                             │
			    ├──────────────────────────────────────────────────────────┤
			    │ builder-machine# ktib check                              │
			    └──────────────────────────────────────────────────────────┘

			    ┌──────────────────────────────────────────────────────────┐
			    │ store phase:                                             │
			    ├──────────────────────────────────────────────────────────┤
			    │ builder-machine# ktib store  --gitpush=true              │
			    └──────────────────────────────────────────────────────────┘

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
		// todo: 还没实现
		newCmdMake(),
		newCmdVersion(),
		// 添加补全命令
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
