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

package fusion

import (
	"errors"
	"testing"
	"time"

	"gitee.com/openeuler/ktib/pkg/fusion/config"
	"gitee.com/openeuler/ktib/pkg/fusion/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

type MockSolver struct {
	mock.Mock
}

func (m *MockSolver) Solve(imageRef string, config *config.FusionConfig) (*types.FusionPlan, error) {
	args := m.Called(imageRef, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.FusionPlan), args.Error(1)
}

func (m *MockSolver) SetStepUpdater(f func(string)) {
	m.Called(f)
}

type MockFS struct {
	mock.Mock
}

func (m *MockFS) Synthesize(imageRef string, plan *types.FusionPlan, outputDir string) error {
	args := m.Called(imageRef, plan, outputDir)
	return args.Error(0)
}

func (m *MockFS) ExtractRPMDB(imageRef string, dest string) error {
	args := m.Called(imageRef, dest)
	return args.Error(0)
}

func (m *MockFS) ExtractFiles(imageRef string, files []string, outputDir string) error {
	args := m.Called(imageRef, files, outputDir)
	return args.Error(0)
}

type MockRPM struct {
	mock.Mock
}

func (m *MockRPM) Reconstruct(sourcePath string, plan *types.FusionPlan, outputDir string) error {
	args := m.Called(sourcePath, plan, outputDir)
	return args.Error(0)
}

type MockVerify struct {
	mock.Mock
}

func (m *MockVerify) Verify(rootfsPath string) error {
	args := m.Called(rootfsPath)
	return args.Error(0)
}

type MockCommit struct {
	mock.Mock
}

func (m *MockCommit) Commit(rootfs string, targetTag string, sourceImage string) error {
	args := m.Called(rootfs, targetTag, sourceImage)
	return args.Error(0)
}

func TestFusionManager_Run(t *testing.T) {
	testCases := []struct {
		name          string
		imageRef      string
		outputDir     string
		targetTag     string
		setupMocks    func(*MockSolver, *MockFS, *MockRPM, *MockVerify, *MockCommit)
		expectedError string
	}{
		{
			name:      "Success_FullFlow",
			imageRef:  "test-image",
			outputDir: "/tmp/output",
			targetTag: "new-image:latest",
			setupMocks: func(s *MockSolver, f *MockFS, r *MockRPM, v *MockVerify, c *MockCommit) {
				plan := &types.FusionPlan{KeptPackages: []string{"pkg1"}}
				s.On("SetStepUpdater", mock.Anything).Return()
				s.On("Solve", "test-image", mock.Anything).Return(plan, nil)
				f.On("Synthesize", "test-image", plan, "/tmp/output").Return(nil)
				f.On("ExtractRPMDB", "test-image", mock.AnythingOfType("string")).Return(nil)
				r.On("Reconstruct", mock.AnythingOfType("string"), plan, "/tmp/output").Return(nil)
				v.On("Verify", "/tmp/output").Return(nil)
				c.On("Commit", "/tmp/output", "new-image:latest", "test-image").Return(nil)
			},
			expectedError: "",
		},
		{
			name:      "Success_NoCommit",
			imageRef:  "test-image",
			outputDir: "/tmp/output",
			targetTag: "",
			setupMocks: func(s *MockSolver, f *MockFS, r *MockRPM, v *MockVerify, c *MockCommit) {
				plan := &types.FusionPlan{KeptPackages: []string{"pkg1"}}
				s.On("SetStepUpdater", mock.Anything).Return()
				s.On("Solve", "test-image", mock.Anything).Return(plan, nil)
				f.On("Synthesize", "test-image", plan, "/tmp/output").Return(nil)
				f.On("ExtractRPMDB", "test-image", mock.AnythingOfType("string")).Return(nil)
				r.On("Reconstruct", mock.AnythingOfType("string"), plan, "/tmp/output").Return(nil)
				v.On("Verify", "/tmp/output").Return(nil)
				// Commit should not be called
			},
			expectedError: "",
		},
		{
			name:      "Failure_Solve",
			imageRef:  "test-image",
			outputDir: "/tmp/output",
			targetTag: "new-image:latest",
			setupMocks: func(s *MockSolver, f *MockFS, r *MockRPM, v *MockVerify, c *MockCommit) {
				s.On("SetStepUpdater", mock.Anything).Return()
				s.On("Solve", "test-image", mock.Anything).Return(nil, errors.New("solve error"))
			},
			expectedError: "dependency solving failed: solve error",
		},
		{
			name:      "Failure_Synthesize",
			imageRef:  "test-image",
			outputDir: "/tmp/output",
			targetTag: "new-image:latest",
			setupMocks: func(s *MockSolver, f *MockFS, r *MockRPM, v *MockVerify, c *MockCommit) {
				plan := &types.FusionPlan{}
				s.On("SetStepUpdater", mock.Anything).Return()
				s.On("Solve", "test-image", mock.Anything).Return(plan, nil)
				f.On("Synthesize", "test-image", plan, "/tmp/output").Return(errors.New("synth error"))
			},
			expectedError: "filesystem synthesis failed: synth error",
		},
		{
			name:      "Failure_ExtractRPMDB",
			imageRef:  "test-image",
			outputDir: "/tmp/output",
			targetTag: "new-image:latest",
			setupMocks: func(s *MockSolver, f *MockFS, r *MockRPM, v *MockVerify, c *MockCommit) {
				plan := &types.FusionPlan{}
				s.On("SetStepUpdater", mock.Anything).Return()
				s.On("Solve", "test-image", mock.Anything).Return(plan, nil)
				f.On("Synthesize", "test-image", plan, "/tmp/output").Return(nil)
				f.On("ExtractRPMDB", "test-image", mock.AnythingOfType("string")).Return(errors.New("extract error"))
			},
			expectedError: "failed to extract RPM DB: extract error",
		},
		{
			name:      "Success_ReconstructFail_Continue",
			imageRef:  "test-image",
			outputDir: "/tmp/output",
			targetTag: "new-image:latest",
			setupMocks: func(s *MockSolver, f *MockFS, r *MockRPM, v *MockVerify, c *MockCommit) {
				plan := &types.FusionPlan{}
				s.On("SetStepUpdater", mock.Anything).Return()
				s.On("Solve", "test-image", mock.Anything).Return(plan, nil)
				f.On("Synthesize", "test-image", plan, "/tmp/output").Return(nil)
				f.On("ExtractRPMDB", "test-image", mock.AnythingOfType("string")).Return(nil)
				// Reconstruct fails, but flow should continue
				r.On("Reconstruct", mock.AnythingOfType("string"), plan, "/tmp/output").Return(errors.New("reconstruct error"))
				v.On("Verify", "/tmp/output").Return(nil)
				c.On("Commit", "/tmp/output", "new-image:latest", "test-image").Return(nil)
			},
			expectedError: "",
		},
		{
			name:      "Failure_Verify",
			imageRef:  "test-image",
			outputDir: "/tmp/output",
			targetTag: "new-image:latest",
			setupMocks: func(s *MockSolver, f *MockFS, r *MockRPM, v *MockVerify, c *MockCommit) {
				plan := &types.FusionPlan{}
				s.On("SetStepUpdater", mock.Anything).Return()
				s.On("Solve", "test-image", mock.Anything).Return(plan, nil)
				f.On("Synthesize", "test-image", plan, "/tmp/output").Return(nil)
				f.On("ExtractRPMDB", "test-image", mock.AnythingOfType("string")).Return(nil)
				r.On("Reconstruct", mock.AnythingOfType("string"), plan, "/tmp/output").Return(nil)
				v.On("Verify", "/tmp/output").Return(errors.New("verify error"))
			},
			expectedError: "verification failed: verify error",
		},
		{
			name:      "Failure_Commit",
			imageRef:  "test-image",
			outputDir: "/tmp/output",
			targetTag: "new-image:latest",
			setupMocks: func(s *MockSolver, f *MockFS, r *MockRPM, v *MockVerify, c *MockCommit) {
				plan := &types.FusionPlan{}
				s.On("SetStepUpdater", mock.Anything).Return()
				s.On("Solve", "test-image", mock.Anything).Return(plan, nil)
				f.On("Synthesize", "test-image", plan, "/tmp/output").Return(nil)
				f.On("ExtractRPMDB", "test-image", mock.AnythingOfType("string")).Return(nil)
				r.On("Reconstruct", mock.AnythingOfType("string"), plan, "/tmp/output").Return(nil)
				v.On("Verify", "/tmp/output").Return(nil)
				c.On("Commit", "/tmp/output", "new-image:latest", "test-image").Return(errors.New("commit error"))
			},
			expectedError: "commit failed: commit error",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockSolver := new(MockSolver)
			mockFS := new(MockFS)
			mockRPM := new(MockRPM)
			mockVerify := new(MockVerify)
			mockCommit := new(MockCommit)

			tt.setupMocks(mockSolver, mockFS, mockRPM, mockVerify, mockCommit)

			fm := &FusionManager{
				Config:     &config.FusionConfig{},
				Solver:     mockSolver,
				FS:         mockFS,
				RPM:        mockRPM,
				Verify:     mockVerify,
				Commit:     mockCommit,
				OnProgress: func(step string, done bool, duration time.Duration) {}, // Mock progress handler
			}

			err := fm.Run(tt.imageRef, tt.outputDir, tt.targetTag)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockSolver.AssertExpectations(t)
			mockFS.AssertExpectations(t)
			mockRPM.AssertExpectations(t)
			mockVerify.AssertExpectations(t)
			mockCommit.AssertExpectations(t)
		})
	}
}
