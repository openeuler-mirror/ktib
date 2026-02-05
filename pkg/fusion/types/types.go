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

package types

import (
	"gitee.com/openeuler/ktib/pkg/fusion/config"
)

// DependencySolver defines the interface for solving image dependencies
type DependencySolver interface {
	// Solve calculates the list of packages and files to keep
	Solve(imageRef string, config *config.FusionConfig) (*FusionPlan, error)
}

// DBReconstructor defines the interface for reconstructing the RPM database
type DBReconstructor interface {
	// Reconstruct builds a new RPM DB based on the kept packages
	Reconstruct(sourcePath string, plan *FusionPlan, outputDir string) error
}

// FSSynthesizer defines the interface for synthesizing the final filesystem
type FSSynthesizer interface {
	// Synthesize creates the final rootfs
	Synthesize(imageRef string, plan *FusionPlan, outputDir string) error
	// ExtractRPMDB extracts the RPM DB from the image to a destination directory
	ExtractRPMDB(imageRef string, dest string) error
}

// Verifier defines the interface for verifying the result
type Verifier interface {
	// Verify checks the integrity and usability of the fused image
	Verify(rootfsPath string) error
}

// FusionPlan contains the result of the solver
type FusionPlan struct {
	KeptPackages []string
	KeptFiles    []string
	// Add more fields as needed
}
