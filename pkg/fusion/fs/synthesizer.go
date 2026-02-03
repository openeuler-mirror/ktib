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

package fs

import (
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	"github.com/sirupsen/logrus"
)

// DefaultSynthesizer is a stub implementation of FSSynthesizer
type DefaultSynthesizer struct{}

// NewDefaultSynthesizer creates a new DefaultSynthesizer
func NewDefaultSynthesizer() *DefaultSynthesizer {
	return &DefaultSynthesizer{}
}

// Synthesize creates the final rootfs
func (s *DefaultSynthesizer) Synthesize(plan *types.FusionPlan, outputDir string) error {
	logrus.Infof("Synthesizing filesystem to %s", outputDir)
	// TODO: Implement Overlay simulation and file copying
	return nil
}
