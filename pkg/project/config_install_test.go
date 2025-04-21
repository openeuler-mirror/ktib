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
package project

import "testing"

func TestInstallPackages(t *testing.T) {
	type args struct {
		yumConfig string
		target    string
		packages  []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test installation of packages",
			args: args{
				yumConfig: "/etc/yum.conf",
				target:    "/tmp/target",
				packages:  []string{"yum", "bash"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InstallPackages(tt.args.yumConfig, tt.args.target, tt.args.packages...); (err != nil) != tt.wantErr {
				t.Errorf("InstallPackages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
