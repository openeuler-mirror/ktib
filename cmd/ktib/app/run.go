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

import (
	"context"
	o "gitee.com/openeuler/ktib/cmd/ktib/app/options"
	sr "gitee.com/openeuler/ktib/pkg/scanner"
	"gitee.com/openeuler/ktib/pkg/types"
)

type InitializeScanner func(context.Context, ScannerConfig) (sr.Scanner, func(), error)

type ScannerConfig struct {
	Target string
}

type Runner struct {
	// TODO ADD RUNNER MEMBER
	driver string
}

func NewRunner(cliOption o.Option) (*Runner, error) {
	return nil, nil
}

func (r *Runner) ScanDockerfile(ctx context.Context) (types.Report, error) {
	// TODO init scanner by diver type
	var s InitializeScanner
	switch {
	case r.driver == "dockerfile-audit":
	case r.driver == "trivy":
	case r.driver == "kysec-CIS":
		s = KySecCISScanner
	}
	return r.Scan(ctx, s)
}

func (r *Runner) Scan(ctx context.Context, s InitializeScanner) (types.Report, error) {

	report, err := scan(ctx, s)
	if err != nil {
		return types.Report{}, err
	}
	return report, nil
}

func scan(ctx context.Context, initializeScanner InitializeScanner) (types.Report, error) {
	s, _, _ := initializeScanner(nil, ScannerConfig{})
	s.ScanArtifact()
	return types.Report{}, nil
}
