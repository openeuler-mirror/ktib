/*
   Copyright (c) 2026 KylinSoft Co., Ltd.
   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
   You can use this software according to the terms and conditions of the Mulan PSL v2.
   You may obtain a copy of Mulan PSL v2 at:
            http://license.coscl.org.cn/MulanPSL2
   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
   See the Mulan PSL v2 for more details.
*/

package analyze

import (
	"testing"

	"gitee.com/openeuler/ktib/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestGenerateRecommendations_Details(t *testing.T) {
	// Mock Config
	cfg := &types.Config{
		Strategy: types.Strategy{
			EnableLevels: []string{"AGGRESSIVE"},
		},
		Rules: []types.Rule{
			{
				ID:          "RM_DEV_TOOLS",
				Level:       "AGGRESSIVE",
				Description: "Build tools",
				Match: types.Match{
					PkgNames: []string{"gcc", "make"},
				},
			},
		},
	}

	analyzer := &Analyzer{
		Rules: *cfg,
	}

	// Mock Data
	pkgs := types.PackageInfo{
		RPM: []types.Package{
			{Name: "gcc", Size: 100},
			{Name: "make", Size: 200},
			{Name: "bash", Size: 500},
		},
	}

	recs := analyzer.GenerateRecommendations(nil, pkgs, types.FilesystemInfo{}, types.WasteDetection{}, "", nil)

	assert.Equal(t, 1, len(recs))
	assert.Equal(t, "RM_DEV_TOOLS", recs[0].Code)
	
	// Check if MatchedItems contains expected values
	// Since order might depend on implementation (though slices are ordered), check containment
	assert.Contains(t, recs[0].MatchedItems, "rpm:gcc")
	assert.Contains(t, recs[0].MatchedItems, "rpm:make")
	assert.NotContains(t, recs[0].MatchedItems, "rpm:bash")
	
	// Check saving calculation
	// Format bytes might be "300 B"
	assert.Equal(t, "300 B", recs[0].Saving)
}
