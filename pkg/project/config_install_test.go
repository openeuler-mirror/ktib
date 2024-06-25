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
