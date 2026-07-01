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

package project

import (
	"fmt"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/ktib/pkg/utils"
	"github.com/sirupsen/logrus"
)

const (
	DefaultProjectImageType = "platform"
	DefaultImageName        = "ktib-image"
	DefaultImageTag         = "latest"
	DefaultTimezone         = "Asia/Shanghai"
	DefaultLocale           = "C.UTF-8"
	DefaultConfigFileName   = "config.yml"
)

// ValidImageTypes returns the supported image types for project workflows.
func ValidImageTypes() []string {
	types := make([]string, len(utils.ValidImageTypes))
	copy(types, utils.ValidImageTypes)
	return types
}

// ProjectWorkflowRequest contains the shared inputs for project-related workflows.
type ProjectWorkflowRequest struct {
	ProjectDir string
	ImageType  string
	ConfigPath string
	ImageName  string
	Tag        string
	Init       bool
	Timezone   string
	Locale     string
}

type requestNormalizer func(ProjectWorkflowRequest) (ProjectWorkflowRequest, error)

// WorkflowService coordinates project workflows and keeps CLI handlers thin.
type WorkflowService struct {
	bootstrapFactory   func(string) *Bootstrap
	writeDefaultConfig func(string, string, string, string) error
	initProject        func(*Bootstrap) error
	buildRootfs        func(*Bootstrap, string) error
	cleanRootfs        func(*Bootstrap) error
	buildImage         func(*Bootstrap, string, string) error
}

// NewWorkflowService creates a workflow service with real project operations.
func NewWorkflowService() *WorkflowService {
	return &WorkflowService{
		bootstrapFactory:   NewBootstrap,
		writeDefaultConfig: WriteDefaultConfig,
		initProject: func(boot *Bootstrap) error {
			return boot.InitProjectStructure()
		},
		buildRootfs: func(boot *Bootstrap, configPath string) error {
			return boot.BuildRootfs(configPath)
		},
		cleanRootfs: func(boot *Bootstrap) error {
			return boot.CleanRootfs()
		},
		buildImage: func(boot *Bootstrap, imageName, tag string) error {
			return boot.BuildImage(imageName, tag)
		},
	}
}

// NormalizeInitRequest validates and normalizes an init workflow request.
func NormalizeInitRequest(req ProjectWorkflowRequest) (ProjectWorkflowRequest, error) {
	return normalizeCommonRequest(req)
}

// NormalizeBuildRootfsRequest validates and normalizes a build-rootfs request.
func NormalizeBuildRootfsRequest(req ProjectWorkflowRequest) (ProjectWorkflowRequest, error) {
	req, err := normalizeCommonRequest(req)
	if err != nil {
		return ProjectWorkflowRequest{}, err
	}
	if req.ConfigPath == "" {
		return ProjectWorkflowRequest{}, fmt.Errorf("when building rootfs, you need to specify the --config")
	}
	req.ConfigPath = filepath.Clean(req.ConfigPath)
	return req, nil
}

// NormalizeCleanRootfsRequest validates and normalizes a clean-rootfs request.
func NormalizeCleanRootfsRequest(req ProjectWorkflowRequest) (ProjectWorkflowRequest, error) {
	return normalizeCommonRequest(req)
}

// NormalizeBuildImageRequest validates and normalizes a build request.
func NormalizeBuildImageRequest(req ProjectWorkflowRequest) (ProjectWorkflowRequest, error) {
	req, err := normalizeCommonRequest(req)
	if err != nil {
		return ProjectWorkflowRequest{}, err
	}
	if req.ImageName == "" {
		req.ImageName = DefaultImageName
	}
	if req.Tag == "" {
		req.Tag = DefaultImageTag
	}
	return req, nil
}

// NormalizeMakeRequest validates and normalizes a make request.
func NormalizeMakeRequest(req ProjectWorkflowRequest) (ProjectWorkflowRequest, error) {
	req, err := NormalizeBuildImageRequest(req)
	if err != nil {
		return ProjectWorkflowRequest{}, err
	}
	if req.Timezone == "" {
		req.Timezone = DefaultTimezone
	}
	if req.Locale == "" {
		req.Locale = DefaultLocale
	}
	if req.Init && req.ConfigPath == "" {
		req.ConfigPath = filepath.Join(req.ProjectDir, DefaultConfigFileName)
	}
	if req.ConfigPath == "" {
		return ProjectWorkflowRequest{}, fmt.Errorf("when building rootfs, you need to specify the --config")
	}
	req.ConfigPath = filepath.Clean(req.ConfigPath)
	return req, nil
}

// InitProject runs the project initialization workflow.
func (s *WorkflowService) InitProject(req ProjectWorkflowRequest) error {
	return s.runWithBootstrap(req, NormalizeInitRequest, func(boot *Bootstrap, _ ProjectWorkflowRequest) error {
		return s.initProject(boot)
	})
}

// BuildRootfs runs the rootfs build workflow.
func (s *WorkflowService) BuildRootfs(req ProjectWorkflowRequest) error {
	return s.runWithBootstrap(req, NormalizeBuildRootfsRequest, func(boot *Bootstrap, normalized ProjectWorkflowRequest) error {
		return s.buildRootfs(boot, normalized.ConfigPath)
	})
}

// CleanRootfs runs the rootfs cleanup workflow.
func (s *WorkflowService) CleanRootfs(req ProjectWorkflowRequest) error {
	return s.runWithBootstrap(req, NormalizeCleanRootfsRequest, func(boot *Bootstrap, _ ProjectWorkflowRequest) error {
		return s.cleanRootfs(boot)
	})
}

// BuildImage runs the image build workflow.
func (s *WorkflowService) BuildImage(req ProjectWorkflowRequest) error {
	return s.runWithBootstrap(req, NormalizeBuildImageRequest, func(boot *Bootstrap, normalized ProjectWorkflowRequest) error {
		return s.buildImage(boot, normalized.ImageName, normalized.Tag)
	})
}

// MakeImage runs the complete image creation workflow.
func (s *WorkflowService) MakeImage(req ProjectWorkflowRequest) error {
	generateDefaultConfig := req.Init && strings.TrimSpace(req.ConfigPath) == ""
	req, err := NormalizeMakeRequest(req)
	if err != nil {
		return err
	}
	boot := s.newBootstrap(req)

	if req.Init {
		if err := s.initProject(boot); err != nil {
			return err
		}
		if generateDefaultConfig {
			if err := s.writeDefaultConfig(req.ConfigPath, req.Timezone, req.Locale, req.ImageType); err != nil {
				return err
			}
		}
	}

	logrus.Println("Building rootfs ...")
	if err := s.buildRootfs(boot, req.ConfigPath); err != nil {
		return err
	}

	logrus.Println("Cleaning rootfs ...")
	if err := s.cleanRootfs(boot); err != nil {
		return err
	}

	logrus.Println("Building image ...")
	if err := s.buildImage(boot, req.ImageName, req.Tag); err != nil {
		return err
	}

	logrus.Println("Make completed")
	return nil
}

func (s *WorkflowService) newBootstrap(req ProjectWorkflowRequest) *Bootstrap {
	boot := s.bootstrapFactory(req.ProjectDir)
	boot.BuildType = req.ImageType
	if req.Locale != "" {
		boot.Locale = req.Locale
	}
	return boot
}

func (s *WorkflowService) runWithBootstrap(req ProjectWorkflowRequest, normalize requestNormalizer, run func(*Bootstrap, ProjectWorkflowRequest) error) error {
	normalized, err := normalize(req)
	if err != nil {
		return err
	}
	return run(s.newBootstrap(normalized), normalized)
}

func normalizeCommonRequest(req ProjectWorkflowRequest) (ProjectWorkflowRequest, error) {
	req.ProjectDir = strings.TrimSpace(req.ProjectDir)
	req.ConfigPath = strings.TrimSpace(req.ConfigPath)
	req.ImageName = strings.TrimSpace(req.ImageName)
	req.Tag = strings.TrimSpace(req.Tag)
	req.Timezone = strings.TrimSpace(req.Timezone)
	req.Locale = strings.TrimSpace(req.Locale)

	if req.ProjectDir == "" {
		return ProjectWorkflowRequest{}, fmt.Errorf("project path is required")
	}

	imageType, err := normalizeImageType(req.ImageType)
	if err != nil {
		return ProjectWorkflowRequest{}, err
	}
	req.ImageType = imageType
	return req, nil
}

func normalizeImageType(imageType string) (string, error) {
	imageType = strings.TrimSpace(imageType)
	if imageType == "" {
		return DefaultProjectImageType, nil
	}
	if !utils.IsValidImageType(imageType) {
		return "", fmt.Errorf("invalid image type: %s. Valid types include: %s", imageType, strings.Join(utils.ValidImageTypes, ", "))
	}
	return imageType, nil
}
