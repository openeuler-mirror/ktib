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
	"gitee.com/openeuler/ktib/pkg/imagemanager"
	"gitee.com/openeuler/ktib/pkg/options"
	utils2 "gitee.com/openeuler/ktib/pkg/utils"
	"github.com/spf13/cobra"
)

func LoginCmd() *cobra.Command {
	var op options.LoginOption
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to a Docker registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				op.ServerAddress = args[0]
			}
			return login(cmd, args, &op)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&op.Password, "password", "p", "", "Password")
	flags.BoolVar(&op.PasswordStdin, "password-stdin", false, "Take the password from stdin")
	flags.StringVarP(&op.Username, "username", "u", "", "Username")
	flags.BoolVarP(&op.TLSVerify, "tls-verify", "", false, "Require HTTPS and verify certificates when contacting registries")
	flags.BoolVar(&op.GetLoginSet, "get-login", false, "Return the current login user for the registry")
	return cmd
}

func login(cmd *cobra.Command, args []string, lops *options.LoginOption) error {
	store, err := utils2.GetStore(cmd)
	if err != nil {
		return err
	}
	imageManager, err := imagemanager.NewImageManager(store)
	if err != nil {
		return err
	}
	ctx := context.Background()
	getLoginSet := cmd.Flag("get-login").Changed
	return imageManager.KtibLogin(ctx, lops, args, getLoginSet)
}
