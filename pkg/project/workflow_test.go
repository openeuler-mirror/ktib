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
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeMakeRequest(t *testing.T) {
	req, err := NormalizeMakeRequest(ProjectWorkflowRequest{
		ProjectDir: "/tmp/project",
		Init:       true,
	})
	if err != nil {
		t.Fatalf("NormalizeMakeRequest() error = %v", err)
	}

	if req.ImageType != DefaultProjectImageType {
		t.Fatalf("NormalizeMakeRequest().ImageType = %q, want %q", req.ImageType, DefaultProjectImageType)
	}
	if req.ImageName != DefaultImageName {
		t.Fatalf("NormalizeMakeRequest().ImageName = %q, want %q", req.ImageName, DefaultImageName)
	}
	if req.Tag != DefaultImageTag {
		t.Fatalf("NormalizeMakeRequest().Tag = %q, want %q", req.Tag, DefaultImageTag)
	}
	if req.ConfigPath != filepath.Join("/tmp/project", DefaultConfigFileName) {
		t.Fatalf("NormalizeMakeRequest().ConfigPath = %q, want %q", req.ConfigPath, filepath.Join("/tmp/project", DefaultConfigFileName))
	}
	if req.Timezone != DefaultTimezone {
		t.Fatalf("NormalizeMakeRequest().Timezone = %q, want %q", req.Timezone, DefaultTimezone)
	}
	if req.Locale != DefaultLocale {
		t.Fatalf("NormalizeMakeRequest().Locale = %q, want %q", req.Locale, DefaultLocale)
	}
}

func TestNormalizeBuildRootfsRequestRequiresConfig(t *testing.T) {
	_, err := NormalizeBuildRootfsRequest(ProjectWorkflowRequest{ProjectDir: "/tmp/project"})
	if err == nil {
		t.Fatal("NormalizeBuildRootfsRequest() error = nil, want error")
	}
}

func TestNormalizeInitRequestRejectsInvalidImageType(t *testing.T) {
	_, err := NormalizeInitRequest(ProjectWorkflowRequest{
		ProjectDir: "/tmp/project",
		ImageType:  "invalid",
	})
	if err == nil {
		t.Fatal("NormalizeInitRequest() error = nil, want error")
	}
}

func TestWriteDefaultConfigUsesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	if err := WriteDefaultConfig(configPath, "", "", ""); err != nil {
		t.Fatalf("WriteDefaultConfig() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "timezone: \""+DefaultTimezone+"\"") {
		t.Fatalf("WriteDefaultConfig() content missing default timezone: %s", content)
	}
	if !strings.Contains(content, "locale: \"%_install_langs "+DefaultLocale+"\"") {
		t.Fatalf("WriteDefaultConfig() content missing default locale: %s", content)
	}
	if !strings.Contains(content, "    - yum") {
		t.Fatalf("WriteDefaultConfig() content missing platform package list: %s", content)
	}
}

func TestWorkflowServiceMakeImageOrder(t *testing.T) {
	var order []string
	var capturedBootstrap *Bootstrap
	var capturedConfig string
	var capturedImageName string
	var capturedTag string

	service := &WorkflowService{
		bootstrapFactory: func(dir string) *Bootstrap {
			capturedBootstrap = &Bootstrap{DestinationDir: dir, BuildType: DefaultProjectImageType}
			return capturedBootstrap
		},
		writeDefaultConfig: func(path, timezone, locale, imageType string) error {
			order = append(order, "writeDefaultConfig")
			capturedConfig = path
			if timezone != DefaultTimezone {
				t.Fatalf("timezone = %q, want %q", timezone, DefaultTimezone)
			}
			if locale != DefaultLocale {
				t.Fatalf("locale = %q, want %q", locale, DefaultLocale)
			}
			if imageType != "minimal" {
				t.Fatalf("imageType = %q, want minimal", imageType)
			}
			return nil
		},
		initProject: func(boot *Bootstrap) error {
			order = append(order, "initProject")
			return nil
		},
		buildRootfs: func(boot *Bootstrap, configPath string) error {
			order = append(order, "buildRootfs")
			return nil
		},
		cleanRootfs: func(boot *Bootstrap) error {
			order = append(order, "cleanRootfs")
			return nil
		},
		buildImage: func(boot *Bootstrap, imageName, tag string) error {
			order = append(order, "buildImage")
			capturedImageName = imageName
			capturedTag = tag
			return nil
		},
	}

	err := service.MakeImage(ProjectWorkflowRequest{
		ProjectDir: "/tmp/project",
		ImageType:  "minimal",
		Init:       true,
	})
	if err != nil {
		t.Fatalf("MakeImage() error = %v", err)
	}

	wantOrder := []string{"initProject", "writeDefaultConfig", "buildRootfs", "cleanRootfs", "buildImage"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("order = %v, want %v", order, wantOrder)
	}
	if capturedBootstrap == nil {
		t.Fatal("bootstrap was not created")
	}
	if capturedBootstrap.BuildType != "minimal" {
		t.Fatalf("bootstrap.BuildType = %q, want minimal", capturedBootstrap.BuildType)
	}
	if capturedConfig != filepath.Join("/tmp/project", DefaultConfigFileName) {
		t.Fatalf("configPath = %q, want %q", capturedConfig, filepath.Join("/tmp/project", DefaultConfigFileName))
	}
	if capturedImageName != DefaultImageName {
		t.Fatalf("imageName = %q, want %q", capturedImageName, DefaultImageName)
	}
	if capturedTag != DefaultImageTag {
		t.Fatalf("tag = %q, want %q", capturedTag, DefaultImageTag)
	}
}

func TestWorkflowServiceMakeImageShortCircuitsOnBuildRootfsError(t *testing.T) {
	var order []string
	service := &WorkflowService{
		bootstrapFactory: func(dir string) *Bootstrap {
			return &Bootstrap{DestinationDir: dir, BuildType: DefaultProjectImageType}
		},
		writeDefaultConfig: func(path, timezone, locale, imageType string) error {
			order = append(order, "writeDefaultConfig")
			return nil
		},
		initProject: func(boot *Bootstrap) error {
			order = append(order, "initProject")
			return nil
		},
		buildRootfs: func(boot *Bootstrap, configPath string) error {
			order = append(order, "buildRootfs")
			return errors.New("build rootfs failed")
		},
		cleanRootfs: func(boot *Bootstrap) error {
			order = append(order, "cleanRootfs")
			return nil
		},
		buildImage: func(boot *Bootstrap, imageName, tag string) error {
			order = append(order, "buildImage")
			return nil
		},
	}

	err := service.MakeImage(ProjectWorkflowRequest{
		ProjectDir: "/tmp/project",
		Init:       true,
	})
	if err == nil {
		t.Fatal("MakeImage() error = nil, want error")
	}

	wantOrder := []string{"initProject", "writeDefaultConfig", "buildRootfs"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("order = %v, want %v", order, wantOrder)
	}
}

func TestWorkflowServiceMakeImageDoesNotRewriteExplicitConfig(t *testing.T) {
	var order []string
	service := &WorkflowService{
		bootstrapFactory: func(dir string) *Bootstrap {
			return &Bootstrap{DestinationDir: dir, BuildType: DefaultProjectImageType}
		},
		writeDefaultConfig: func(path, timezone, locale, imageType string) error {
			order = append(order, "writeDefaultConfig")
			return nil
		},
		initProject: func(boot *Bootstrap) error {
			order = append(order, "initProject")
			return nil
		},
		buildRootfs: func(boot *Bootstrap, configPath string) error {
			order = append(order, "buildRootfs")
			if configPath != "/tmp/custom.yml" {
				t.Fatalf("configPath = %q, want /tmp/custom.yml", configPath)
			}
			return nil
		},
		cleanRootfs: func(boot *Bootstrap) error {
			order = append(order, "cleanRootfs")
			return nil
		},
		buildImage: func(boot *Bootstrap, imageName, tag string) error {
			order = append(order, "buildImage")
			return nil
		},
	}

	err := service.MakeImage(ProjectWorkflowRequest{
		ProjectDir: "/tmp/project",
		Init:       true,
		ConfigPath: "/tmp/custom.yml",
	})
	if err != nil {
		t.Fatalf("MakeImage() error = %v", err)
	}

	wantOrder := []string{"initProject", "buildRootfs", "cleanRootfs", "buildImage"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("order = %v, want %v", order, wantOrder)
	}
}

func TestWorkflowServiceCleanRootfsUsesNormalizedState(t *testing.T) {
	var capturedBootstrap *Bootstrap
	service := &WorkflowService{
		bootstrapFactory: func(dir string) *Bootstrap {
			capturedBootstrap = &Bootstrap{DestinationDir: dir, BuildType: DefaultProjectImageType}
			return capturedBootstrap
		},
		writeDefaultConfig: WriteDefaultConfig,
		initProject: func(boot *Bootstrap) error {
			return nil
		},
		buildRootfs: func(boot *Bootstrap, configPath string) error {
			return nil
		},
		cleanRootfs: func(boot *Bootstrap) error {
			return nil
		},
		buildImage: func(boot *Bootstrap, imageName, tag string) error {
			return nil
		},
	}

	err := service.CleanRootfs(ProjectWorkflowRequest{
		ProjectDir: "/tmp/project",
		ImageType:  "init",
		Locale:     "zh_CN.UTF-8",
	})
	if err != nil {
		t.Fatalf("CleanRootfs() error = %v", err)
	}
	if capturedBootstrap == nil {
		t.Fatal("bootstrap was not created")
	}
	if capturedBootstrap.BuildType != "init" {
		t.Fatalf("bootstrap.BuildType = %q, want init", capturedBootstrap.BuildType)
	}
	if capturedBootstrap.Locale != "zh_CN.UTF-8" {
		t.Fatalf("bootstrap.Locale = %q, want zh_CN.UTF-8", capturedBootstrap.Locale)
	}
}
