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
	"reflect"
	"sort"
	"testing"

	coretypes "gitee.com/openeuler/ktib/pkg/types"
)

func TestSolveGraph(t *testing.T) {
	// Setup test data
	pkgs := []coretypes.Package{
		{Name: "pkgA", Requires: []string{"pkgB"}},
		{Name: "pkgB", Requires: []string{"libfoo"}},
		{Name: "pkgC", Provides: []string{"libfoo"}},
		{Name: "pkgD", Requires: []string{"pkgA"}},
		{Name: "pkgE", Requires: []string{"missing"}},
		{Name: "pkgCycle1", Requires: []string{"pkgCycle2"}},
		{Name: "pkgCycle2", Requires: []string{"pkgCycle1"}},
	}

	solver := &DefaultSolver{}

	tests := []struct {
		name     string
		keep     []string
		expected []string
	}{
		{
			name:     "Simple Chain (A->B->C)",
			keep:     []string{"pkgA"},
			expected: []string{"pkgA", "pkgB", "pkgC"},
		},
		{
			name:     "Direct Keep",
			keep:     []string{"pkgB"},
			expected: []string{"pkgB", "pkgC"},
		},
		{
			name:     "Missing Dependency",
			keep:     []string{"pkgE"},
			expected: []string{"pkgE"},
		},
		{
			name:     "Cycle",
			keep:     []string{"pkgCycle1"},
			expected: []string{"pkgCycle1", "pkgCycle2"},
		},
		{
			name:     "Multiple Roots",
			keep:     []string{"pkgA", "pkgE"},
			expected: []string{"pkgA", "pkgB", "pkgC", "pkgE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := solver.solveGraph(pkgs, tt.keep)
			sort.Strings(result)
			sort.Strings(tt.expected)
			
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
