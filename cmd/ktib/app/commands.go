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
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "ktib",
		Short: "ktib: Trusted access controll tool for kcr ",
		Long: dedent.Dedent(`

			    ┌──────────────────────────────────────────────────────────┐
			    │ KTIB                                                     │
			    │ Trusted access controll tool for kcr                     │
			    │                                                          │
			    │ Please give us feedback at:                              │
			    │ http://172.17.66.176/kylin-virtualization/ktib/-/issues  │
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
			    │ builder-machine# ktib scan --check-test=ture             │
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
			return nil
		},
	}
	// TODO init or load config
	// TODO register all commands
	cmds.AddCommand(
		newCmdInit(),
		newCmdScan(),
		newCmdImage(),
		newCmdBuilder(),
		// todo: 还没实现
		newCmdMake())
	return cmds
}
