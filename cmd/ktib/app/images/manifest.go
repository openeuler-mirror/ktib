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

package images

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
)

type manifestCreateOpts struct {
	options.ManifestCreateOptions
	annotations  []string
	tlsVerifyCLI bool
	insecure     bool
}

type manifestAnnotateOpts struct {
	options.ManifestAnnotateOptions
	annotations []string
}

type manifestPushOpts struct {
	options.PushOption
}

func Manifest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Manipulate manifest lists and image indexes",
		Long:  `Creates, modifies, and pushes manifest lists and image indexes.`,
	}
	cmd.AddCommand(
		newSubCmdCreate(),
		newSubCmdAnnotate(),
		newSubCmdAdd(),
		newSubCmdInspect(),
		newSubCmdPush(),
	)
	return cmd
}

func newSubCmdCreate() *cobra.Command {
	var mcOptions = manifestCreateOpts{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create manifest list or image index",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cmd, args, mcOptions)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&mcOptions.All, "all", false, "add all of the lists' images if the images to add are lists")
	flags.BoolVarP(&mcOptions.Amend, "amend", "a", false, "modify an existing list if one with the desired name already exists")
	flags.StringArrayVar(&mcOptions.annotations, "annotation", nil, "set annotations on the new list")
	flags.BoolVar(&mcOptions.tlsVerifyCLI, "tls-verify", false, "verify the TLS certificate")
	flags.BoolVar(&mcOptions.insecure, "insecure", false, "allow insecure TLS connections")
	return cmd
}

func newSubCmdAnnotate() *cobra.Command {
	var maOptions = manifestAnnotateOpts{}
	cmd := &cobra.Command{
		Use:   "annotate",
		Short: "Add or update information about an entry in a manifest list or image index",
		Long: `Add or update information about a specific entry in a manifest list or image index.
		IMAGE_DIGEST must be the digest of an existing entry in the manifest list.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return annotate(cmd, args, maOptions)
		},
	}
	flags := cmd.Flags()
	flags.StringArrayVar(&maOptions.annotations, "annotation", nil, "set an `annotation` for the specified image")
	flags.StringVar(&maOptions.Arch, "arch", "", "override the `architecture` of the specified image")
	flags.StringVar(&maOptions.OS, "os", "", "override the `OS` of the specified image")
	flags.StringSliceVar(&maOptions.Features, "features", nil, "override the `features` of the specified image")
	flags.StringSliceVar(&maOptions.OSFeatures, "os-features", nil, "override the OS `features` of the specified image")
	flags.StringVar(&maOptions.OSVersion, "os-version", "", "override the OS `version` of the specified image")
	return cmd
}

func newSubCmdAdd() *cobra.Command {
	var addOptions options.ManifestAddOptions
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add manifest list or image index",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return add(cmd, args, addOptions)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&addOptions.Insecure, "insecure", false, "allow insecure TLS connections")
	return cmd
}

func newSubCmdInspect() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect manifest list or image index",
		Args:  cobra.ExactArgs(1),
		RunE:  inspect,
	}
	return cmd
}

func newSubCmdPush() *cobra.Command {
	var pushOptions = manifestPushOpts{}
	cmd := &cobra.Command{
		Use:   "push",
		Short: "todo",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return manifestPush(cmd, args, pushOptions)
		},
	}
	flags := cmd.Flags()
	// todo：based on image push parameters, they can be completed together with the push parameters later
	flags.StringVar(&pushOptions.SignBy, "sign-by", "", "sign the image using a GPG key with the specified `FINGERPRINT`")
	flags.StringVar(&pushOptions.Username, "username", "", "The username to use for authentication.")
	flags.StringVar(&pushOptions.Password, "password", "", "The password to use for authentication.")
	flags.StringVarP(&pushOptions.Format, "format", "f", "", "manifest type (oci or v2s2 or docker) to attempt to use when pushing the manifest list (default is manifest type of source)")
	flags.BoolVar(&pushOptions.Insecure, "insecure", false, "neither require HTTPS nor verify certificates when accessing the registry")
	return cmd
}

// Helper function to get store and imageManager instances
func getImageManager(cmd *cobra.Command) (*imagemanager.ImageManager, error) {
	store, err := utils.GetStore(cmd)
	if err != nil {
		return nil, err
	}
	return imagemanager.NewImageManager(store)
}

// Used to parse the --annotation parameter
func parseAnnotations(annotations []string) (map[string]string, error) {
	parsedAnnotations := make(map[string]string)
	for _, annotation := range annotations {
		k, v, parsed := strings.Cut(annotation, "=")
		if !parsed {
			return nil, fmt.Errorf("expected --annotation %q to be in key=value format", annotation)
		}
		parsedAnnotations[k] = v
	}
	return parsedAnnotations, nil
}

// Used to validate string arguments
func validateNonEmpty(param, name string) error {
	if param == "" {
		return fmt.Errorf(`invalid %s "%s"`, name, param)
	}
	return nil
}

func create(cmd *cobra.Command, args []string, op manifestCreateOpts) error {
	imageManager, err := getImageManager(cmd)
	if err != nil {
		return err
	}

	if cmd.Flags().Changed("tls-verify") {
		op.SkipTLSVerify = types.NewOptionalBool(!op.tlsVerifyCLI)
	}
	if cmd.Flags().Changed("insecure") {
		if op.SkipTLSVerify != types.OptionalBoolUndefined {
			return errors.New("--insecure can't be used with --tls-verify")
		}
		op.SkipTLSVerify = types.NewOptionalBool(op.insecure)
	}

	annotations, err := parseAnnotations(op.annotations)
	if err != nil {
		return err
	}
	op.Annotations = annotations

	imageID, err := imageManager.ManifestCreate(context.Background(), args[0], args[1:], op.ManifestCreateOptions)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", imageID)
	return nil
}

func annotate(cmd *cobra.Command, args []string, op manifestAnnotateOpts) error {
	imageManager, err := getImageManager(cmd)
	if err != nil {
		return err
	}

	annotations, err := parseAnnotations(op.annotations)
	if err != nil {
		return err
	}
	op.Annotations = annotations

	id, err := imageManager.ManifestAnnotate(args[0], args[1], op.ManifestAnnotateOptions)
	if err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func manifestPush(cmd *cobra.Command, args []string, op manifestPushOpts) error {
	imageManager, err := getImageManager(cmd)
	if err != nil {
		return err
	}

	listImageSpec := args[0]
	destSpec := args[len(args)-1]

	if err := validateNonEmpty(listImageSpec, "image name"); err != nil {
		return err
	}
	if err := validateNonEmpty(destSpec, "destination"); err != nil {
		return err
	}

	manifestSHA256, err := imageManager.ManifestPush(context.Background(), listImageSpec, destSpec, op.PushOption)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", manifestSHA256)
	return nil
}

func add(cmd *cobra.Command, args []string, op options.ManifestAddOptions) error {
	imageManager, err := getImageManager(cmd)
	if err != nil {
		return err
	}
	manifest := args[0]
	images := args[1:]
	listID, err := imageManager.ManifestAdd(context.Background(), manifest, images, op)
	if err != nil {
		return err
	}
	fmt.Println(listID)
	return nil
}

func inspect(cmd *cobra.Command, args []string) error {
	imageManager, err := getImageManager(cmd)
	if err != nil {
		return err
	}
	buf, err := imageManager.ManifestInspect(args[0])
	if err != nil {
		return err
	}
	fmt.Println(string(buf))
	return nil
}
