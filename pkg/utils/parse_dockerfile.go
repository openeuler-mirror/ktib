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

package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ParseDockerfileFromImage parses the Dockerfile to get the repository addresses of the FROM images
// For example: cr.kylinos.cn/test/myapp:01, gets cr.kylinos.cn
func ParseDockerfileFromImage(dockerfilePath string) ([]string, error) {
	var repositories []string

	file, err := os.Open(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dockerfile %s: %w", dockerfilePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read dockerfile %s: %w", dockerfilePath, err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				imageName := parts[1]
				repository := parseImageRepository(imageName)
				if repository != "" {
					repositories = append(repositories, repository)
				}
			}
		}
	}

	return repositories, nil
}

// parseImageRepository extracts the repository address from the full image name
// Example: cr.kylinos.cn/test/myapp:01 -> cr.kylinos.cn
// Example: ubuntu:20.04 -> "" (no explicit repository address)
// Example: registry.io:5000/user/app@sha256:abc123 -> registry.io:5000
func parseImageRepository(imageName string) string {
	if imageName == "" {
		return ""
	}

	if idx := strings.Index(imageName, "@"); idx != -1 {
		imageName = imageName[:idx]
	}

	if idx := strings.LastIndex(imageName, ":"); idx != -1 {
		tagPart := imageName[idx+1:]
		if !strings.Contains(tagPart, "/") {
			imageName = imageName[:idx]
		}
	}

	if idx := strings.Index(imageName, "/"); idx != -1 {
		return imageName[:idx]
	}

	return ""
}
