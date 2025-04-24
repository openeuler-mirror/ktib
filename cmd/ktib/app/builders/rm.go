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
	"strings"

	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/common/pkg/completion"
	"github.com/spf13/cobra"
)

var (
	op         = options.RemoveOption{}
	filters    []string
	rmCidFiles = []string{}
)

func RMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm",
		Short: "Remove one or more containers",
		Long: `The 'rm' command removes one or more containers.

Example:
  ktib builders rm -f container1 container2 --cidfile /path/to/cidfile --filter "status=exited"

Arguments:
  CONTAINER  The ID or name of the container(s) to remove.`,
		Aliases: []string{"remove-builder"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rm(cmd, args, op)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&op.Force, "force", "f", false, "Force removal of a running or unusable container")
	cidfileFlagName := "cidfile"
	flags.StringArrayVar(&rmCidFiles, cidfileFlagName, nil, "Read the container ID from the file")
	_ = cmd.RegisterFlagCompletionFunc(cidfileFlagName, completion.AutocompleteDefault)

	filterFlagName := "filter"
	flags.StringArrayVar(&filters, filterFlagName, []string{}, "Filter output based on conditions given")
	return cmd
}

func rm(cmd *cobra.Command, args []string, op options.RemoveOption) error {
	if len(args) == 0 {
		return errors.New(fmt.Sprintf("No container names provided for remove-builder"))
	}

	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}

	var errorMsgs []string
	for _, name := range args {
		builderobj, err := builder.FindBuilder(store, name)
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("Not found the %s builder: %s", name, err))
			continue
		}
		err = builderobj.Remove(op)
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("failed to remove the builder of %s: %s", name, err))
		}
	}

	if len(errorMsgs) > 0 {
		return errors.New(strings.Join(errorMsgs, "\n"))
	}

	return nil
}
