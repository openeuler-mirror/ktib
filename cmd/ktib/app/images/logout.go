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
	"errors"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/types"
	"github.com/spf13/cobra"
	"os"
)

func logout(cmd *cobra.Command, args []string) error {
	// TODO return auth.Logout(...)
	var logoutOps *auth.LogoutOptions
	logoutOps = &auth.LogoutOptions{
		Stdout:                    os.Stdout,
		AcceptUnspecifiedRegistry: true,
		AcceptRepositories:        true,
	}
	sctx := &types.SystemContext{
		AuthFilePath: logoutOps.AuthFile,
	}
	return auth.Logout(sctx, logoutOps, args)
}

func LogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out from a Docker registry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("you can only logout of one warehouse at a time")
			}
			return logout(cmd, args)
		},
	}
	return cmd
}
