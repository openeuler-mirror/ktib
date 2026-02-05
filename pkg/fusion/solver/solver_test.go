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

func TestResolveAppAnchors(t *testing.T) {
	solver := &DefaultSolver{}

	pkgs := []coretypes.Package{
		{Name: "python3", Files: []string{"/usr/bin/python3", "/usr/lib/python3.9"}},
		{Name: "bash", Files: []string{"/usr/bin/bash", "/bin/sh"}}, // /bin/sh -> bash
		{Name: "app", Files: []string{"/app/server.py"}},
	}

	tests := []struct {
		name     string
		config   coretypes.ImageConfig
		expected []string
	}{
		{
			name:     "Cmd with absolute path",
			config:   coretypes.ImageConfig{Cmd: []string{"/usr/bin/python3", "main.py"}},
			expected: []string{"python3"},
		},
		{
			name:     "Entrypoint with relative path and WorkDir",
			config:   coretypes.ImageConfig{Entrypoint: []string{"./server.py"}, WorkingDir: "/app"},
			expected: []string{"app"},
		},
		{
			name:     "Cmd with binary name only (PATH lookup)",
			config:   coretypes.ImageConfig{Cmd: []string{"bash"}},
			expected: []string{"bash"},
		},
		{
			name:     "Complex command line",
			config:   coretypes.ImageConfig{Cmd: []string{"/bin/sh", "-c", "echo hello"}},
			expected: []string{"bash"},
		},
		{
			name:     "Unknown command",
			config:   coretypes.ImageConfig{Cmd: []string{"/bin/unknown"}},
			expected: nil,
		},
		{
			name:     "Shell wrapping python",
			config:   coretypes.ImageConfig{Cmd: []string{"/bin/sh", "-c", "python3 -m llama_cpp.server --host 0.0.0.0"}},
			expected: []string{"bash", "llama_cpp-python", "python3"},
		},
		{
			name:     "Python module resolution (llama_cpp)",
			config:   coretypes.ImageConfig{Cmd: []string{"python3", "-m", "llama_cpp.server", "--host", "0.0.0.0"}},
			expected: []string{"llama_cpp-python", "python3"},
		},
		{
			name:     "Python script resolution",
			config:   coretypes.ImageConfig{Cmd: []string{"python", "/app/main.py"}},
			expected: []string{"app-pkg"},
		},
	}

	// Add mock packages for python tests
	pkgs = append(pkgs, coretypes.Package{
		Name: "llama_cpp-python",
		Files: []string{
			"/usr/lib/python3.9/site-packages/llama_cpp/__init__.py",
			"/usr/lib/python3.9/site-packages/llama_cpp/server.py",
		},
	})
	pkgs = append(pkgs, coretypes.Package{
		Name: "app-pkg",
		Files: []string{
			"/app/main.py",
		},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := solver.resolveAppAnchors(tt.config, pkgs)
			sort.Strings(result)
			sort.Strings(tt.expected)

			if len(result) == 0 && len(tt.expected) == 0 {
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
