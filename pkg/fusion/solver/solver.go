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

package solver

import (
	"gitee.com/openeuler/ktib/pkg/fusion/config"
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	"github.com/sirupsen/logrus"
)

// DefaultSolver is a basic implementation of DependencySolver
type DefaultSolver struct{}

// NewDefaultSolver creates a new DefaultSolver
func NewDefaultSolver() *DefaultSolver {
	return &DefaultSolver{}
}

// Solve calculates the list of packages and files to keep
func (s *DefaultSolver) Solve(imageRef string, cfg *config.FusionConfig) (*types.FusionPlan, error) {
	logrus.Debugf("Solving dependencies for %s with config: %+v", imageRef, cfg)

	// Placeholder logic:
	// 1. Analyze image using pkg/analyze (to be implemented)
	// 2. Build dependency graph
	// 3. Resolve keep_packages
	
	// For now, return a dummy plan based on config
	plan := &types.FusionPlan{
		KeptPackages: cfg.Fusion.KeepPackages,
		KeptFiles:    []string{},
	}

	return plan, nil
}
