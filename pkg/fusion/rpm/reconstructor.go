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

package rpm

import (
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	"github.com/sirupsen/logrus"
)

// DefaultReconstructor is a stub implementation of DBReconstructor
type DefaultReconstructor struct{}

// NewDefaultReconstructor creates a new DefaultReconstructor
func NewDefaultReconstructor() *DefaultReconstructor {
	return &DefaultReconstructor{}
}

// Reconstruct builds a new RPM DB based on the kept packages
func (r *DefaultReconstructor) Reconstruct(plan *types.FusionPlan, outputDir string) error {
	logrus.Infof("Reconstructing RPM DB in %s for %d packages", outputDir, len(plan.KeptPackages))
	// TODO: Implement actual RPM DB generation
	return nil
}
