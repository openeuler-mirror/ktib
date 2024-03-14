package builders

import (
	"errors"
	"fmt"
	"strings"

	"gitee.com/openeuler/ktib/pkg/builder"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/common/pkg/completion"
	"github.com/containers/podman/v4/cmd/podman/common"
	"github.com/containers/podman/v4/cmd/podman/registry"
	"github.com/containers/podman/v4/cmd/podman/validate"
	"github.com/containers/podman/v4/libpod/define"
	"github.com/spf13/cobra"
)

var (
	op         = options.RemoveOption{}
	filters    []string
	rmCidFiles = []string{}
)

func RMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm",
		Aliases: []string{"remove-builder"},
		Args: func(cmd *cobra.Command, args []string) error {
			return validate.CheckAllLatestAndIDFile(cmd, args, false, "cidfile")
		},
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
	_ = cmd.RegisterFlagCompletionFunc(filterFlagName, common.AutocompletePsFilters)
	return cmd
}

func rm(cmd *cobra.Command, args []string, op options.RemoveOption) error {
	name := args[0]
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	builderobj, err := builder.FindBuilder(store, name)
	if err != nil {
		return errors.New(fmt.Sprintf("Not found the %s builder", name))
	}
	err = builderobj.Remove()
	if err != nil {
		return errors.New(fmt.Sprintf("failed to  remove the builder of %s: %s", name, err))
	}
	return nil
}

func setExitCode(err error) {
	// If error is set to no such container, do not reset
	if registry.GetExitCode() == 1 {
		return
	}
	if errors.Is(err, define.ErrNoSuchCtr) || strings.Contains(err.Error(), define.ErrNoSuchCtr.Error()) {
		registry.SetExitCode(1)
	} else if errors.Is(err, define.ErrCtrStateInvalid) || strings.Contains(err.Error(), define.ErrCtrStateInvalid.Error()) {
		registry.SetExitCode(2)
	}
}
