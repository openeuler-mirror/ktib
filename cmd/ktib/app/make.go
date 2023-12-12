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

import "github.com/spf13/cobra"

type makeOption struct {
	file string
}

func runMake() error {
	// TODO pkg/builder impl build images subject
	return nil
}

func newCmdMake() *cobra.Command {

	var options makeOption
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Build an images from a Dockerfile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMake()
		},
		Args: cobra.MaximumNArgs(1),
	}
	flag := cmd.Flags()
	flag.StringVarP(&options.file, "file", "f", "string", "Name of the Dockerfile (Default is 'PATH/Dockerfile')")
	return cmd
}
