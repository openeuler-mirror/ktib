package builders

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/common/pkg/completion"
	"github.com/containers/podman/v4/cmd/podman/common"
	"github.com/containers/podman/v4/cmd/podman/registry"
	"github.com/containers/podman/v4/cmd/podman/utils"
	"github.com/containers/podman/v4/cmd/podman/validate"
	"github.com/containers/podman/v4/libpod/define"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var (
	op = options.RemoveOption{
		RmOptions: entities.RmOptions{Filters: make(map[string][]string)},
	}
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
	var errs utils.OutputErrors
	for _, cidFile := range rmCidFiles {
		content, err := os.ReadFile(cidFile)
		if err != nil {
			if op.Ignore && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("reading CIDFile: %w", err)
		}
		id := strings.Split(string(content), "\n")[0]
		args = append(args, id)
	}

	for _, f := range filters {
		split := strings.SplitN(f, "=", 2)
		if len(split) < 2 {
			return fmt.Errorf("invalid filter %q", f)
		}
		op.Filters[split[0]] = append(op.Filters[split[0]], split[1])
	}
	containerEngine, err := registry.NewContainerEngine(cmd, args)
	if err != nil {
		return err
	}
	nameOrID := utils.RemoveSlash(args)
	res, err := containerEngine.ContainerRm(context.Background(), nameOrID, op.RmOptions)
	if err != nil {
		if op.Force && strings.Contains(err.Error(), define.ErrNoSuchCtr.Error()) {
			return nil
		}
		setExitCode(err)
		return err
	}
	for _, r := range res {
		switch {
		case r.Err != nil:
			if errors.Is(r.Err, define.ErrWillDeadlock) {
				logrus.Errorf("Potential deadlock detected - please run 'podman system renumber' to resolve")
			}
			if op.Force && strings.Contains(r.Err.Error(), define.ErrNoSuchCtr.Error()) {
				continue
			}
			setExitCode(r.Err)
			errs = append(errs, r.Err)
		default:
			fmt.Println(r.Id)
		}
	}
	return errs.PrintErrors()
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
