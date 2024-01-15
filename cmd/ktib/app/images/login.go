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
	"gitee.com/openeuler/ktib/pkg/options"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
	"os"
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
	ctx := context.Background()
	var loginOps *auth.LoginOptions
	loginOps = &auth.LoginOptions{
		Password:                  lops.Password,
		Username:                  lops.Username,
		StdinPassword:             lops.PasswordStdin,
		GetLoginSet:               true,
		Stdin:                     os.Stdin,
		Stdout:                    os.Stdout,
		AcceptRepositories:        true,
		AcceptUnspecifiedRegistry: true,
	}
	sctx := &types.SystemContext{
		AuthFilePath:                      loginOps.AuthFile,
		DockerCertPath:                    loginOps.CertDir,
		DockerDaemonInsecureSkipTLSVerify: lops.TLSVerify,
	}
	setRegistriesConfPath(sctx)
	loginOps.GetLoginSet = cmd.Flag("get-login").Changed
	return auth.Login(ctx, sctx, loginOps, args)
}

func setRegistriesConfPath(systemContext *types.SystemContext) {
	if systemContext.SystemRegistriesConfPath != "" {
		return
	}
	if envOverride, ok := os.LookupEnv("CONTAINERS_REGISTRIES_CONF"); ok {
		systemContext.SystemRegistriesConfPath = envOverride
		return
	}
	if envOverride, ok := os.LookupEnv("REGISTRIES_CONFIG_PATH"); ok {
		systemContext.SystemRegistriesConfPath = envOverride
		return
	}
}
