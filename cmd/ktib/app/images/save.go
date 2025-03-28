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
	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
	"slices"
	"strings"
)

const (
	OCIManifestDir  = "oci-dir"
	OCIArchive      = "oci-archive"
	V2s2ManifestDir = "docker-dir"
	V2s2Archive     = "docker-archive"
)

var (
	MultiImageArchive bool
	ValidSaveFormats  = []string{OCIManifestDir, OCIArchive, V2s2ManifestDir, V2s2Archive}
)

func SaveCmd() *cobra.Command {
	var op options.SaveOption
	cmd := &cobra.Command{
		Use:   "save",
		Short: "save images",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("need at least 1 argument")
			}
			format, err := cmd.Flags().GetString("format")
			if err != nil {
				return err
			}
			if !slices.Contains(ValidSaveFormats, format) {
				return fmt.Errorf("format value must be one of %s", strings.Join(ValidSaveFormats, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return imageSave(cmd, args, op)
		},
	}
	flag := cmd.Flags()
	flag.StringVarP(&op.Output, "output", "o", "", "Write to a file, instead of stdout")
	flag.StringVarP(&op.Format, "format", "f", V2s2Archive, "Save image to oci-archive, oci-dir (directory with oci manifest type), docker-archive, docker-dir (directory with v2s2 manifest type)")
	flag.BoolVarP(&op.MultiImageArchive, "multi-image-archive", "m", MultiImageArchive, "Interpret additional arguments as images not tags and create a multi-image-archive (only for docker-archive)")
	return cmd
}

func imageSave(cmd *cobra.Command, args []string, op options.SaveOption) error {
	if len(op.Output) == 0 {
		return fmt.Errorf("output is required")
	}
	if err := utils.ValidateFileName(op.Output); err != nil {
		return err
	}
	store, err := utils.GetStore(cmd)
	if err != nil {
		return err
	}
	imageManager, err := imagemanager.NewImageManager(store)
	if err != nil {
		return err
	}
	var tags []string
	if len(args) > 1 {
		tags = args[1:]
	}
	err = imageManager.SaveImage(context.Background(), op, tags, args[0])
	if err != nil {
		return err
	}
	return nil
}
