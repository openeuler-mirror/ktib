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

package builders

import (
	"errors"
	"fmt"
	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func from(cmd *cobra.Command, args []string, op *options.FromOption) error {
	if len(args) == 0 {
		return errors.New("an images name (or \"scratch\") must be specified")
	}
	if len(args) > 1 {
		return errors.New("too many arguments specified")
	}
	store, err := utils.GetStore(cmd)
	store.GraphRoot()
	if err != nil {
		return err
	}
	if store.Exists(op.Names) {
		return errors.New("builder name is exists, You have to remove that container to be able to reuse the name")
	}
	option := builder.BuilderOptions{
		FromImage:  args[0],
		Container:  op.Names,
		PullPolicy: op.PullPolicy,
	}
	builders, err := builder.NewBuilder(store, option)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", builders.ContainerID)
	if err := builders.Save(); err != nil {
		return err
	}
	return nil
}

func FROMCmd() *cobra.Command {
	var op options.FromOption
	cmd := &cobra.Command{
		Use:     "from",
		Aliases: []string{"from", "create-builder"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return from(cmd, args, &op)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&op.Names, "-name", "n", "", "Container name")
	flags.BoolVar(&op.PullPolicy, "pullpolicy", false, "Force images pull policy set ifnotparent")
	flags.BoolVar(&op.HostUIDMap, "-hostuidmap", false, "Force host UID map")
	flags.BoolVar(&op.HostGIDMap, "-hostgidmap", false, "Force host GID map")
	flags.StringVar(&op.UIDMap, "-uidmap", "", "UID map")
	flags.StringVar(&op.GIDMap, "-gidmap", "", "GID map")
	flags.StringVar(&op.SubUIDMap, "-subuidmap", "", "subuid UID map for a user")
	flags.StringVar(&op.SubGIDMap, "-subgidmap", "", "subgid GID map for a group")
	return cmd
}
